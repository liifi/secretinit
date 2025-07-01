### What is this?

**`secretinit`** is a command-line tool designed to streamline how applications access and use secrets like API keys, database credentials, or sensitive configuration values. It provides a consistent way to load these into your application's environment variables.

**Related Tool**: This project also includes [`credinit`](cmd/credinit/README.md), a credential management tool that leverages Git's credential helper system as general-purpose, cross-platform secure storage for any URL-based service. It focuses on URL-based credentials and provides automatic multi-credential loading (creating *_URL, *_USER, and *_PASS variables).

> **Important Note**: The "git" backend in both tools uses Git's credential helper system as **general-purpose credential storage for any URL-based service** - not just Git repositories. This means you can securely store and retrieve credentials for APIs, databases, web services, SaaS platforms, or any other service accessed via URL, using your operating system's native secure storage.

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

## Tools Overview

### `secretinit` - Universal Secret Injection
The main tool that supports multiple backend types and provides a unified interface for secret injection across different cloud providers and systems.

**Usage:**
```bash
# Basic usage
export API_KEY="secretinit:git:https://api.example.com:::password"
secretinit myapp

# With environment variable mappings
secretinit -m "API_KEY->GITHUB_TOKEN,DB_PASS->DATABASE_PASSWORD" myapp
```

### `credinit` - General-Purpose Credential Storage
A specialized tool that leverages Git's credential helper system as universal, cross-platform credential storage for **any URL-based service**. It can store and retrieve credentials for APIs, databases, web services, or any other URL-accessible resource using your operating system's native secure storage (Keychain on macOS, Credential Manager on Windows, etc.). **Note**: Despite using Git's credential system, this is not limited to git repositories.

**Usage:**
```bash
# Store credentials for any service
credinit --store --url https://api.example.com --user myuser

# Multi-credential mode: Creates *_URL, *_USER, and *_PASS variables
export API=secretinit:git:https://api.example.com
credinit myapp

# Single credential mode: Replace with specific value
export API_TOKEN=secretinit:git:https://api.example.com:::password
credinit myapp
```

For detailed information about `credinit`, see [cmd/credinit/README.md](cmd/credinit/README.md).

-----

## Quick Start Examples

### Git Credential Storage (Both Tools)
```bash
# Store credentials for any API or service using credinit
credinit --store --url https://api.example.com --user myuser
credinit --store --url https://database.company.com --user dbuser

# secretinit: Single credential replacement
export API_TOKEN="secretinit:git:https://api.example.com:::password"
export DB_USER="secretinit:git:https://database.company.com:::username"
secretinit curl -H "Authorization: Bearer \$API_TOKEN" https://api.example.com/data

# credinit: Multi-credential mode (creates *_URL, *_USER, *_PASS)
export API=secretinit:git:https://api.example.com
export DATABASE=secretinit:git:https://database.company.com
credinit myapp

# credinit: Single credential mode (with keyPath)
export TOKEN=secretinit:git:https://api.example.com:::password
credinit myapp
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

### Future Cloud Examples (secretinit only)
```bash
# Google Cloud Secret Manager  
export API_KEY="secretinit:gcp:sm:myproject/api-key"
secretinit myapp

# Azure Key Vault
export CERTIFICATE="secretinit:azure:kv:myvault/ssl-cert"
secretinit myapp
```