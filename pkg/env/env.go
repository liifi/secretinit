package env


import (
	"os"
	"strings"
)

func ScanSecretEnvVars() map[string]string {
	secretVars := make(map[string]string)
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			if strings.HasPrefix(parts[1], "secretinit:") {
				secretVars[parts[0]] = strings.TrimPrefix(parts[1], "secretinit:")
			}
		}
	}
	return secretVars
}
