package uninstall

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathDepth(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix-style paths")
	}
	tests := []struct {
		path  string
		depth int
	}{
		{"/", 0},
		{"/foo", 1},
		{"/foo/bar", 2},
		{"/foo/bar/baz", 3},
		{"/home/user/.khunquant", 3},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.depth, pathDepth(tt.path), "pathDepth(%q)", tt.path)
	}
}

func TestValidateRemovePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix-style paths")
	}
	validPath := filepath.Join(t.TempDir(), "khunquant")

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"empty", "", true},
		{"relative path", "foo/bar", true},
		{"root", "/", true},
		{"depth 1", "/foo", true},
		{"depth 2 is valid", "/foo/bar", false},
		{"real temp subdir", validPath, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRemovePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunUninstallRejectsDangerousKhunHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix-style paths")
	}
	removed := false
	remove := func(string) error { removed = true; return nil }

	for _, dangerousPath := range []string{"/", "/home", "relative/path"} {
		removed = false
		err := runUninstall(uninstallOptions{
			yes:        true,
			userHome:   t.TempDir(),
			khunHome:   dangerousPath,
			configPath: filepath.Join(t.TempDir(), "home", ".khunquant", "config.json"),
			remove:     remove,
		})
		assert.Error(t, err, "expected error for khunHome=%q", dangerousPath)
		assert.False(t, removed, "remove must not be called for khunHome=%q", dangerousPath)
	}
}

func TestRunUninstallRejectsDangerousConfigPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix-style paths")
	}
	home := t.TempDir()
	khunHome := filepath.Join(home, ".khunquant")
	removed := false
	remove := func(string) error { removed = true; return nil }

	// configPath outside khunHome that is too shallow
	err := runUninstall(uninstallOptions{
		yes:        true,
		userHome:   home,
		khunHome:   khunHome,
		configPath: "/danger",
		remove:     remove,
	})
	assert.Error(t, err)
	assert.False(t, removed)
}

func TestDedupeTargetsByPathNotKind(t *testing.T) {
	targets := []uninstallTarget{
		{kind: "binary", path: "/usr/local/bin/khunquant", removeFile: true},
		{kind: "data", path: "/usr/local/bin/khunquant", removeFile: true},
	}
	got := dedupeTargets(targets)
	require.Len(t, got, 1)
	assert.Equal(t, "binary", got[0].kind) // first entry wins
	assert.Equal(t, "/usr/local/bin/khunquant", got[0].path)
}

func TestNewUninstallCommand(t *testing.T) {
	cmd := NewUninstallCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "uninstall", cmd.Use)
	assert.Equal(t, "Remove khunquant, launcher binaries, autostart settings, and local data", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.Flags().Lookup("yes"))
	assert.NotNil(t, cmd.Flags().Lookup("dry-run"))
}

func TestUninstallTargetsIncludeDataBinariesAndAutostart(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	khunHome := filepath.Join(home, ".khunquant")

	targets := uninstallTargets(uninstallOptions{
		userHome:   home,
		khunHome:   khunHome,
		configPath: filepath.Join(khunHome, "config.json"),
	})

	assert.Contains(t, targetPaths(targets), khunHome)
	assert.Contains(t, targetPaths(targets), filepath.Join(home, ".local", "bin", binaryName("khunquant")))
	assert.Contains(t, targetPaths(targets), filepath.Join(home, ".local", "bin", binaryName("khunquant-launcher")))

	switch runtime.GOOS {
	case "darwin":
		assert.Contains(t, targetPaths(targets), filepath.Join(home, "Library", "LaunchAgents", "io.khunquant.launcher.plist"))
	case "linux":
		assert.Contains(t, targetPaths(targets), filepath.Join(home, ".config", "autostart", "khunquant-web.desktop"))
	case "windows":
		assert.Contains(t, targetPaths(targets), `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\KhunQuantLauncher`)
	}
}

func TestUninstallTargetsIncludeExternalConfig(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	khunHome := filepath.Join(home, ".khunquant")
	configPath := filepath.Join(home, "custom-config.json")

	targets := uninstallTargets(uninstallOptions{
		userHome:   home,
		khunHome:   khunHome,
		configPath: configPath,
	})

	assert.Contains(t, targetPaths(targets), configPath)
}

func TestRunUninstallDryRunDoesNotRemove(t *testing.T) {
	called := false
	err := runUninstall(uninstallOptions{
		yes:        true,
		dryRun:     true,
		userHome:   filepath.Join(t.TempDir(), "home"),
		khunHome:   filepath.Join(t.TempDir(), ".khunquant"),
		configPath: filepath.Join(t.TempDir(), ".khunquant", "config.json"),
		remove: func(string) error {
			called = true
			return nil
		},
	})

	require.NoError(t, err)
	assert.False(t, called)
}

func TestRunUninstallAbortsWithoutConfirmation(t *testing.T) {
	called := false
	err := runUninstall(uninstallOptions{
		stdin:      strings.NewReader("n\n"),
		userHome:   filepath.Join(t.TempDir(), "home"),
		khunHome:   filepath.Join(t.TempDir(), ".khunquant"),
		configPath: filepath.Join(t.TempDir(), ".khunquant", "config.json"),
		remove: func(string) error {
			called = true
			return nil
		},
	})

	require.NoError(t, err)
	assert.False(t, called)
}

func TestRunUninstallReturnsRemoveErrors(t *testing.T) {
	removeErr := errors.New("permission denied")
	err := runUninstall(uninstallOptions{
		yes:        true,
		userHome:   filepath.Join(t.TempDir(), "home"),
		khunHome:   filepath.Join(t.TempDir(), ".khunquant"),
		configPath: filepath.Join(t.TempDir(), ".khunquant", "config.json"),
		remove: func(string) error {
			return removeErr
		},
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "permission denied")
}

func targetPaths(targets []uninstallTarget) []string {
	paths := make([]string, 0, len(targets))
	for _, target := range targets {
		paths = append(paths, target.path)
	}
	return paths
}

func binaryName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
