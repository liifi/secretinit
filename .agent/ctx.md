# Functional Context

## Project Overview

**secretinit** is a credential injection system that enables secure secret management for applications by intercepting environment variables with special prefixes and replacing them with actual secret values retrieved from various backends.

## Core Components

### 1. Main Tools

#### `secretinit` (Main Tool)
- **Purpose**: Universal secret injection wrapper for any application
- **Backends**: Supports multiple backends (git, aws, gcp, azure)
- **Pattern**: `secretinit:backend:service:resource:::keyPath`
- **Example**: `export DB_PASS="secretinit:git:https://api.example.com:::password"`

#### `credinit` (General-Purpose Credential Tool)
- **Purpose**: Universal credential management leveraging Git's credential helper system as general-purpose secure storage for any URL-based service
- **Backends**: Git credential helpers (leverages OS-native secure storage for any URL, not just Git repositories)
- **Pattern**: `secretinit:git:url[:::keyPath]`
- **Modes**: Multi-credential (creates *_URL, *_USER, *_PASS) or single credential (with keyPath)
- **Example**: `export API_KEY="secretinit:git:https://api.example.com:::password"`

### 2. Package Architecture

#### `pkg/env`
- **Function**: Environment variable scanning
- **Pattern**: Finds variables with `secretinit:` prefix
- **Returns**: Map of variable names to secret addresses

#### `pkg/parser`
- **Function**: Secret address string parsing
- **Input**: `"aws:sm:my-secret:::username"`
- **Output**: `SecretSource{Backend: "aws", Service: "sm", Resource: "my-secret", KeyPath: "username"}`

#### `pkg/backend`
- **Function**: Backend implementations for secret retrieval
- **Interface**: `RetrieveSecret(resource, keyPath string) (string, error)`
- **Implementations**: `GitBackend` (with AWS, GCP, Azure planned)

#### `pkg/processor`
- **Function**: Orchestrates secret processing workflow
- **Pattern**: Register backends → Process secrets → Return resolved values
- **Usage**: Both tools use this for consistent processing

#### `pkg/mappings`
- **Function**: Environment variable transformations
- **Pattern**: `"SOURCE->TARGET,SOURCE2->TARGET2"`
- **Example**: `"DB_USER->DATABASE_USERNAME,DB_PASS->DATABASE_PASSWORD"`

### 3. Workflow Patterns

#### Secret Resolution Process
1. **Scan**: Find environment variables with `secretinit:` prefix
2. **Parse**: Extract backend, service, resource, and keyPath
3. **Route**: Send to appropriate backend implementation
4. **Retrieve**: Get actual secret value from backend
5. **Map**: Apply any specified environment variable mappings
6. **Execute**: Run target application with enhanced environment

#### Git Backend Workflow
1. **Parse URL**: Extract clean URL and username from resource
2. **Call Git Credential Helper**: Use `git credential fill` to retrieve credentials from configured storage
3. **Extract**: Parse username/password from git credential helper output
4. **Return**: Provide requested credential component (username or password)

**Note**: The git backend leverages Git's credential helper system as a **general-purpose, cross-platform credential storage mechanism for any URL-based service**. This is not limited to git repositories - credentials for APIs, databases, web services, SaaS platforms, or any other URL-accessible service can be stored and retrieved using the OS-native secure storage that git credential helpers provide (Keychain on macOS, Credential Manager on Windows, etc.).

## Configuration Patterns

### Environment Variable Patterns

#### Standard Pattern
```bash
export VAR_NAME="secretinit:backend:service:resource:::keyPath"
```

#### Git Backend Examples
```bash
# secretinit: Single credential replacement
export API_TOKEN="secretinit:git:https://api.example.com:::password"
export API_USER="secretinit:git:https://api.example.com:::username"

# credinit: Multi-credential mode (creates *_URL, *_USER, *_PASS)
export API="secretinit:git:https://api.example.com"
export DATABASE="secretinit:git:https://database.example.com"

# credinit: Single credential mode (with keyPath)
export TOKEN="secretinit:git:https://api.example.com:::password"
```

#### Future Backend Examples
```bash
export DB_PASS="secretinit:aws:sm:myapp/db-creds:::password"
export API_KEY="secretinit:gcp:sm:myproject/api-key"
export CERT="secretinit:azure:kv:myvault/ssl-cert:::certificate"
```

### Command Line Patterns

#### Basic Usage
```bash
secretinit myapp
credinit myapp
```

#### With Mappings
```bash
secretinit -m "DB_USER->DATABASE_USERNAME" myapp
credinit --mappings "API_TOKEN->GITHUB_TOKEN" build-script
```

#### Credential Storage (credinit only)
```bash
credinit --store --url https://example.com --user myuser
```

## Security Model

### Git Credential Integration
- **Storage**: Leverages Git's credential helper system as general-purpose secure storage for any URL-based service
- **Security**: Inherits security properties of configured git credential helper
- **Cross-Platform**: Works with platform-specific secure storage (Keychain, Credential Manager, etc.)
- **Universal**: Not limited to Git repositories - works with APIs, databases, web services, etc.

### Secret Isolation
- **Process Isolation**: Secrets only exist in target process environment
- **No Persistence**: Resolved secrets are not written to disk
- **Minimal Exposure**: Only requested keyPath components are extracted

### Error Handling
- **Fail Fast**: Exit immediately on credential retrieval failures
- **No Fallbacks**: Don't continue with empty/default credentials
- **Debugging**: Debug logging available via `CREDINIT_LOG_LEVEL=DEBUG`

## Extensibility Points

### Adding New Backends
1. Implement `backend.Backend` interface
2. Register backend in processor
3. Add parsing support for backend-specific patterns
4. Update documentation and examples

### Command Line Extensions
- Pre/post command execution hooks
- Configuration file support
- Additional credential storage options
- Signal handling enhancements

## Use Cases

### Development Environments
- Local development with production-like credentials
- CI/CD pipeline secret injection
- Testing with real credentials

### Production Deployments
- Container credential injection
- Serverless function credential management
- Multi-service credential coordination

### General-Purpose Credential Storage Scenarios
- **API token management**: Store API tokens for various services (GitHub, Slack, databases, etc.) using git credential helpers
- **Cross-platform credential sharing**: Leverage OS-native secure storage through git for any URL-based service
- **Multi-credential injection**: Automatically create *_URL, *_USER, and *_PASS variables for applications
- **Universal credential storage**: Use git credential helpers as a cross-platform credential store for APIs, databases, web services, SaaS platforms, or any URL-accessible service
