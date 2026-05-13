package auth

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthCommand(t *testing.T) {
	cmd := NewAuthCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "auth", cmd.Use)
	assert.Equal(t, "Manage authentication (login, logout, status)", cmd.Short)

	assert.Len(t, cmd.Aliases, 0)

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	assert.False(t, cmd.HasFlags())
	assert.True(t, cmd.HasSubCommands())

	allowedCommands := []string{
		"login",
		"logout",
		"status",
		"models",
		"encrypt",
	}

	subcommands := cmd.Commands()
	assert.Len(t, subcommands, len(allowedCommands))

	for _, subcmd := range subcommands {
		found := slices.Contains(allowedCommands, subcmd.Name())
		assert.True(t, found, "unexpected subcommand %q", subcmd.Name())

		assert.Len(t, subcmd.Aliases, 0)
		assert.False(t, subcmd.Hidden)

		assert.False(t, subcmd.HasSubCommands())

		assert.Nil(t, subcmd.Run)
		assert.NotNil(t, subcmd.RunE)

		assert.Nil(t, subcmd.PersistentPreRun)
		assert.Nil(t, subcmd.PersistentPostRun)
	}
}

func TestSecurityConfigHasEncryptedCredentials(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom-config.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	hasEncrypted, err := securityConfigHasEncryptedCredentials(configPath)
	require.NoError(t, err)
	assert.False(t, hasEncrypted)

	if err := os.WriteFile(filepath.Join(dir, ".security.yml"), []byte("channels:\n  telegram:\n    token: enc://abc\n"), 0o600); err != nil {
		t.Fatalf("write security config: %v", err)
	}

	hasEncrypted, err = securityConfigHasEncryptedCredentials(configPath)
	require.NoError(t, err)
	assert.True(t, hasEncrypted)
}

func TestReadOldSSHKeyPathPreservesSpaces(t *testing.T) {
	got, err := readOldSSHKeyPath(bufio.NewReader(strings.NewReader("/tmp/my keys/id_ed25519\n")))
	require.NoError(t, err)
	assert.Equal(t, "/tmp/my keys/id_ed25519", got)
}

func TestBackupSecurityConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	securityPath := filepath.Join(dir, ".security.yml")
	securityContent := []byte("channels:\n  telegram:\n    token: enc://abc\n")
	if err := os.WriteFile(configPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(securityPath, securityContent, 0o600); err != nil {
		t.Fatalf("write security config: %v", err)
	}

	require.NoError(t, backupSecurityConfig(configPath))

	backup, err := os.ReadFile(securityPath + ".bak")
	require.NoError(t, err)
	assert.Equal(t, string(securityContent), string(backup))
}
