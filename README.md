### What is this?

**`secretinit`** is a command-line tool designed to streamline how applications access and use secrets like API keys, database credentials, or sensitive configuration values. It provides a consistent way to load these into your application's environment variables.

> **Important Note**: The "git" backend uses Git's credential helper system as **general-purpose credential storage for any URL-based service** - not just Git repositories. This means you can securely store and retrieve credentials for APIs, databases, web services, SaaS platforms, or any other service accessed via URL, using your operating system's native secure storage (Keychain on macOS, Credential Manager on Windows, etc.).

The core idea is to replace sensitive values in your environment (or a configuration file that `secretinit` reads) with a small, readable string that tells `secretinit` where the real secret lives.

-----

### How It Works

You run `secretinit` **in front of your actual application**. `secretinit` will read the environment variables you've set, fetch the actual secret values, and then launch your program with a *modified environment* where the original "secret address strings" have been replaced by the real secrets.

Here's a quick example of how it works:

Let's say in your current shell, you set an environment variable pointing to a secret in AWS Secrets Manager:

```bash
export MY_DB_PASSWORD="aws:sm:my-app/db-creds:::password"
```

Now, if you try to `echo` this variable directly, you'll see the secret address string:

```bash
echo $MY_DB_PASSWORD
# Expected output: aws:sm:my-app/db-creds:::password
```

But when you run `secretinit` followed by another command (like `bash -c 'echo $MY_DB_PASSWORD'`), `secretinit` intercepts, fetches the actual password, and launches the `bash` command with the real value:

```bash
secretinit bash -c 'echo $MY_DB_PASSWORD'
# Expected output (assuming the secret is "supersecurepass"): supersecurepass
```

Your `bash` command (which represents your application) sees only the real password, not the secret address string.

-----

The "secret address string" follows a pattern: `backend:service:resource[:::key_path]`.

  * **`backend`**: This tells `secretinit` the type of secret storage system. For example, `git` for credentials managed by your Git CLI, `aws` for Amazon Web Services, `gcp` for Google Cloud Platform, or `azure` for Microsoft Azure.
  * **`service`**: For cloud providers, this specifies the exact service within that cloud. So, for AWS, `sm` would mean Secrets Manager, and `ps` would mean Parameter Store. In Azure, `kv` would indicate Key Vault.
  * **`resource`**: This is the actual name or identifier of the secret within that service.
      * For **Git**, this means `secretinit` will query your **local Git credential helper system** for stored credentials. The resource part would be the **URL** (e.g., `https://api.example.com`, `https://myservice.com/api`, or `https://database.company.com`) for which you want to retrieve credentials. **Important**: This leverages Git's credential helper system as general-purpose, cross-platform secure storage - it's not limited to git repositories. You can store credentials for any URL-based service (APIs, databases, web services, etc.) using your OS's native credential storage (Keychain on macOS, Credential Manager on Windows, etc.).
      * For **AWS**, it could be a simple name like `my-app/db-creds` or a full Amazon Resource Name (ARN) such as `arn:aws:secretsmanager:...`. `secretinit` is designed to correctly parse these, even though ARNs contain colons themselves.
      * For **GCP** and **Azure**, it would be the name or path that identifies your secret within their respective management services.
  * **`key_path` (Optional)**: If your secret is a structured piece of data (like a JSON document), this lets you retrieve a specific part (e.g., `username`). For **Git**, this lets you specify if you need the `username` or the `password` for the provided URL.

This design makes your application code cleaner, more portable, and significantly more secure by centralizing secret management and injection.

-----

## Tool Overview

**`secretinit`** is a universal secret injection tool that supports multiple backend types and provides a unified interface for secret injection across different cloud providers and systems. It supports both single credential mode and git multi-credential expansion.

### Key Features

- **Multiple Backends**: Supports git credential helpers, AWS Secrets Manager, AWS Parameter Store (with more planned)
- **Git Multi-Credential Mode**: Automatically creates `*_URL`, `*_USER`, and `*_PASS` variables when no keyPath is specified for git backend
- **Single Credential Mode**: Direct replacement for specific credential components
- **Environment Variable Mappings**: Transform secret variable names using `-m/--mappings`
- **Credential Storage**: Store credentials using `--store` for git backend

### Basic Usage

```bash
# Store credentials for any URL-based service
secretinit --store --url https://api.example.com --user myuser

# Multi-credential mode: Creates API_URL, API_USER, API_PASS
export API="secretinit:git:https://api.example.com"
secretinit myapp

# Single credential mode: Replace with specific value
export API_TOKEN="secretinit:git:https://api.example.com:::password"
export DB_PASS="secretinit:aws:sm:myapp/database:::password"
secretinit myapp

# With environment variable mappings (via command line)
secretinit -m "DATABASE_USERNAME=API_USER,DATABASE_PASSWORD=API_PASS" myapp

# With environment variable mappings (via environment variable)
SECRETINIT_MAPPINGS="DATABASE_USERNAME=API_USER,DATABASE_PASSWORD=API_PASS" secretinit myapp
```

