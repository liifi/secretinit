# Store credential
$answer = Read-Host "Do you want to wipe and store the pass for https://user@example.com? (y/N)"
if ($answer -eq "y" -or $answer -eq "Y") {
    secretinit.exe --store --url https://example.com --user user
} else {
    Write-Host "Credential storage skipped."
}

# Test as credential loader with mappings (command line)
$env:M="secretinit:git:https://user@example.com"; secretinit.exe -m "A_URL=M_URL,A_USER=M_USER,A_PASS=M_PASS" pwsh -c 'env | grep -i a_'

# Test as credential loader with mappings (environment variable)
$env:M="secretinit:git:https://user@example.com"; $env:SECRETINIT_MAPPINGS="A_URL=M_URL,A_USER=M_USER,A_PASS=M_PASS"; secretinit.exe pwsh -c 'env | grep -i a_'

# Test as secret retriever only
$env:TOKEN="secretinit:git:https://user@example.com:::password"; secretinit.exe pwsh -c 'env | grep TOKEN'

# Test as secret value only
secretinit.exe -o git:user@example.com:::password