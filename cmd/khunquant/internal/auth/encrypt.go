package auth

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cryptoquantumwave/khunquant/cmd/khunquant/internal"
	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/credential"
	"github.com/cryptoquantumwave/khunquant/pkg/fileutil"
)

func newEncryptCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "encrypt",
		Short: "Encrypt plaintext credentials in .security.yml with a passphrase",
		Long: `Prompts for a passphrase, generates an SSH key if one does not exist,
then re-saves .security.yml with all SecureString values encrypted as enc:// blobs.

The passphrase is persisted to $KHUNQUANT_HOME/.passphrase (0600) so future
khunquant invocations decrypt automatically. KHUNQUANT_KEY_PASSPHRASE env var
still takes precedence when set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEncrypt()
		},
	}
}

func runEncrypt() error {
	configPath := internal.GetConfigPath()

	encrypted, err := securityConfigHasEncryptedCredentials(configPath)
	if err != nil {
		return err
	}
	if encrypted {
		fmt.Println("Existing encrypted credentials found.")
		fmt.Println("Enter the OLD passphrase and OLD Ed25519 private key path to unlock them before rotation.")
		if err := promptAndInstallOldCredentialInputs(os.Stdin); err != nil {
			return err
		}
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("cannot load config from %s: %w", configPath, err)
	}

	// Check if already encrypted by seeing whether PassphraseProvider returns a value
	// and the file has enc:// blobs. Simple heuristic: ask user to confirm rotation.
	if credential.PassphraseProvider() != "" {
		fmt.Println("Credentials appear to already be encrypted.")
		fmt.Print("Re-encrypt with a new passphrase? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Println("\nSet up credential encryption")
	fmt.Println("-----------------------------")
	passphrase, err := credential.PromptPassphrase()
	if err != nil {
		return fmt.Errorf("passphrase: %w", err)
	}

	if !encrypted {
		if err := credential.SetupSSHKey(); err != nil {
			return fmt.Errorf("SSH key setup: %w", err)
		}
	}

	// Wire passphrase for this process so MarshalYAML encrypts on save without
	// exposing the secret through the process environment.
	credential.PassphraseProvider = func() string { return passphrase }

	rotated := cfg.PrepareEncryptedCredentialsForRotation()
	if rotated > 0 {
		if err := backupSecurityConfig(configPath); err != nil {
			return fmt.Errorf("backup security config: %w", err)
		}
	}

	// Re-save — SecureString.MarshalYAML sees a non-empty PassphraseProvider() and
	// encrypts every field automatically (pkg/config/config_struct.go:158-180).
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving encrypted config: %w", err)
	}

	if err := credential.SavePassphraseFile(passphrase); err != nil {
		fmt.Printf("Warning: could not save passphrase file: %v\n", err)
	}

	passphraseFile := credential.PassphraseFilePath()
	fmt.Println("\nCredentials encrypted successfully.")
	if rotated > 0 {
		fmt.Printf("Re-encrypted %d existing encrypted credential(s).\n", rotated)
	}
	fmt.Printf("Passphrase saved to %s\n", passphraseFile)
	fmt.Println("\nFuture khunquant invocations will decrypt automatically.")
	fmt.Printf("To override, set:  export %s=<passphrase>\n", credential.PassphraseEnvVar)

	return nil
}

func securityConfigHasEncryptedCredentials(configPath string) (bool, error) {
	securityPath := filepath.Join(filepath.Dir(configPath), config.SecurityConfigFile)
	data, err := os.ReadFile(securityPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("cannot read security config from %s: %w", securityPath, err)
	}
	return strings.Contains(string(data), credential.EncScheme), nil
}

func promptAndInstallOldCredentialInputs(stdin *os.File) error {
	oldPassphrase, err := credential.PromptPassphraseWithLabel("Old encryption passphrase", false)
	if err != nil {
		return fmt.Errorf("old passphrase: %w", err)
	}

	fmt.Print("Old Ed25519 private key path: ")
	oldSSHKeyPath, err := readOldSSHKeyPath(bufio.NewReader(stdin))
	if err != nil {
		return err
	}
	if _, err := os.Stat(oldSSHKeyPath); err != nil {
		return fmt.Errorf("old Ed25519 private key path %q is not readable: %w", oldSSHKeyPath, err)
	}

	credential.PassphraseProvider = func() string { return oldPassphrase }
	os.Setenv(credential.SSHKeyPathEnvVar, oldSSHKeyPath)
	return nil
}

func readOldSSHKeyPath(reader *bufio.Reader) (string, error) {
	oldSSHKeyPath, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading old Ed25519 private key path: %w", err)
	}
	oldSSHKeyPath = strings.TrimSpace(oldSSHKeyPath)
	if oldSSHKeyPath == "" {
		return "", fmt.Errorf("old Ed25519 private key path must not be empty")
	}
	return oldSSHKeyPath, nil
}

func backupSecurityConfig(configPath string) error {
	securityPath := filepath.Join(filepath.Dir(configPath), config.SecurityConfigFile)
	if _, err := os.Stat(securityPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return fileutil.CopyFile(securityPath, securityPath+".bak", 0o600)
}
