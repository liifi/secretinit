# Store credential
$answer = Read-Host "Do you want to wipe and store the pass for https://user@example.com? (y/N)"
if ($answer -eq "y" -or $answer -eq "Y") {
    credinit.exe --store --url https://example.com --user user
} else {
    Write-Host "Credential storage skipped."
}

# Test as credential loader with mappings
$env:M="secretinit:git:https://user@example.com"; credinit.exe -m "M_URL->A_URL,M_USER->A_USER,M_PASS->A_PASS" pwsh -c "env | grep -i a_"

# Test as secret retriever only
$env:TOKEN="secretinit:git:https://user@example.com:::password"; credinit.exe pwsh -c "env | grep TOKEN"

# Test as secret value only
credinit.exe -o git:user@example.com:::password