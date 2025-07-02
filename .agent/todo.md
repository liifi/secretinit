# TODO

## High Priority

### Backend Implementation
- [x] **AWS Backend (Secrets Manager + Parameter Store)**
  - Implement `AWSBackend` struct
  - Add AWS SDK dependency
  - Support ARN and simple name formats
  - Handle JSON key extraction
  - ~~Add region configuration~~ (using AWS SDK default discovery)
  - Support both Secrets Manager (sm) and Parameter Store (ps) services

- [ ] **GCP Secret Manager Backend**
  - Implement `GCPBackend` struct
  - Add GCP SDK dependency
  - Support project/secret/version format
  - Handle JSON key extraction
  - Add authentication methods

- [ ] **Azure Key Vault Backend**
  - Implement `AzureBackend` struct
  - Add Azure SDK dependency
  - Support vault/secret format
  - Handle certificate and key extraction
  - Add authentication methods

### Error Handling Improvements
- [ ] **Better Error Messages**
  - User-friendly error descriptions
  - Actionable error messages
  - Context-specific help text
  - Debug information separation

- [ ] **Git Credential Helper Validation**
  - Check if git is installed
  - Verify credential helper is configured
  - Provide setup guidance on failures
  - Test credential helper functionality

### Command Line Interface
- [ ] **Improved Help System**
  - Better usage documentation
  - Examples in help text
  - Command-specific help
  - Error message improvements

- [ ] **Configuration File Support**
  - YAML/JSON config files
  - Environment variable mappings in config
  - Default backend configurations
  - Profile support (dev/staging/prod)

## Medium Priority

### User Experience
- [ ] **Cross-Reference Documentation**
  - Link between main README and credinit README
  - Consistent examples across documentation
  - Usage pattern documentation

- [ ] **Interactive Mode**
  - Interactive credential setup
  - Guided configuration wizard
  - Credential testing functionality

### Security Enhancements
- [ ] **Credential Exposure Audit**
  - Review debug logging for sensitive data
  - Audit error messages for credential leaks
  - Implement credential scrubbing utilities
  - Add security documentation

- [ ] **Process Security**
  - Investigate credential visibility to other processes
  - Consider memory protection techniques
  - Document security limitations

### Testing
- [ ] **Integration Test Suite**
  - Real git credential helper testing
  - Cross-platform compatibility tests
  - Performance benchmarking
  - Automated testing in CI/CD

- [ ] **Mock Testing Framework**
  - Mock git credential commands
  - Fake backend implementations
  - Test data management
  - Error scenario testing

## Low Priority

### Performance Optimizations
- [ ] **Credential Caching**
  - Cache retrieved credentials within process
  - Configurable cache duration
  - Cache invalidation strategies
  - Memory usage optimization

- [ ] **Parallel Processing**
  - Concurrent credential retrieval
  - Batch processing optimizations
  - Rate limiting for backends
  - Connection pooling

### Feature Extensions
- [ ] **Pre/Post Command Hooks**
  - Execute commands before main process
  - Cleanup commands after process exits
  - Environment setup scripts
  - Logging and monitoring hooks
  - Allow --pre and --post commands for executing other processes
  - Allow USR1 for quick restarts of program without pre and post?
  - Allow USR2 for quick restarts of program with pre and post?

- [ ] **Signal Handling Enhancement**
  - USR1 for credential refresh
  - USR2 for configuration reload
  - Graceful shutdown handling
  - Child process signal forwarding

- [ ] **Credential Rotation**
  - Automatic credential refresh
  - Rotation policy support
  - Credential lifecycle management
  - Integration with credential rotation services

### Advanced Features
- [x] **Single value retrieval**
  - `secretinit --stdout secretinit:....` command
  - Single credential retrieval no subprocess
  - Use case for when they one just needs to retrieve one value

- [ ] **Environment File Support**
  - Allow loading .env files containing environment variables (load by default)
  - Allow --no-env-file parameter to disable it
  - Allow --env-file parameter to define custom path if different

- [ ] **Environment Variable Mappings**
  - Allow SECRETINIT_MAPPINGS="" or via command with -m --mappings

- [ ] **Credential Validation**
  - Test credential connectivity
  - Validate credential format
  - Credential health checks
  - Monitoring integration

## Development Infrastructure

### Build and Release
- [ ] **CI/CD Pipeline**
  - Automated testing on multiple platforms
  - Cross-compilation for all platforms
  - Automated releases on tags
  - Security scanning integration

- [ ] **Release Management**
  - Semantic versioning automation
  - Release notes generation
  - Binary signing
  - Package repository publishing

### Code Quality
- [ ] **Linting and Formatting**
  - golangci-lint integration
  - Automated code formatting
  - Import organization
  - Documentation generation

- [ ] **Dependency Management**
  - Dependency security scanning
  - License compliance checking
  - Dependency update automation
  - Vendor directory management

## Documentation
- [ ] **API Documentation**
  - Package-level documentation
  - Function documentation with examples
  - Architecture decision records
  - Integration guides

- [ ] **User Guides**
  - Getting started tutorial
  - Platform-specific setup guides
  - Troubleshooting documentation
  - Best practices guide

- [ ] **Developer Documentation**
  - Contributing guidelines
  - Development environment setup
  - Testing guidelines
  - Release process documentation

## Research and Investigation
- [ ] **Alternative Backends**
  - HashiCorp Vault integration
  - Kubernetes secrets support
  - Docker secrets integration
  - Environment-specific backends

- [ ] **Security Research**
  - Process memory protection
  - Credential transmission security
  - Audit logging capabilities
  - Compliance framework support

- [ ] **Performance Research**
  - Memory usage profiling
  - CPU usage optimization
  - Network latency reduction
  - Concurrent processing patterns

## Migration and Compatibility
- [ ] **Integration Examples**
  - Docker Compose examples
  - Kubernetes deployment examples
  - CI/CD pipeline examples
  - Application framework integrations

## Monitoring and Observability
- [ ] **Metrics Collection**
  - Credential retrieval timing
  - Success/failure rates
  - Backend performance metrics
  - Usage analytics

- [ ] **Logging Enhancements**
  - Structured logging support
  - Log level configuration
  - Audit logging capabilities
  - Integration with log aggregation systems