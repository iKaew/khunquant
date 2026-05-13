package tools

import (
	"context"
	"os"
	"path/filepath"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

// ConfigEncryptKeysTool encrypts all SecureString credentials in .security.yml
// using AES-256-GCM with an SSH private key.
//
// Passphrases are intentionally not accepted through tool arguments because
// tool calls can be recorded in conversation history, logs, or tracing.
type ConfigEncryptKeysTool struct {
	cfg *config.Config
}

// NewConfigEncryptKeysTool creates the tool.
func NewConfigEncryptKeysTool(cfg *config.Config) *ConfigEncryptKeysTool {
	return &ConfigEncryptKeysTool{cfg: cfg}
}

func (t *ConfigEncryptKeysTool) Name() string {
	return NameConfigEncryptKeys
}

func (t *ConfigEncryptKeysTool) Description() string {
	return `Encrypt all exchange API keys and secrets in .security.yml using AES-256-GCM.

IMPORTANT SECURITY RULES:
- This tool does not accept passphrases through arguments.
- Use the interactive CLI command instead: khunquant auth encrypt.

Passphrases must be entered through terminal raw mode so they do not enter the
LLM context, tool-call payloads, logs, or traces.`
}

func (t *ConfigEncryptKeysTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"rotate": map[string]any{
				"type":        "boolean",
				"description": "Deprecated. Rotation must be performed by the interactive CLI command.",
			},
		},
		"required": []string{},
	}
}

func (t *ConfigEncryptKeysTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	if _, ok := args["passphrase"]; ok {
		return ErrorResult("passphrases are not accepted through tool arguments; run `khunquant auth encrypt` in an interactive terminal")
	}
	if _, ok := args["old_passphrase"]; ok {
		return ErrorResult("passphrases are not accepted through tool arguments; run `khunquant auth encrypt` in an interactive terminal")
	}
	return ErrorResult("credential encryption requires interactive secret entry; run `khunquant auth encrypt` in a terminal")
}

// resolveConfigPath returns the config.json path using the same priority as the CLI.
func resolveConfigPath() string {
	if p := os.Getenv("KHUNQUANT_CONFIG"); p != "" {
		return p
	}
	if home := os.Getenv("KHUNQUANT_HOME"); home != "" {
		return filepath.Join(home, "config.json")
	}
	userHome, _ := os.UserHomeDir()
	return filepath.Join(userHome, ".khunquant", "config.json")
}
