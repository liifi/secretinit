# secretinit

A command-line tool that injects secrets into your application's environment variables from secure storage backends.

## Quick Examples

```bash
# 1. Run your app with secret injection
export DB_PASSWORD="secretinit:aws:sm:myapp/db-creds:::password"
secretinit myapp

# 2. Get a single secret value
secretinit --stdout "git:https://api.example.com:::password"

# 3. Multi-credential expansion + mapping
export API="secretinit:git:https://api.example.com"
secretinit -m "DATABASE_USER=API_USER,DATABASE_PASS=API_PASS" myapp

# 4. Use .env file
echo 'API_TOKEN=secretinit:git:https://api.example.com:::password' > .env
secretinit myapp
```

## How It Works

Instead of hardcoding secrets, you use placeholder strings that tell `secretinit` where to find the real values:

```bash
# Before: Hardcoded secret 😟
export API_KEY="sk-1234567890abcdef"
myapp

# After: Secret reference 😎  
export API_KEY="secretinit:aws:sm:myapp/api-key"
secretinit myapp
```

`secretinit` fetches the real secret and launches your app with the actual value.

## Secret Address Format

```
backend:service:resource[:::key_path]
```

- **backend**: `git`, `aws` (more coming soon)
- **service**: `sm` (Secrets Manager), `ps` (Parameter Store) for AWS
- **resource**: Secret name/path/URL
- **key_path**: Optional - extract specific field from JSON secrets

## Supported Backends

| Backend | Service | Example |
|---------|---------|---------|
| Git | Any URL | `git:https://api.example.com:::password` |
| AWS | Secrets Manager | `aws:sm:myapp/db-creds:::password` |
| AWS | Parameter Store | `aws:ps:/myapp/config:::database.host` |

## Usage Modes

### 1. Process Launcher (Most Common)
Run your application with secret injection:

```bash
# Single secret
export API_KEY="secretinit:aws:sm:myapp/api-key"
secretinit myapp

# Multiple secrets  
export DB_USER="secretinit:git:https://db.example.com:::username"
export DB_PASS="secretinit:git:https://db.example.com:::password"
secretinit myapp

# Multi-credential mode (creates API_URL, API_USER, API_PASS)
export API="secretinit:git:https://api.example.com"
secretinit myapp

# Map auto-created variables to what your app expects
export API="secretinit:git:https://api.example.com"
secretinit -m "DB_HOST=API_URL,DB_USER=API_USER,DB_PASS=API_PASS" myapp
```

### 2. Single Secret Retrieval
Get one secret value to stdout:

```bash
# Get password for scripting
PASSWORD=$(secretinit --stdout "aws:sm:myapp/db:::password")

# Use in command substitution
curl -u "user:$(secretinit -o git:https://api.example.com:::password)" https://api.example.com
```

### 3. Environment Variable Mappings
Copy secret values to additional variables or rename auto-expanded variables:

```bash
# Git multi-credential expansion creates API_USER, API_PASS
# Map them to what your legacy app expects
export API="secretinit:git:https://api.example.com"
secretinit -m "DATABASE_USERNAME=API_USER,DATABASE_PASSWORD=API_PASS" myapp

# Copy one secret to multiple variable names
export SECRET="secretinit:aws:sm:myapp/token"
secretinit -m "API_TOKEN=SECRET,AUTH_KEY=SECRET,ACCESS_TOKEN=SECRET" myapp

# Environment variable mappings
SECRETINIT_MAPPINGS="DATABASE_USERNAME=API_USER,DATABASE_PASSWORD=API_PASS" secretinit myapp
```

## Git Backend Setup

The git backend uses your OS's secure credential storage:

```bash
# Store credentials for any service (not just Git!)
secretinit --store --url https://api.example.com --user myuser

# Configure credential helper (one-time setup)
git config --global credential.helper manager  # Recommended for all platforms
```

## Quick Setup

1. **Install Git** and configure a credential helper
2. **For AWS**: Configure AWS credentials (`aws configure` or IAM roles)
3. **Store some credentials**:
   ```bash
   secretinit --store --url https://api.example.com --user myuser
   ```
4. **Test it**:
   ```bash
   export API_KEY="secretinit:git:https://api.example.com:::password"
   secretinit echo "API_KEY=\$API_KEY"
   ```

## Environment Variables

- `SECRETINIT_MAPPINGS`: Set variable mappings (`TARGET=SOURCE,TARGET2=SOURCE2`)
- `SECRETINIT_LOG_LEVEL`: Set to `DEBUG` for detailed logging

## .env File Support

`secretinit` automatically loads environment variables from a `.env` file in the current directory:

```bash
# .env file
API_TOKEN=secretinit:git:https://api.example.com:::password
DB_USER=secretinit:git:https://db.example.com:::username
DB_PASS=secretinit:git:https://db.example.com:::password

# Project-specific mappings
SECRETINIT_MAPPINGS=DATABASE_USERNAME=DB_USER,DATABASE_PASSWORD=DB_PASS
```

### .env File Options:
- **Default**: Automatically loads `.env` from current directory
- **Custom file**: `secretinit -e prod.env myapp`
- **Disable loading**: `secretinit -n myapp`
- **Precedence**: `.env file variables` override `system environment variables`

## Platform-Specific Notes

### Git Credential Helpers
- **macOS/Linux/Windows**: `git config --global credential.helper manager`
- **Legacy options**: `osxkeychain` (macOS), `wincred` (Windows), `cache` (Linux)

### WSL Users
Use Linux credential helpers in WSL rather than Windows GCM to avoid hanging issues.

---

For more examples and advanced usage, run `secretinit --help`.