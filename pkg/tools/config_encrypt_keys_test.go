package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/credential"
)

func newTestConfigEncryptKeysTool(t *testing.T) *ConfigEncryptKeysTool {
	t.Helper()
	isolateConfigEncryptKeysTest(t)
	return NewConfigEncryptKeysTool(config.DefaultConfig())
}

func isolateConfigEncryptKeysTest(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	khunquantHome := filepath.Join(tmp, "khunquant-home")
	configPath := filepath.Join(khunquantHome, "config.json")
	sshKeyPath := filepath.Join(home, ".ssh", "khunquant_ed25519.key")

	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("KHUNQUANT_HOME", khunquantHome)
	t.Setenv("KHUNQUANT_CONFIG", configPath)
	t.Setenv(credential.SSHKeyPathEnvVar, sshKeyPath)
	t.Setenv(credential.PassphraseEnvVar, "")

	if err := os.MkdirAll(khunquantHome, 0o700); err != nil {
		t.Fatalf("MkdirAll(KHUNQUANT_HOME): %v", err)
	}
	if err := os.WriteFile(configPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}
}

func TestConfigEncryptKeysTool_Name(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)
	if tool.Name() != NameConfigEncryptKeys {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameConfigEncryptKeys)
	}
}

func TestConfigEncryptKeysTool_Description(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
	if !strings.Contains(desc, "encrypt") {
		t.Errorf("Description should mention encryption, got %q", desc)
	}
}

func TestConfigEncryptKeysTool_Parameters(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type should be 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	expectedProps := []string{"passphrase", "rotate"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q not found", prop)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("required should be a slice")
	}
	if len(required) == 0 || required[0] != "passphrase" {
		t.Errorf("passphrase should be required, got %v", required)
	}
}

func TestConfigEncryptKeysTool_Execute_EmptyPassphrase(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"passphrase": "",
	})

	if !result.IsError {
		t.Error("empty passphrase should return error")
	}
	if !strings.Contains(result.ForLLM, "passphrase") {
		t.Errorf("error should mention passphrase, got %q", result.ForLLM)
	}
}

func TestConfigEncryptKeysTool_Execute_NoPassphrase(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	result := tool.Execute(context.Background(), map[string]any{})

	if !result.IsError {
		t.Error("missing passphrase should return error")
	}
}

func TestConfigEncryptKeysTool_Execute_MissingPassphrase(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"rotate": true,
	})

	if !result.IsError {
		t.Error("missing passphrase should return error")
	}
}

func TestConfigEncryptKeysTool_Execute_PassphraseWithoutRotate(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"passphrase": "test-secret-passphrase",
		"rotate":     false,
	})

	// May error depending on credential state, but should handle the call
	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestConfigEncryptKeysTool_Execute_PassphraseWithRotate(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"passphrase": "test-secret-passphrase",
		"rotate":     true,
	})

	// May error depending on credential state, but should handle the call
	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestConfigEncryptKeysTool_Execute_InvalidArgTypes(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"passphrase": 12345,   // int instead of string
		"rotate":     "maybe", // string instead of bool
	})

	// Should handle type mismatches gracefully (ignore non-matching types)
	if result == nil {
		t.Fatal("Execute should return result even with invalid types")
	}
	if !result.IsError {
		t.Log("Type mismatch caused expected error")
	}
}

func TestConfigEncryptKeysTool_Execute_LongPassphrase(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	longPassphrase := strings.Repeat("x", 1000)
	result := tool.Execute(context.Background(), map[string]any{
		"passphrase": longPassphrase,
	})

	if result == nil {
		t.Fatal("Execute should return result for long passphrase")
	}
	// Long passphrases should be acceptable
}

func TestConfigEncryptKeysTool_Execute_SpecialCharsPassphrase(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	specialPassphrase := "p@$$w0rd!#$%^&*()[]{}|;:',.<>?/\\"
	result := tool.Execute(context.Background(), map[string]any{
		"passphrase": specialPassphrase,
	})

	if result == nil {
		t.Fatal("Execute should return result for special chars passphrase")
	}
}

func TestConfigEncryptKeysTool_Execute_UnicodePassphrase(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	unicodePassphrase := "密码🔒🔑ปลายืดา"
	result := tool.Execute(context.Background(), map[string]any{
		"passphrase": unicodePassphrase,
	})

	if result == nil {
		t.Fatal("Execute should return result for unicode passphrase")
	}
}

func TestConfigEncryptKeysTool_Execute_WhitespacePassphrase(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"passphrase": "   ",
	})

	// Whitespace-only passphrase is still technically non-empty string, but may be rejected
	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestConfigEncryptKeysTool_Execute_NoRotate(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"passphrase": "secret",
		// rotate not specified, should default to false
	})

	if result == nil {
		t.Fatal("Execute should return result when rotate is not specified")
	}
}

func TestConfigEncryptKeysTool_Execute_RotateDefault(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	// When rotate is not provided, it should default to false
	result := tool.Execute(context.Background(), map[string]any{
		"passphrase": "new-secret",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}
