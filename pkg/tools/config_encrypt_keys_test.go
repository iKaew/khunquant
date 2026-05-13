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
	for _, want := range []string{"khunquant auth encrypt", "does not accept passphrases"} {
		if !strings.Contains(desc, want) {
			t.Errorf("Description should mention %q, got %q", want, desc)
		}
	}
}

func TestConfigEncryptKeysTool_ParametersDoNotAcceptSecrets(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type should be 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	for _, forbidden := range []string{"passphrase", "old_passphrase", "old_ssh_key_path"} {
		if _, ok := props[forbidden]; ok {
			t.Errorf("secret-bearing property %q should not be accepted", forbidden)
		}
	}
	if _, ok := props["rotate"]; !ok {
		t.Error("expected deprecated rotate property to remain documented")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("required should be a slice")
	}
	if len(required) != 0 {
		t.Errorf("no secret args should be required, got %v", required)
	}
}

func TestConfigEncryptKeysTool_ExecuteRejectsInteractiveUseThroughTool(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	result := tool.Execute(context.Background(), map[string]any{})
	if result == nil {
		t.Fatal("Execute should return result")
	}
	if !result.IsError {
		t.Fatal("tool execution should fail closed")
	}
	if !strings.Contains(result.ForLLM, "interactive") {
		t.Fatalf("error should direct caller to interactive CLI, got %q", result.ForLLM)
	}
}

func TestConfigEncryptKeysTool_ExecuteRejectsPassphraseArgs(t *testing.T) {
	tool := newTestConfigEncryptKeysTool(t)

	for _, args := range []map[string]any{
		{"passphrase": "new-secret"},
		{"old_passphrase": "old-secret"},
		{"passphrase": 12345},
	} {
		result := tool.Execute(context.Background(), args)
		if result == nil {
			t.Fatal("Execute should return result")
		}
		if !result.IsError {
			t.Fatalf("secret args should fail closed: %#v", args)
		}
		if !strings.Contains(result.ForLLM, "passphrases are not accepted") {
			t.Fatalf("error should reject passphrase args, got %q", result.ForLLM)
		}
	}
}
