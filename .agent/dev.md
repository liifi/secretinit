# Development Guide

## Architecture Patterns

### Package Organization

```
pkg/
├── env/          # Environment variable scanning
├── parser/       # Secret address parsing  
├── backend/      # Backend implementations
├── processor/    # Secret processing orchestration (unified)
└── mappings/     # Environment variable transformations

cmd/
└── secretinit/   # Universal secret injection tool
```

### Interface Design Pattern

#### Backend Interface
```go
type Backend interface {
    RetrieveSecret(resource, keyPath string) (string, error)
}
```

**Rules:**
- Simple interface with single responsibility
- Resource string is backend-specific (URL for git, ARN for AWS, etc.)
- KeyPath specifies which part of credential to return
- Error handling is explicit

#### Processor Pattern
```go
type SecretProcessor struct {
    backends map[string]Backend
}

func (p *SecretProcessor) RegisterBackend(name string, backend Backend)
func (p *SecretProcessor) ProcessSecrets(secretVars map[string]string) (map[string]string, error)
```

**Rules:**
- Registry pattern for backend management
- Consistent processing workflow
- Error aggregation and reporting

### Error Handling Patterns

#### Fail Fast Pattern
```go
if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```

**When to use:**
- Credential retrieval failures
- Invalid secret address formats
- Missing required parameters

#### Graceful Degradation Pattern
```go
if creds := getCredential(url, user); creds != nil {
    // Use credentials
} else {
    debugLog("Failed to get credentials, continuing...")
    // Continue without credentials
}
```

**When to use:**
- Optional credentials
- Development/testing scenarios
- Non-critical secret retrieval

### Debugging Patterns

#### Conditional Debug Logging
```go
var debugEnabled = os.Getenv("SECRETINIT_LOG_LEVEL") == "DEBUG"

func debugLog(format string, args ...interface{}) {
    if debugEnabled {
        fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
    }
}
```

#### Sensitive Data Masking
```go
maskedOutput := string(output)
lines := strings.Split(maskedOutput, "\n")
for i, line := range lines {
    if strings.HasPrefix(line, "password=") {
        lines[i] = "password=***"
    }
}
```

### Code Sharing Patterns

#### Shared Functionality
- **DO**: Place common logic in packages (`pkg/processor`, `pkg/mappings`)
- **DON'T**: Duplicate parsing or credential logic between tools

#### Tool-Specific Logic
- **DO**: Keep tool-specific CLI handling in `cmd/` directories
- **DON'T**: Mix CLI logic with core functionality

#### Function Visibility
```go
// Public functions for cross-package use
func ParseURLForUser(rawURL string) (string, string)

// Private functions for internal use
func parseURLForUser(rawURL string) (string, string)
```

## Testing Patterns

### Unit Test Structure
```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected ExpectedType
        wantErr  bool
    }{
        {
            name:     "Description of test case",
            input:    "test input",
            expected: ExpectedType{},
            wantErr:  false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.expected) {
                t.Errorf("got = %v, want %v", got, tt.expected)
            }
        })
    }
}
```

### Integration Test Patterns
- Use `https://example.com` for test URLs
- Mock git credential responses where possible
- Test full workflow with fake backends

## Git Integration Patterns

### URL Parsing
```go
func parseURLForUser(rawURL string) (string, string) {
    // Handle https://user@example.com/repo
    if strings.Contains(rawURL, "@") && strings.Contains(rawURL, "://") {
        // Extract scheme and parse user@host
    }
    // Handle simple user@host format
    return cleanURL, username
}
```

### Git Credential Commands
```go
input := fmt.Sprintf("url=%s\n", url)
if user != "" {
    input += fmt.Sprintf("username=%s\n", user)
}
input += "\n" // Blank line terminates input

cmd := exec.Command("git", "credential", "fill")
cmd.Stdin = strings.NewReader(input)
```

## Command Line Patterns

### Argument Parsing
```go
func ParseMappingsFromArgs(args []string) (map[string]string, int) {
    // Parse --mappings/-m flags
    // Return mappings and command start index
}
```

### Environment Variable Handling
```go
// Start with current environment
newEnv := os.Environ()

// Add resolved secrets
for key, value := range secrets {
    newEnv = append(newEnv, fmt.Sprintf("%s=%s", key, value))
}

// Apply mappings
finalEnv := mappings.ApplyMappingsToEnv(newEnv, mappingMap)
```

### Process Execution
```go
cmd := exec.Command(args[0], args[1:]...)
cmd.Env = enhancedEnv
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Stdin = os.Stdin

// Handle exit codes properly
if err := cmd.Run(); err != nil {
    if exitError, ok := err.(*exec.ExitError); ok {
        os.Exit(exitError.ExitCode())
    }
    os.Exit(1)
}
```

## Extension Patterns

### Adding New Backends

1. **Create Backend Implementation**
```go
type AWSBackend struct{}

func (b *AWSBackend) RetrieveSecret(resource, keyPath string) (string, error) {
    // AWS-specific implementation
}
```

2. **Register Backend**
```go
proc.RegisterBackend("aws", &backend.AWSBackend{})
```

3. **Update Parser (if needed)**
```go
case "aws":
    // AWS-specific parsing logic
```

### Adding Command Line Features

1. **Extend Argument Parsing**
```go
if arg == "--new-flag" {
    // Handle new flag
}
```

2. **Add to Usage Documentation**
3. **Update Integration Points**

## Git Multi-Credential Processing

### Purpose
The unified processor now handles git multi-credential logic automatically:

### Key Features
- **Multi-credential mode**: When no keyPath is specified for git backend, creates three environment variables
- **Auto-detection**: Automatically detects git secrets without keyPath
- **Variable naming**: Uses exact variable name as prefix (no suffix extraction)
- **Dual behavior**: Can function in single credential mode when keyPath is provided

### Implementation
```go
// Multi-credential mode: MYAPP=secretinit:git:https://api.example.com
// Creates: MYAPP (original), MYAPP_URL (clean URL), MYAPP_USER, MYAPP_PASS

// Single credential mode: API_TOKEN=secretinit:git:https://api.example.com:::password  
// Creates: API_TOKEN=<password_value>
```

### Variable Creation Logic
1. **Multi-credential mode** (no keyPath):
   - Keep original variable with `secretinit:` prefix
   - Create `{VAR}_URL` with clean URL (username removed)
   - Create `{VAR}_USER` with retrieved username
   - Create `{VAR}_PASS` with retrieved password

2. **Single credential mode** (with keyPath):
   - Replace original variable with retrieved credential value
