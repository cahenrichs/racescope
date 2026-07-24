package config

import "testing"

func TestFromEnv(t *testing.T) {
	tests := map[string]struct {
		port        string
		wantAddress string
	}{
		"default": {wantAddress: ":8080"},
		"custom":  {port: "9090", wantAddress: ":9090"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("PORT", test.port)

			cfg, err := FromEnv()
			if err != nil {
				t.Fatalf("FromEnv() error = %v", err)
			}
			if cfg.Address != test.wantAddress {
				t.Fatalf("Address = %q, want %q", cfg.Address, test.wantAddress)
			}
		})
	}
}

func TestFromEnvRejectsInvalidPort(t *testing.T) {
	for _, port := range []string{"not-a-port", "0", "65536"} {
		t.Run(port, func(t *testing.T) {
			t.Setenv("PORT", port)
			if _, err := FromEnv(); err == nil {
				t.Fatal("FromEnv() error = nil, want PORT validation error")
			}
		})
	}
}
