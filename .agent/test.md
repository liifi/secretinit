# Test Guide

## Testing Strategy

### Unit Tests
- Test individual functions in isolation
- Use table-driven tests for multiple scenarios
- Mock external dependencies (git commands, file system)

### Integration Tests
- Test full workflows with real git credential operations
- Use test repositories and credentials
- Verify environment variable processing end-to-end

### Manual Testing
- Test with actual git credential helpers
- Verify cross-platform compatibility
- Test error scenarios and edge cases

## Test Data Patterns

### Standard Test URLs
Use `https://example.com` for consistency and to avoid hitting real services. **Note**: Test URLs should demonstrate general-purpose credential storage, not just Git repositories:

```bash
# Good test URLs - demonstrating various service types
https://example.com/api
https://user@example.com/service
https://database.example.com
https://api.service.com/v1

# Avoid real services in tests
# Bad: https://github.com/user/repo
# Bad: https://api.github.com
```

### Test Secret Addresses
```bash
# Git backend tests - various service types (not just Git repos)
secretinit:git:https://api.example.com:::password
secretinit:git:https://user@database.example.com:::username
secretinit:git:https://service.example.com/api:::token

# Git multi-credential mode (no keyPath)
secretinit:git:https://api.example.com
secretinit:git:https://database.example.com

# Future backend tests
secretinit:aws:sm:test-secret:::password
secretinit:gcp:sm:test-project/secret:::key
secretinit:azure:kv:test-vault/secret:::value
```

## Parser Testing

### Basic Parse Tests
```go
func TestParseSecretString(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected parser.SecretSource
        wantErr  bool
    }{
        {
            name:  "Git: Basic URL",
            input: "git:https://api.example.com",
            expected: parser.SecretSource{
                Backend:  "git",
                Service:  "",
                Resource: "https://api.example.com",
                KeyPath:  "",
            },
            wantErr: false,
        },
        {
            name:  "Git: With KeyPath",
            input: "git:https://api.example.com:::password",
            expected: parser.SecretSource{
                Backend:  "git",
                Service:  "",
                Resource: "https://api.example.com",
                KeyPath:  "password",
            },
            wantErr: false,
        },
    }
}
```

### Edge Case Tests
```go
{
    name:    "Git: User in URL",
    input:   "git:https://user@database.example.com:::username",
    expected: parser.SecretSource{
        Backend:  "git",
        Resource: "https://user@database.example.com",
        KeyPath:  "username",
    },
},
{
    name:    "Invalid: Missing Backend",
    input:   "https://example.com/secret",
    wantErr: true,
},
```

## Backend Testing

### Git Backend Tests
```go
func TestGitBackend_RetrieveSecret(t *testing.T) {
    tests := []struct {
        name     string
        resource string
        keyPath  string
        mockGit  func() (string, string, error)
        expected string
        wantErr  bool
    }{
        {
            name:     "Retrieve Password",
            resource: "https://api.example.com",
            keyPath:  "password",
            mockGit: func() (string, string, error) {
                return "testuser", "testpass", nil
            },
            expected: "testpass",
            wantErr:  false,
        },
        {
            name:     "Retrieve Username",
            resource: "https://api.example.com", 
            keyPath:  "username",
            mockGit: func() (string, string, error) {
                return "testuser", "testpass", nil
            },
            expected: "testuser",
            wantErr:  false,
        },
    }
}
```

### URL Parsing Tests
```go
func TestParseURLForUser(t *testing.T) {
    tests := []struct {
        name         string
        input        string
        expectedURL  string
        expectedUser string
    }{
        {
            name:         "Full URL with User",
            input:        "https://user@database.example.com",
            expectedURL:  "https://database.example.com",
            expectedUser: "user",
        },
        {
            name:         "Simple user@host",
            input:        "user@example.com",
            expectedURL:  "https://example.com",
            expectedUser: "user",
        },
        {
            name:         "No User",
            input:        "https://api.example.com",
            expectedURL:  "https://api.example.com",
            expectedUser: "",
        },
    }
}
```

## Environment Variable Testing

### Scanning Tests
```go
func TestScanSecretEnvVars(t *testing.T) {
    // Set test environment variables
    os.Setenv("TEST_SECRET", "secretinit:git:https://example.com/repo:::password")
    os.Setenv("NORMAL_VAR", "normal_value")
    defer os.Unsetenv("TEST_SECRET")
    defer os.Unsetenv("NORMAL_VAR")
    
    result := env.ScanSecretEnvVars()
    
    expected := map[string]string{
        "TEST_SECRET": "git:https://example.com/repo:::password",
    }
    
    if !reflect.DeepEqual(result, expected) {
        t.Errorf("got = %v, want %v", result, expected)
    }
}
```

