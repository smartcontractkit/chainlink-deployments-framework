package cli

import (
	"testing"

	"github.com/spf13/cobra"

	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func findCmd(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}

	return nil
}

func TestNewEnvCmds_BasicStructure(t *testing.T) {
	t.Parallel()
	var c Commands
	root := c.NewEnvCmds(cldf_domain.NewDomain("/tmp", "test-domain"))
	if root == nil {
		t.Fatal("NewEnvCmds returned nil")
	}
	if root.Use != "env" {
		t.Errorf("expected root.Use == \"env\", got %q", root.Use)
	}
	if root.Short != "Env commands" {
		t.Errorf("expected root.Short == \"Env commands\", got %q", root.Short)
	}
	if f := root.PersistentFlags().Lookup("environment"); f == nil {
		t.Fatal("persistent flag \"environment\" not found")
	}
	// top‐level subcommands: load, secrets
	for _, name := range []string{"load", "secrets"} {
		if findCmd(root, name) == nil {
			t.Errorf("subcommand %q not present", name)
		}
	}

	load := findCmd(root, "load")
	if load == nil {
		t.Fatal("load missing")
	}
	if len(load.Commands()) != 0 {
		t.Errorf("expected load to have no sub-commands, got %d", len(load.Commands()))
	}

	// secrets parent
	sec := findCmd(root, "secrets")
	if sec == nil {
		t.Fatal("secrets subcommand missing")
	}
	if sec.Short != "Secrets operations (OCR, TOML, ...)" {
		t.Errorf("expected secrets.Short == \"Secrets operations (OCR, TOML, ...)\", got %q", sec.Short)
	}

	// secrets → ocr & toml
	for _, name := range []string{"ocr"} {
		if findCmd(sec, name) == nil {
			t.Errorf("secrets command missing child %q", name)
		}
	}

	// ocr group
	ocr := findCmd(sec, "ocr")
	if ocr.Short != "OCR-specific secrets" {
		t.Errorf("expected ocr.Short == \"OCR-specific secrets\", got %q", ocr.Short)
	}
	if len(ocr.Commands()) != 1 || ocr.Commands()[0].Name() != "generate" {
		t.Errorf("expected ocr to have one child named \"generate\", got %v", ocr.Commands())
	}
}

func TestEnvOCRSecretGenerate_Command(t *testing.T) {
	t.Parallel()
	cmd := Commands{}.newEnvOCRSecretGenerate()
	if cmd == nil {
		t.Fatal("newEnvOCRSecretGenerate returned nil")
	}
	if cmd.Use != "generate" {
		t.Errorf("expected Use == \"generate\", got %q", cmd.Use)
	}
	if cmd.Short != "Generate OCR secrets" {
		t.Errorf("expected Short == \"Generate OCR secrets\", got %q", cmd.Short)
	}

	for _, name := range []string{"signers", "proposers"} {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag %q not found", name)
			continue
		}
		if f.Value.String() != "false" {
			t.Errorf("expected default of %q == false, got %q", name, f.Value.String())
		}
	}
	if cmd.Long != envOcrSecretsGenerateLong {
		t.Error("LongDesc for generate OCR secrets not wired up correctly")
	}
	if cmd.Example != envOcrSecretsGenerateExample {
		t.Error("Example for generate OCR secrets not wired up correctly")
	}
}

func TestEnvLoad_Command(t *testing.T) {
	t.Parallel()
	domain := cldf_domain.NewDomain("/tmp", "test-domain")
	cmd := Commands{}.newEnvLoad(domain)
	if cmd == nil {
		t.Fatal("newEnvLoad returned nil")
	}
	if cmd.Use != "load" {
		t.Errorf("expected Use == \"load\", got %q", cmd.Use)
	}
	if cmd.Short != "Runs load environment sanity check" {
		t.Errorf("expected Short == \"Runs load environment sanity check\", got %q", cmd.Short)
	}
	if cmd.Long != envLoadLong {
		t.Error("LongDesc for load not wired up correctly")
	}
	if cmd.Example != envLoadExample {
		t.Error("Example for load not wired up correctly")
	}
	if cmd.Flags().HasFlags() {
		t.Error("expected no local flags on load command")
	}
}
