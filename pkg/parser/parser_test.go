package parser_test // Conventionally, test files are in a _test package

import (
	"reflect" // Used for deep comparison of structs
	"testing"

	// Import the package you're testing.
	"github.com/liifi/secretinit/pkg/parser"
)

func TestParseSecretString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		expected parser.SecretSource
	}{
		// Git Tests
		{
			name:    "Git: No KeyPath",
			input:   "git:https://github.com/myorg/secrets.git",
			wantErr: false,
			expected: parser.SecretSource{
				Backend:  "git",
				Service:  "", // Git doesn't use a service field in this model
				Resource: "https://github.com/myorg/secrets.git",
				KeyPath:  "",
			},
		},
		{
			name:    "Git: Request Username",
			input:   "git:https://github.com/myorg/private-secrets.git:::username",
			wantErr: false,
			expected: parser.SecretSource{
				Backend:  "git",
				Service:  "",
				Resource: "https://github.com/myorg/private-secrets.git",
				KeyPath:  "username",
			},
		},
		{
			name:    "Git: Request Token",
			input:   "git:https://github.com/myorg/token-repo.git:::token",
			wantErr: false,
			expected: parser.SecretSource{
				Backend:  "git",
				Service:  "",
				Resource: "https://github.com/myorg/token-repo.git",
				KeyPath:  "token",
			},
		},
		{
			name:    "Git: Embedded Creds, Request Password",
			input:   "git:https://user:pass@github.com/myorg/embedded-creds.git:::password",
			wantErr: false,
			expected: parser.SecretSource{
				Backend:  "git",
				Service:  "",
				Resource: "https://user:pass@github.com/myorg/embedded-creds.git", // The URL part, including embedded creds
				KeyPath:  "password",
			},
		},
		{
			name:    "Git: SSH URL, Request Specific KeyPath '_anything'",
			input:   "git:ssh://git@github.com/myorg/ssh-secrets.git:::_anything",
			wantErr: false,
			expected: parser.SecretSource{
				Backend:  "git",
				Service:  "",
				Resource: "ssh://git@github.com/myorg/ssh-secrets.git",
				KeyPath:  "_anything", // Test that "_anything" is captured as a literal KeyPath string
			},
		},

		// AWS Tests
		{
			name:    "AWS: Simple Name",
			input:   "aws:sm:my-app/db-creds",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "aws", Service: "sm", Resource: "my-app/db-creds", KeyPath: "",
			},
		},
		{
			name:    "AWS: Simple Name with Key",
			input:   "aws:sm:my-app/db-creds:::username",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "aws", Service: "sm", Resource: "my-app/db-creds", KeyPath: "username",
			},
		},
		{
			name:    "AWS: Full ARN, no Key",
			input:   "aws:sm:arn:aws:secretsmanager:us-east-1:123456789012:secret:my-app/db-creds-ABCDEF",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "aws", Service: "sm", Resource: "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-app/db-creds-ABCDEF", KeyPath: "",
			},
		},
		{
			name:    "AWS: Full ARN with Key",
			input:   "aws:sm:arn:aws:secretsmanager:us-east-1:123456789012:secret:my-app/db-creds-ABCDEF:::username",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "aws", Service: "sm", Resource: "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-app/db-creds-ABCDEF", KeyPath: "username",
			},
		},
		{
			name:    "AWS: Full ARN, Request Specific KeyPath '_anything'",
			input:   "aws:sm:arn:aws:secretsmanager:us-east-1:123456789012:secret:my-app/db-creds-ABCDEF:::_anything",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "aws", Service: "sm", Resource: "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-app/db-creds-ABCDEF", KeyPath: "_anything",
			},
		},
		{
			name:    "AWS: Parameter Store",
			input:   "aws:ps:/my-app/config/api_key",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "aws", Service: "ps", Resource: "/my-app/config/api_key", KeyPath: "",
			},
		},
		{
			name:    "AWS: Secret with Colon in Resource ID (no key) - Passes correctly",
			input:   "aws:sm:secret-name-with:colon:in:resource-ID",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "aws", Service: "sm", Resource: "secret-name-with:colon:in:resource-ID", KeyPath: "",
			},
		},
		{
			name:    "AWS: Secret with Colon in Resource ID (Specific KeyPath '_anything')",
			input:   "aws:sm:secret-name-with:colon:in:resource-ID:::_anything",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "aws", Service: "sm", Resource: "secret-name-with:colon:in:resource-ID", KeyPath: "_anything",
			},
		},
		{
			name:    "AWS: Secret with Colon in Resource ID (with key) - Using ':::'",
			input:   "aws:sm:secret-name-with:colon:in:resource-ID:::username",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "aws", Service: "sm", Resource: "secret-name-with:colon:in:resource-ID", KeyPath: "username",
			},
		},

		// GCP Tests
		{
			name:    "GCP: Basic Secret",
			input:   "gcp:sm:my-gcp-project/db-pass",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "gcp", Service: "sm", Resource: "my-gcp-project/db-pass", KeyPath: "",
			},
		},
		{
			name:    "GCP: With Version - Resource includes version",
			input:   "gcp:sm:my-gcp-project/api-key:2",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "gcp", Service: "sm", Resource: "my-gcp-project/api-key:2", KeyPath: "",
			},
		},
		{
			name:    "GCP: With KeyPath (Service Account) - Using ':::'",
			input:   "gcp:sm:my-gcp-project/service-account:latest:::private_key_id",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "gcp", Service: "sm", Resource: "my-gcp-project/service-account:latest", KeyPath: "private_key_id",
			},
		},
		{
			name:    "GCP: Colon in Secret Name - Passes correctly",
			input:   "gcp:sm:my-gcp-project/some:secret:name",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "gcp", Service: "sm", Resource: "my-gcp-project/some:secret:name", KeyPath: "",
			},
		},
		{
			name:    "GCP: Colon in Secret Name with Key - Using ':::'",
			input:   "gcp:sm:my-gcp-project/some:secret:name:::user",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "gcp", Service: "sm", Resource: "my-gcp-project/some:secret:name", KeyPath: "user",
			},
		},

		// Azure Tests
		{
			name:    "Azure: Basic Secret",
			input:   "azure:kv:my-keyvault/app-secret",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "azure", Service: "kv", Resource: "my-keyvault/app-secret", KeyPath: "",
			},
		},
		{
			name:    "Azure: With KeyPath - Using ':::'",
			input:   "azure:kv:my-keyvault/blob-storage-conn-string:::AccountKey",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "azure", Service: "kv", Resource: "my-keyvault/blob-storage-conn-string", KeyPath: "AccountKey",
			},
		},
		{
			name:    "Azure: Colon in Secret Name - Passes correctly",
			input:   "azure:kv:my-keyvault/another:secret:with:colon",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "azure", Service: "kv", Resource: "my-keyvault/another:secret:with:colon", KeyPath: "",
			},
		},
		{
			name:    "Azure: Colon in Secret Name with Specific KeyPath '_anything'",
			input:   "azure:kv:my-keyvault/another:secret:with:colon:::_anything",
			wantErr: false,
			expected: parser.SecretSource{
				Backend: "azure", Service: "kv", Resource: "my-keyvault/another:secret:with:colon", KeyPath: "_anything",
			},
		},

		// Error Cases
		{
			name:    "Invalid: Missing Backend",
			input:   "my-secret",
			wantErr: true,
		},
		{
			name:    "Invalid: Unsupported Backend",
			input:   "unsupported:type:my-secret",
			wantErr: true,
		},
		{
			name:    "Invalid AWS: Missing Service",
			input:   "aws:my-secret",
			wantErr: true,
		},
		{
			name:    "Invalid Git URL",
			input:   "git:invalid url path:::username",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseSecretString(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSecretString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParseSecretString() got = %+v, want %+v", got, tt.expected)
			}
		})
	}
}