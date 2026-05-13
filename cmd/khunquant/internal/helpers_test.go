package internal

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetKhunquantHomeReturnsAbsolutePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override not reliable on Windows")
	}
	t.Setenv("KHUNQUANT_HOME", "")
	t.Setenv("HOME", "/tmp/testhome")
	got := GetKhunquantHome()
	require.NotEmpty(t, got, "GetKhunquantHome() should not return empty string")
	assert.True(t, filepath.IsAbs(got), "GetKhunquantHome() = %q, want absolute path", got)
}

func TestGetConfigPath(t *testing.T) {
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := filepath.Join("/tmp/home", ".khunquant", "config.json")

	assert.Equal(t, want, got)
}

func TestGetConfigPath_WithKHUNQUANT_HOME(t *testing.T) {
	t.Setenv("KHUNQUANT_HOME", "/custom/khunquant")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := filepath.Join("/custom/khunquant", "config.json")

	assert.Equal(t, want, got)
}

func TestGetConfigPath_WithKHUNQUANT_CONFIG(t *testing.T) {
	t.Setenv("KHUNQUANT_CONFIG", "/custom/config.json")
	t.Setenv("KHUNQUANT_HOME", "/custom/khunquant")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := "/custom/config.json"

	assert.Equal(t, want, got)
}

func TestGetConfigPath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific HOME behavior varies; run on windows")
	}

	testUserProfilePath := `C:\Users\Test`
	t.Setenv("USERPROFILE", testUserProfilePath)

	got := GetConfigPath()
	want := filepath.Join(testUserProfilePath, ".khunquant", "config.json")

	require.True(t, strings.EqualFold(got, want), "GetConfigPath() = %q, want %q", got, want)
}
