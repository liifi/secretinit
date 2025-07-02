package exec

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// ExecuteCommand executes the given command with the provided environment variables.
// It includes proper signal handling to forward signals to the child process.
func ExecuteCommand(args []string, env []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No command provided to execute.")
		os.Exit(1)
	}

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
		os.Exit(1)
	}

	go func() {
		sig := <-sigChan
		if cmd.Process != nil {
			// Forward the signal to the child process
			cmd.Process.Signal(sig)
		}
	}()

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(1)
	}
}

// ExecuteCommandWithDebug is like ExecuteCommand but includes debug logging
func ExecuteCommandWithDebug(args []string, env []string, debugLog func(string, ...interface{})) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No command provided to execute.")
		os.Exit(1)
	}

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
		os.Exit(1)
	}
	if debugLog != nil {
		debugLog("Started process with PID: %d", cmd.Process.Pid)
	}

	go func() {
		sig := <-sigChan
		if cmd.Process != nil {
			// Forward the signal to the child process
			cmd.Process.Signal(sig)
		}
	}()

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(1)
	}
}
