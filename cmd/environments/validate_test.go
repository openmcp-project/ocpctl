package environments

import (
	"testing"
)

func TestValidateEnvironmentName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "simple name", input: "my-env", wantErr: false},
		{name: "uuid", input: "81011f8b-c812-40ab-b26e-cac6ecc6d3c7", wantErr: false},
		{name: "single char", input: "a", wantErr: false},
		{name: "digits only", input: "123", wantErr: false},
		{name: "dots and hyphens", input: "my.env-1", wantErr: false},
		{name: "exactly 36 chars", input: "81011f8b-c812-40ab-b26e-cac6ecc6d3c7", wantErr: false},
		{name: "37 chars", input: "81011f8b-c812-40ab-b26e-cac6ecc6d3c71", wantErr: true},
		{name: "uppercase", input: "MyEnv", wantErr: true},
		{name: "space", input: "my env", wantErr: true},
		{name: "special chars", input: "äöü", wantErr: true},
		{name: "underscore", input: "my_env", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvironmentName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEnvironmentName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
