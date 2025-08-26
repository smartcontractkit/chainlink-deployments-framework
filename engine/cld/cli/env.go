package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/cosmos/go-bip39"
	"github.com/spf13/cobra"

	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments/pkg/cli"
	"github.com/smartcontractkit/chainlink-deployments/pkg/environment"
)

// NewEnvCmds creates a new set of commands for managing environment.
func (c Commands) NewEnvCmds(
	domain cldf_domain.Domain,
) *cobra.Command {
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Env commands",
	}

	secretsCmd := &cobra.Command{
		Use:   "secrets",
		Short: "Secrets operations (OCR, TOML, ...)",
	}

	ocrCmd := &cobra.Command{
		Use:   "ocr",
		Short: "OCR-specific secrets",
	}
	ocrCmd.AddCommand(c.newEnvOCRSecretGenerate())

	secretsCmd.AddCommand(ocrCmd)
	envCmd.AddCommand(
		c.newEnvLoad(domain),
		secretsCmd,
	)

	envCmd.PersistentFlags().
		StringP("environment", "e", "", "Deployment environment (required)")
	_ = envCmd.MarkPersistentFlagRequired("environment")

	return envCmd
}

var (
	envOcrSecretsGenerateLong = cli.LongDesc(`
	Generates BIP39 OCR secrets for xsigners or xproposers
`)

	envOcrSecretsGenerateExample = cli.Examples(`
  		# Generate both xsigners and xproposers for the staging environment
 	 	exemplar env secrets ocr generate --environment staging

  		# Generate only xsigners
  		exemplar env secrets ocr generate --environment staging --signers

  		# Generate only xproposers
  		exemplar env secrets ocr generate --environment staging --proposers
`)
)

// newEnvOCRSecretGenerate creates a command to generate OCR BIP39 secrets.
func (Commands) newEnvOCRSecretGenerate() *cobra.Command {
	const entropySize = 256
	var (
		signers   bool
		proposers bool
	)
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate OCR secrets",
		Long:    envOcrSecretsGenerateLong,
		Example: envOcrSecretsGenerateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			secrets, err := environment.LoadEnvSecrets(envKey)
			if err != nil && !errors.Is(err, environment.ErrNoOCRSecrets) {
				return err
			}
			if signers {
				entropy, err := bip39.NewEntropy(entropySize)
				if err != nil {
					return err
				}
				mnemonic, err := bip39.NewMnemonic(entropy)
				if err != nil {
					return err
				}
				secrets.OCR.XSigners = mnemonic
			}
			if proposers {
				entropy, err := bip39.NewEntropy(entropySize)
				if err != nil {
					return err
				}
				mnemonic, err := bip39.NewMnemonic(entropy)
				if err != nil {
					return err
				}
				secrets.OCR.XProposers = mnemonic
			}

			return secrets.WriteToFile(envKey)
		},
	}
	cmd.Flags().BoolVar(&signers, "signers", false, "Generate new xsigners (must share with all signers)")
	cmd.Flags().BoolVar(&proposers, "proposers", false, "Generate new xproposers (for proposers only)")

	return cmd
}

var (
	envLoadLong = cli.LongDesc(`
		Runs a sanity check by loading the environment configuration and verifying connectivity.
`)

	envLoadExample = cli.Examples(`
  		# Verify that the staging environment loads correctly
  		exemplar env load --environment staging
`)
)

// newEnvLoad creates the "load" subcommand for environment checks.
func (c Commands) newEnvLoad(domain cldf_domain.Domain) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "load",
		Short:   "Runs load environment sanity check",
		Long:    envLoadLong,
		Example: envLoadExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			_, err := environment.LoadEnvironment(
				func() context.Context { return cmd.Context() },
				c.lggr,
				envKey,
				domain,
				true,
			)
			if err != nil {
				return fmt.Errorf("LoadEnvironment failed: %w", err)
			}
			cmd.Println("✅ Environment loaded successfully.")

			return nil
		},
	}

	return cmd
}
