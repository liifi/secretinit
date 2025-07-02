# Known Issues

## Current Issues

### Git Credential Helper Compatibility

#### WSL + Windows Git Credential Manager
- **Issue**: `secretinit` may hang when using Windows Git Credential Manager from WSL
- **Cause**: GUI prompts cannot be displayed in WSL environment
- **Impact**: Credential storage and retrieval of non-existent credentials
- **Workaround**: 
  - Run as Administrator: `SETX WSLENV "%WSLENV%:GIT_EXEC_PATH/wp"`
  - Restart WSL: `wsl --shutdown`
  - Avoid `credential.guiprompt false` config
- **Recommendation**: Use WSL for reading existing credentials only

#### Git Credential Helper Detection
- **Issue**: No validation that git credential helper is properly configured
- **Impact**: Silent failures when no credential helper is set up
- **Status**: Needs improvement in error messaging

### URL Parsing Edge Cases

#### Complex URL Formats
- **Issue**: Limited support for URLs with ports, paths, or query parameters
- **Example**: `https://user@api.example.com:8080/v1/secrets?param=value`
- **Current**: May not parse correctly
- **Status**: Needs enhancement

#### SSH URL Support
- **Issue**: SSH URLs like `git@example.com:service/path` not fully tested for general services
- **Status**: Basic support exists but needs validation for non-Git services

### Error Handling

#### Silent Failures
- **Issue**: Some credential retrieval failures may not provide clear error messages
- **Impact**: Difficult to debug credential issues
- **Status**: Needs improvement

#### Exit Code Propagation
- **Issue**: Some error scenarios may not properly propagate exit codes
- **Status**: Most cases handled, edge cases may exist

## Platform-Specific Issues

### macOS
- **Git Credential Manager**: Requires separate installation
- **Keychain Integration**: Works well with `osxkeychain` helper
- **Status**: Generally stable

### Linux
- **Credential Storage**: Limited to cache or file-based storage by default
- **Git Credential Manager**: Requires manual installation
- **Status**: Functional but limited secure storage options

### Windows
- **Git Credential Manager**: Usually included with Git for Windows
- **Windows Credential Manager**: Good integration with `wincred`
- **Status**: Generally stable

## Performance Issues

### Git Credential Command Overhead
- **Issue**: Each credential retrieval spawns a git process
- **Impact**: Performance overhead for many secrets
- **Status**: Acceptable for typical use cases, optimization possible

### Memory Usage
- **Issue**: All environment variables loaded into memory
- **Impact**: Large environments may use excessive memory
- **Status**: Not a problem for typical use cases

## Security Considerations

### Debug Logging
- **Issue**: Debug logs may expose sensitive information
- **Mitigation**: Password masking implemented
- **Status**: Partially addressed, needs review

### Credential Exposure
- **Issue**: Credentials exist in process environment
- **Impact**: May be visible to other processes
- **Status**: Inherent limitation of environment variable approach

### Error Messages
- **Issue**: Error messages might expose sensitive information
- **Status**: Needs audit

## Compatibility Issues

### Go Version Compatibility
- **Current**: Built with Go 1.24.4
- **Issue**: Compatibility with older Go versions not tested
- **Status**: Modern Go required

### Git Version Compatibility
- **Issue**: Requires git with credential helper support
- **Minimum**: Git 1.7.9+ (credential helpers introduced)
- **Status**: Most modern systems supported

## User Experience Issues

### Command Line Interface
- **Issue**: Limited argument validation and help text
- **Impact**: Poor user experience for invalid usage
- **Status**: Basic validation exists, needs improvement

### Configuration Management
- **Issue**: No configuration file support
- **Impact**: All configuration must be via command line or environment
- **Status**: Feature request

### Error Messages
- **Issue**: Some error messages are technical and not user-friendly
- **Status**: Needs improvement

## Testing Issues

### Integration Test Coverage
- **Issue**: Limited integration tests with real git credential helpers
- **Impact**: May miss real-world compatibility issues
- **Status**: Manual testing required

### Cross-Platform Testing
- **Issue**: Automated testing only on single platform
- **Impact**: Platform-specific issues may not be caught
- **Status**: Needs CI/CD improvement

### Mock Testing
- **Issue**: Limited mocking of external dependencies
- **Impact**: Tests depend on git installation
- **Status**: Acceptable for current scope

## Build and Release Issues

### Release Automation
- **Issue**: Release process is mostly manual
- **Status**: GoReleaser configured but needs CI integration

### Dependency Management
- **Issue**: No external dependencies currently, may change with new backends
- **Status**: Monitor as project grows
