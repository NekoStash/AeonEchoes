package provider

import (
	"os"
	"testing"

	"aeonechoes/server/internal/domain"
)

func TestAuthHeaderValueIgnoresAPIKeyEnv(t *testing.T) {
	t.Setenv("AEON_ECHOES_TEST_API_KEY", "env-secret")

	if got := AuthHeaderValue(domain.ProviderConfig{APIKeyEnv: "AEON_ECHOES_TEST_API_KEY"}); got != "" {
		t.Fatalf("AuthHeaderValue() = %q, want empty when only APIKeyEnv is configured", got)
	}

	if got := AuthHeaderValue(domain.ProviderConfig{APIKey: " direct-secret ", APIKeyEnv: "AEON_ECHOES_TEST_API_KEY"}); got != "direct-secret" {
		t.Fatalf("AuthHeaderValue() = %q, want trimmed direct API key", got)
	}

	if value := os.Getenv("AEON_ECHOES_TEST_API_KEY"); value != "env-secret" {
		t.Fatalf("test environment sanity check failed: %q", value)
	}
}
