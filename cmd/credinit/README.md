# CredInit

A cross-platform Go application that manages credentials via Git's credential helper system and acts as a process manager. **CredInit leverages Git's credential helpers as universal, secure credential storage for any URL-based service** - not just Git repositories, but APIs, databases, web services, or any other URL-accessible resource.

> **Key Point**: Despite the name "Git credential helper", this system works as general-purpose credential storage for **any URL-based service**. The "Git" part refers to the underlying storage mechanism, not the intended use case.

## Purpose

CredInit uses Git's credential helper system as a general-purpose, cross-platform credential storage mechanism for any URL-based service. This allows you to store and retrieve credentials for APIs, databases, web services, SaaS platforms, or any other service accessed via URL using your operating system's native secure storage:

- **macOS**: Keychain integration via `osxkeychain` helper
- **Windows**: Credential Manager via `wincred` or Git Credential Manager
- **Linux**: Various options including Git Credential Manager or cache

This approach provides a consistent interface for credential management across platforms without requiring additional credential storage systems. **The key benefit is that any URL-based service can use this secure storage, regardless of whether it's related to Git or not.**

## Usage

### Running a command with credential injection
```bash
# Multi-credential mode: Creates MYAPP_URL, MYAPP_USER, and MYAPP_PASS
export MYAPP=secretinit:git:https://api.example.com
credinit myapp arg1 arg2

# Single credential mode: Replace with specific value (password or username)
export API_TOKEN=secretinit:git:https://api.example.com:::password
credinit myapp arg1 arg2
```

### Using environment variable mappings
```bash
# Multi-credential mode with mappings
export MYAPP=secretinit:git:https://api.example.com
credinit --mappings "MYAPP_USER->DB_USERNAME,MYAPP_PASS->DB_PASSWORD" myapp arg1 arg2

# Short form
credinit -m "MYAPP_USER->DB_USERNAME,MYAPP_PASS->DB_PASSWORD" myapp arg1 arg2
```

### Storing credentials
```bash
# Store credentials for any URL-based service (APIs, databases, etc.)
credinit --store --url https://api.example.com --user myuser
credinit --store --url https://database.company.com --user dbuser
credinit --store --url https://internal-service.corp.com --user serviceuser

# Or let it prompt for missing values
credinit --store
```

### How it works

**Multi-credential mode** (no keyPath specified):
1. Scans environment variables for `*=secretinit:git:URL` patterns 
2. Uses Git's credential helper system to retrieve stored credentials for matching URLs
3. Creates three environment variables: `*_URL`, `*_USER`, and `*_PASS` using the base prefix
4. Applies any specified environment variable mappings
5. Executes the target command with enhanced environment
6. Forwards signals to the child process

**Single credential mode** (keyPath specified):
1. Scans environment variables for `*=secretinit:git:URL:::keyPath` patterns
2. Retrieves only the specified credential component (username or password)
3. Replaces the original environment variable value with the retrieved credential
4. Behaves like secretinit for single credential replacement

## Credential Store Configuration

CredInit uses Git's credential helper system as general-purpose secure storage. You should configure a secure credential helper for your platform to store credentials for any URL-based service:

### macOS
```bash
# Option 1: Git Credential Manager (recommended, download git-credential-manager)
git config --global credential.helper manager

# Option 2: Use the osxkeychain
git config --global credential.helper osxkeychain
```

### Linux
```bash
# Option 1: Git Credential Manager (recommended, download git-credential-manager)
git config --global credential.helper manager

# Option 2: Use in memory cache, or read about other git credential stores
git config --global credential.helper 'cache --timeout=<seconds>'
```

### Windows
```bash
# Option 1: Git Credential Manager (recommended, included in windows)
git config --global credential.helper manager

# Option 2: Windows Credential Manager
git config --global credential.helper wincred
```

### WSL
```bash
# Option 1: Git Credential Manager from Windows
git config --global credential.helper /mnt/c/git/install/path/mingw64/bin/git-credential-manager.exe

# IMPORTANT: WSL may cause credinit to hang when using Windows' GCM.
#            No issue for loading existing credentials, but for non
#            existent credential or for storing, it must be from GUI.
#            To allow it to work from WSL, one must run as Administrator
#            SETX WSLENV "%WSLENV%:GIT_EXEC_PATH/wp"
#            wsl --shutdown
#            In wsl git config, do not use 'credential.guiprompt false'
#            If credinit hangs, it is due to either the config or WSLENV
#            Ideally only read existing values from WSL.
```

### Configs
```bash
git config --global credential.helper manager
git config --global credential.githubauthmodes pat
# Do not use the following in WSL if using windows git-credential-manager.exe
git config --global credential.guiprompt false
```

## Other Options
- **git-credential-store** (avoid - stores in plain text)
- **Third-party tools**: gopass, lastpass, 1password, keepassxc
- See: https://git-scm.com/doc/credential-helpers

## Important Notes

- Only environment variables with `secretinit:git:` prefix are processed for credential loading
- Regular environment variables without this prefix are ignored for credential processing  
- **Multi-credential mode**: When no keyPath is specified, creates three variables (*_URL, *_USER, *_PASS)
- **Single credential mode**: When keyPath is specified (:::password or :::username), behaves like secretinit
- **Credential storage works for any URL-based service** - not limited to Git repositories
- Credentials are stored directly in Git's credential helper system (no prefix needed for storage)
- Mappings use arrow syntax: `SOURCE->TARGET,SOURCE2->TARGET2`
- Configure a secure credential helper before storing credentials

## Building

```bash
# Build for current platform
go build -o credinit
```

## Releasing from local

> TODO: Release from CI

```bash
# goreleaser
goreleaser build --snapshot --clean # test build locally
./dist/main_windows_amd64_v1/credinit.exe --version # test a build

# Add a tag
git add .
git commit -m "Some message"
git tag v1.0.0
git push origin main
git push origina v1.0.0

# Expose GITHUB_TOKEN


# For testing the release process without publishing
goreleaser release --snapshot --clean

# or for a real release (requires git tag)
goreleaser release --clean
```