### Mapping Tests
```go
func TestApplyMappings(t *testing.T) {
    tests := []struct {
        name     string
        env      map[string]string
        mappings string
        expected map[string]string
    }{
        {
            name: "Basic Mapping",
            env: map[string]string{
                "DB_USER": "admin",
                "DB_PASS": "secret",
            },
            mappings: "DB_USER->DATABASE_USERNAME,DB_PASS->DATABASE_PASSWORD",
            expected: map[string]string{
                "DB_USER":           "admin",
                "DB_PASS":           "secret", 
                "DATABASE_USERNAME": "admin",
                "DATABASE_PASSWORD": "secret",
            },
        },
    }
}
```

## Integration Testing

### Full Workflow Tests
```bash
#!/bin/bash
# integration_test.sh

# Setup test environment
export TEST_SECRET="secretinit:git:https://api.example.com:::password"

# Test secretinit
echo "Testing secretinit..."
SECRETINIT_LOG_LEVEL=DEBUG ./secretinit echo "Integration test passed"

# Test multi-credential mode  
echo "Testing multi-credential mode..."
SECRETINIT_LOG_LEVEL=DEBUG ./secretinit echo "Integration test passed"

# Test with mappings
echo "Testing with mappings..."
./secretinit -m "TEST_SECRET->API_TOKEN" env | grep API_TOKEN
```

### Git Credential Helper Testing
```bash
# Setup test credential helper (for CI/testing)
git config --global credential.helper 'cache --timeout=300'

# Store test credentials
echo "url=https://example.com
username=testuser
password=testpass" | git credential approve

# Test retrieval
echo "url=https://example.com" | git credential fill
```

## Manual Testing Scenarios

### Basic Functionality
```bash
# 1. Test environment variable scanning
export API_KEY="secretinit:git:https://example.com/api:::password"
./secretinit env | grep API_KEY

# 2. Test credential storage
./secretinit --store --url https://example.com --user testuser

# 3. Test mappings
export DB_PASS="secretinit:git:https://example.com/db:::password"
./secretinit -m "DB_PASS->DATABASE_PASSWORD" env | grep DATABASE_PASSWORD
```

### Error Scenarios
```bash
# 1. Test missing credentials
export MISSING_SECRET="secretinit:git:https://missing.example.com:::password"
./secretinit echo "Should fail"

# 2. Test invalid secret format
export INVALID_SECRET="secretinit:invalid:format"
./secretinit echo "Should fail"

# 3. Test invalid mappings
./secretinit -m "INVALID->MAPPING->FORMAT" echo "Should fail"
```

### Cross-Platform Testing
```bash
# Test on different platforms
# - macOS with osxkeychain
# - Linux with cache helper
# - Windows with wincred
# - WSL with Windows GCM

# Verify credential helpers work
git config --get credential.helper
```

## Test Data Management

### Test Credentials
```bash
# Use consistent test data
USERNAME: testuser
PASSWORD: testpass123
URL: https://example.com
```

### Cleanup Procedures
```bash
# Clean up test credentials
git credential reject <<EOF
url=https://example.com
username=testuser
EOF

# Clean up environment variables
unset TEST_SECRET
unset API_KEY
unset DB_PASS
```

## Performance Testing

### Load Testing
```bash
# Test with many environment variables
for i in {1..100}; do
    export "TEST_VAR_$i=secretinit:git:https://example.com/test$i:::password"
done

time ./secretinit echo "Load test complete"
```

### Memory Testing
```bash
# Monitor memory usage
valgrind --tool=memcheck ./secretinit echo "Memory test"

# Or use built-in Go tools
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

## Continuous Integration Testing

### GitHub Actions Example
```yaml
- name: Setup Git Credentials
  run: |
    git config --global credential.helper cache
    echo "url=https://example.com
    username=testuser  
    password=testpass" | git credential approve

- name: Run Integration Tests
  run: |
    export TEST_SECRET="secretinit:git:https://example.com:::password"
    ./secretinit echo "CI test passed"
```

### Test Coverage
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Target: >80% coverage for core packages
```

## Git Multi-Credential Testing

### Multi-Credential Mode Testing
```bash
# Test multi-credential loading (creates *_URL, *_USER, *_PASS)
export MYAPP="secretinit:git:https://api.example.com"
export TEST_SECRET="secretinit:git:https://api.example.com:::password"
```

### Expected Behavior
- **No keyPath**: Creates PREFIX_URL, PREFIX_USER, PREFIX_PASS variables (using exact variable name as prefix)
- **With keyPath**: Replaces variable with specific credential value
- **Variable naming**: MYAPP -> MYAPP_URL, MYAPP_USER, MYAPP_PASS

### Test Cases
```go
// Multi-credential mode
{
    input:   "MYAPP=secretinit:git:https://api.example.com",
    expected: {
        "MYAPP":      "secretinit:git:https://api.example.com", // Original preserved
        "MYAPP_URL":  "https://api.example.com",               // Clean URL
        "MYAPP_USER": "testuser",                              // From git
        "MYAPP_PASS": "testpass",                              // From git
    },
}

// Single credential mode  
{
    input:   "API_TOKEN=secretinit:git:https://api.example.com:::password",
    expected: {
        "API_TOKEN": "testpass",
    },
}
```
