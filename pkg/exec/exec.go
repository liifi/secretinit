package exec

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

// parseCommand parses a command string into executable and arguments
// This provides basic shell-like parsing without the security risks of using a shell
func parseCommand(cmdStr string) (string, []string) {
	if cmdStr == "" {
		return "", nil
	}

	// Simple parsing: split on spaces, but respect quoted strings
	var args []string
	var current strings.Builder
	inQuotes := false
	var quoteChar rune

	for i, r := range cmdStr {
		switch {
		case !inQuotes && (r == '"' || r == '\''):
			inQuotes = true
			quoteChar = r
		case inQuotes && r == quoteChar:
			inQuotes = false
		case !inQuotes && r == ' ':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		case r == '\\' && i+1 < len(cmdStr):
			// Handle simple escape sequences
			next := rune(cmdStr[i+1])
			if next == '"' || next == '\'' || next == '\\' || next == ' ' {
				current.WriteRune(next)
				i++ // Skip the next character
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if len(args) == 0 {
		return "", nil
	}

	return args[0], args[1:]
}

// ExecuteCommandWithHooks executes the given command with optional pre/post commands.
// It includes proper signal handling and ensures post commands run even if main command fails.
func ExecuteCommandWithHooks(args []string, env []string, preCommand, postCommand string, debugLog func(string, ...interface{}), infoLog func(string, ...interface{})) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No command provided to execute.")
		os.Exit(1)
	}

	// Execute pre-command if specified
	if preCommand != "" {
		debugLog("Executing pre-command: %s", preCommand)
		infoLog("[PRE] Running: %s", preCommand)
		exitCode, err := executeCommand(preCommand, env, debugLog)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[PRE] Command failed with exit code %d: %v\n", exitCode, err)
			os.Exit(exitCode)
		}
		infoLog("[PRE] Completed successfully")
	}

	// Track exit code for proper cleanup
	var exitCode int

	// Ensure post-command runs even if main command fails
	defer func() {
		if postCommand != "" {
			debugLog("Executing post-command: %s", postCommand)
			infoLog("[POST] Running: %s", postCommand)
			postExitCode, err := executeCommand(postCommand, env, debugLog)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[POST] Command failed with exit code %d: %v\n", postExitCode, err)
				// Don't exit here - we want to preserve the main command's exit code
			} else {
				infoLog("[POST] Completed successfully")
			}
		}
		// Exit with the recorded exit code after post-command completes
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}()

	// Execute main command
	infoLog("[MAIN] Running: %s%s", args[0], func() string {
		if len(args) > 1 {
			return " " + strings.Join(args[1:], " ")
		}
		return ""
	}())

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start command: %v\n", err)
		exitCode = 1
		return
	}
	debugLog("Started main process with PID: %d", cmd.Process.Pid)

	go func() {
		sig := <-sigChan
		if cmd.Process != nil {
			// Forward the signal to the child process
			cmd.Process.Signal(sig)
		}
	}()

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
			infoLog("[MAIN] Command exited with code: %d", exitCode)
		} else {
			exitCode = 1
			infoLog("[MAIN] Command failed: %v", err)
		}
	} else {
		infoLog("[MAIN] Completed successfully")
	}
}

// executeCommand executes a command string by parsing it directly (no shell)
// Returns the exit code and error for better error reporting
func executeCommand(cmdStr string, env []string, debugLog func(string, ...interface{})) (int, error) {
	executable, args := parseCommand(cmdStr)
	if executable == "" {
		return 1, fmt.Errorf("empty command")
	}

	debugLog("Executing command: %s with args: %v", executable, args)

	cmd := exec.Command(executable, args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), err
		}
		return 1, err
	}
	return 0, nil
}
