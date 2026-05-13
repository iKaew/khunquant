package uninstall

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cryptoquantumwave/khunquant/cmd/khunquant/internal"
)

type removeFunc func(string) error

type uninstallOptions struct {
	yes        bool
	dryRun     bool
	remove     removeFunc
	userHome   string
	khunHome   string
	configPath string
	stdin      io.Reader
}

type uninstallTarget struct {
	path       string
	kind       string
	removeFile bool
}

func NewUninstallCommand() *cobra.Command {
	var yes bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove khunquant, launcher binaries, autostart settings, and local data",
		Long: strings.TrimSpace(`Remove KhunQuant from this machine.

This removes KhunQuant data, installed binaries in common install locations,
and launch-at-login settings created by the web launcher.`),
		Example: strings.TrimSpace(`khunquant uninstall --dry-run
khunquant uninstall --yes`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUninstall(uninstallOptions{
				yes:        yes,
				dryRun:     dryRun,
				remove:     os.RemoveAll,
				khunHome:   internal.GetKhunquantHome(),
				configPath: internal.GetConfigPath(),
				stdin:      os.Stdin,
			})
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be removed without deleting anything")
	return cmd
}

func runUninstall(opts uninstallOptions) error {
	if opts.remove == nil {
		opts.remove = os.RemoveAll
	}
	if opts.khunHome == "" {
		opts.khunHome = internal.GetKhunquantHome()
	}
	if opts.configPath == "" {
		opts.configPath = internal.GetConfigPath()
	}
	if opts.userHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve user home: %w", err)
		}
		opts.userHome = home
	}
	if opts.stdin == nil {
		opts.stdin = os.Stdin
	}

	if err := validateRemovePath(opts.khunHome); err != nil {
		return fmt.Errorf("unsafe data directory: %w", err)
	}
	if opts.configPath != "" && !pathWithin(opts.configPath, opts.khunHome) {
		if err := validateRemovePath(opts.configPath); err != nil {
			return fmt.Errorf("unsafe config path: %w", err)
		}
	}

	targets := uninstallTargets(opts)
	fmt.Println("KhunQuant uninstall will remove:")
	for _, target := range targets {
		fmt.Printf("  - %s: %s\n", target.kind, target.path)
	}

	if opts.dryRun {
		fmt.Println("Dry run only. Nothing was removed.")
		return nil
	}

	if !opts.yes && !confirm(opts.stdin) {
		fmt.Println("Aborted.")
		return nil
	}

	var errs []error
	for _, target := range targets {
		if !target.removeFile {
			continue
		}
		if err := opts.remove(target.path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("%s: %w", target.path, err))
			fmt.Printf("  ! failed: %s (%v)\n", target.path, err)
			continue
		}
		fmt.Printf("  removed: %s\n", target.path)
	}

	if err := removeWindowsAutoStart(); err != nil {
		errs = append(errs, err)
		fmt.Printf("  ! failed: Windows autostart (%v)\n", err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	fmt.Println("KhunQuant uninstall complete.")
	return nil
}

func uninstallTargets(opts uninstallOptions) []uninstallTarget {
	targets := []uninstallTarget{}
	add := func(kind, path string) {
		if path == "" {
			return
		}
		targets = append(targets, uninstallTarget{kind: kind, path: filepath.Clean(path), removeFile: true})
	}
	addDisplayOnly := func(kind, path string) {
		if path == "" {
			return
		}
		targets = append(targets, uninstallTarget{kind: kind, path: path})
	}

	add("data directory", opts.khunHome)
	if !pathWithin(opts.configPath, opts.khunHome) {
		add("config file", opts.configPath)
	}

	add("shell history", filepath.Join(os.TempDir(), ".khunquant_history"))
	add("SSH key", defaultSSHKeyPath(opts.userHome))
	add("SSH public key", defaultSSHKeyPath(opts.userHome)+".pub")

	if envKeyPath := os.Getenv("KHUNQUANT_SSH_KEY_PATH"); envKeyPath != "" && envKeyPath != defaultSSHKeyPath(opts.userHome) {
		add("SSH key", envKeyPath)
		add("SSH public key", envKeyPath+".pub")
	}

	for _, path := range binaryPaths(opts.userHome) {
		add("binary", path)
	}

	switch runtime.GOOS {
	case "darwin":
		add("autostart setting", filepath.Join(opts.userHome, "Library", "LaunchAgents", "io.khunquant.launcher.plist"))
	case "linux":
		add("autostart setting", filepath.Join(opts.userHome, ".config", "autostart", "khunquant-web.desktop"))
	case "windows":
		addDisplayOnly("autostart setting", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\KhunQuantLauncher`)
	}

	return dedupeTargets(targets)
}

func binaryPaths(userHome string) []string {
	names := []string{"khunquant", "khunquant-launcher", "khunquant-launcher-tui"}
	if runtime.GOOS == "windows" {
		for i, name := range names {
			names[i] = name + ".exe"
		}
	}

	dirs := []string{
		"/usr/local/bin",
		filepath.Join(userHome, ".local", "bin"),
		filepath.Join(userHome, "bin"),
	}

	if exePath, err := os.Executable(); err == nil && isKhunquantBinary(filepath.Base(exePath)) && !pathWithin(exePath, os.TempDir()) {
		if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
			exePath = resolved
		}
		dirs = append(dirs, filepath.Dir(exePath))
	}

	paths := make([]string, 0, len(dirs)*len(names))
	for _, dir := range dirs {
		for _, name := range names {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			paths = append(paths, path)
		}
	}
	return paths
}

func isKhunquantBinary(name string) bool {
	switch name {
	case "khunquant", "khunquant.exe", "khunquant-launcher", "khunquant-launcher.exe", "khunquant-launcher-tui", "khunquant-launcher-tui.exe":
		return true
	default:
		return false
	}
}

func defaultSSHKeyPath(userHome string) string {
	return filepath.Join(userHome, ".ssh", "khunquant_ed25519.key")
}

func pathWithin(path, parent string) bool {
	if path == "" || parent == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func dedupeTargets(targets []uninstallTarget) []uninstallTarget {
	seen := map[string]bool{}
	deduped := make([]uninstallTarget, 0, len(targets))
	for _, target := range targets {
		if seen[target.path] {
			continue
		}
		seen[target.path] = true
		deduped = append(deduped, target)
	}
	return deduped
}

func confirm(stdin io.Reader) bool {
	fmt.Print("Continue? This cannot be undone. [y/N] ")
	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes"
}

func removeWindowsAutoStart() error {
	if runtime.GOOS != "windows" {
		return nil
	}
	cmd := exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "KhunQuantLauncher", "/f")
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			// exit code 1 means the registry key was not found
			return nil
		}
		return fmt.Errorf("remove Windows autostart: %w", err)
	}
	return nil
}

// pathDepth counts the number of path components below the filesystem root or drive letter.
// e.g. "/" → 0, "/foo" → 1, "/foo/bar" → 2
func pathDepth(path string) int {
	clean := filepath.Clean(path)
	vol := filepath.VolumeName(clean)
	rest := clean[len(vol):]
	count := 0
	for part := range strings.SplitSeq(rest, string(filepath.Separator)) {
		if part != "" {
			count++
		}
	}
	return count
}

// validateRemovePath returns an error if path is unsafe to pass to os.RemoveAll.
// It requires an absolute path with at least two components below the root.
func validateRemovePath(path string) error {
	if path == "" {
		return errors.New("path is empty")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path %q is not absolute", path)
	}
	if pathDepth(path) < 2 {
		return fmt.Errorf("path %q is too close to filesystem root", path)
	}
	return nil
}