-----

## Quick Start Examples

### Git Credential Storage
```bash
# Store credentials for any API or service
secretinit --store --url https://api.example.com --user myuser
secretinit --store --url https://database.company.com --user dbuser

# Single credential replacement
export API_TOKEN="secretinit:git:https://api.example.com:::password"
export DB_USER="secretinit:git:https://database.company.com:::username"
secretinit curl -H "Authorization: Bearer \$API_TOKEN" https://api.example.com/data

# Multi-credential mode: Creates API_URL, API_USER, API_PASS
export API="secretinit:git:https://api.example.com"
export DATABASE="secretinit:git:https://database.company.com"
secretinit myapp

# Environment variable mappings (via command line)
export API="secretinit:git:https://api.example.com"
secretinit -m "DATABASE_USERNAME=API_USER,DATABASE_PASSWORD=API_PASS" myapp

# Environment variable mappings (via environment variable)
export API="secretinit:git:https://api.example.com"
SECRETINIT_MAPPINGS="DATABASE_USERNAME=API_USER,DATABASE_PASSWORD=API_PASS" secretinit myapp
```

### Git Credential Helper Configuration

Configure a secure credential helper for your platform:

#### macOS
```bash
# Option 1: Git Credential Manager (recommended)
git config --global credential.helper manager

# Option 2: Use the osxkeychain
git config --global credential.helper osxkeychain
```

#### Linux
```bash
# Option 1: Git Credential Manager (recommended) 
git config --global credential.helper manager

# Option 2: Use in memory cache
git config --global credential.helper 'cache --timeout=<seconds>'
```

#### Windows
```bash
# Option 1: Git Credential Manager (recommended, included in Windows)
git config --global credential.helper manager

# Option 2: Windows Credential Manager
git config --global credential.helper wincred
```

#### WSL
```bash
# Use Git Credential Manager from Windows
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

### Cloud Examples

```bash
# AWS Secrets Manager (supported)
export DB_PASSWORD="secretinit:aws:sm:myapp/db-creds:::password"
secretinit myapp

# AWS Parameter Store (supported)
export APP_CONFIG="secretinit:aws:ps:/myapp/config"
secretinit myapp

# AWS Secrets Manager with ARN
export DB_CREDS="secretinit:aws:sm:arn:aws:secretsmanager:us-west-2:123456789012:secret:myapp/db-creds-AbCdEf"
secretinit myapp

# AWS Parameter Store with JSON key extraction
export DB_HOST="secretinit:aws:ps:/myapp/db-config:::database.host"
secretinit myapp

# AWS Secrets Manager with JSON key extraction
export API_KEY="secretinit:aws:sm:myapp/api-config:::api_key"
export DB_HOST="secretinit:aws:sm:myapp/db-config:::database.host"
secretinit myapp
```

### Future Cloud Examples
```bash
# Google Cloud Secret Manager  
export API_KEY="secretinit:gcp:sm:myproject/api-key"
secretinit myapp

# Azure Key Vault
export CERTIFICATE="secretinit:azure:kv:myvault/ssl-cert"
secretinit myapp
```

## Configuration

### Environment Variables

**`secretinit`** supports configuration through environment variables:

- **`SECRETINIT_MAPPINGS`**: Set environment variable mappings (same format as `-m/--mappings`)
- **`SECRETINIT_LOG_LEVEL`**: Set to `DEBUG` for detailed logging output

```bash
# Using environment variable for mappings
SECRETINIT_MAPPINGS="DATABASE_USERNAME=API_USER,DATABASE_PASSWORD=API_PASS" secretinit myapp

# Enable debug logging
SECRETINIT_LOG_LEVEL=DEBUG secretinit myapp

# Combine both
SECRETINIT_MAPPINGS="DATABASE_USERNAME=API_USER" SECRETINIT_LOG_LEVEL=DEBUG secretinit myapp
```

**Mapping Priority**: Command line mappings (`-m/--mappings`) override environment variable mappings (`SECRETINIT_MAPPINGS`), allowing you to set defaults via environment variables and override specific mappings as needed.

## Important Notes

- Only environment variables with `secretinit:` prefix are processed for credential loading
- Regular environment variables without this prefix are ignored for credential processing  
- **Multi-credential mode**: When no keyPath is specified for git backend, creates three variables (`*_URL`, `*_USER`, `*_PASS`)
- **Single credential mode**: When keyPath is specified (`:::password` or `:::username`), replaces the variable with the specific value
- **Credential storage works for any URL-based service** - not limited to Git repositories
- Credentials are stored directly in Git's credential helper system (no prefix needed for storage)
- Mappings use equals syntax: `TARGET=SOURCE,TARGET2=SOURCE2`
- Configure a secure credential helper before storing credentials