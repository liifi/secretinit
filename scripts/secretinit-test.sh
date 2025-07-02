# Store credential
read -p "Do you want to wipe and store the pass for https://user@example.com? [y/N]: " yn
case "$yn" in
    [Yy]* ) secretinit --store --url https://example.com --user user;;
    * ) echo "Skipping credential store.";;
esac

# Test as credential loader with mappings (command line)
M=secretinit:git:https://user@example.com secretinit -m "A_URL=M_URL,A_USER=M_USER,A_PASS=M_PASS" bash -c "env | grep -i a_"

# Test as credential loader with mappings (environment variable)
M=secretinit:git:https://user@example.com SECRETINIT_MAPPINGS="A_URL=M_URL,A_USER=M_USER,A_PASS=M_PASS" secretinit bash -c "env | grep -i a_"

# Test as secret retriever only
TOKEN=secretinit:git:https://user@example.com:::password secretinit bash -c "env | grep TOKEN"

# Test as secret value only
secretinit -o git:user@example.com:::password