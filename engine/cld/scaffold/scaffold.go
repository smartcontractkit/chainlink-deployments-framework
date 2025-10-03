package scaffold

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	enginedomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

//go:embed templates/*
var templates embed.FS

// ScaffoldOptions holds configuration options for domain scaffolding.
type ScaffoldOptions struct {
	runModTidy bool
}

// ScaffoldOption is a functional option type for configuring domain scaffolding.
type ScaffoldOption func(*ScaffoldOptions)

// WithModTidy enables running 'go mod tidy' after scaffolding.
// This will resolve and download all dependencies for the new domain.
func WithModTidy() ScaffoldOption {
	return func(opts *ScaffoldOptions) {
		opts.runModTidy = true
	}
}

// defaultScaffoldOptions returns the default configuration for domain scaffolding.
func defaultScaffoldOptions() *ScaffoldOptions {
	return &ScaffoldOptions{
		runModTidy: false, // Don't run go mod tidy by default
	}
}

// getRepositoryName extracts the repository name from the root directory path.
func getRepositoryName(rootDir string) string {
	return filepath.Base(filepath.Dir(rootDir))
}

// ScaffoldDomain creates a new domain directory structure within the specified base path.
// Use WithModTidy() option to run 'go mod tidy' after scaffolding.
func ScaffoldDomain(domain enginedomain.Domain, opts ...ScaffoldOption) error {
	// Apply default options and then user-provided options
	options := defaultScaffoldOptions()
	for _, opt := range opts {
		opt(options)
	}

	var err error

	// Check if the directory already exists or if there is an error accessing it
	if err = checkDirExists(domain.DirPath()); err != nil {
		return fmt.Errorf("failed to create %s domain directory: %w", domain.String(), err)
	}

	// Setup all the args for the templates
	renderArgs := map[string]string{
		"package": domain.Key(),
		"repo":    getRepositoryName(domain.RootPath()),
	}

	// Define the structure
	structure := dirFSNode(domain.Key(), []*fsnode{
		fileFSNode("go.mod", withTemplate("go.mod.tmpl")),
		dirFSNode("pkg", []*fsnode{gitKeepFSNode()}),
		dirFSNode(enginedomain.LibDirName, []*fsnode{gitKeepFSNode()}),
		dirFSNode(enginedomain.InternalDirName, []*fsnode{gitKeepFSNode()}),
		dirFSNode(enginedomain.CmdDirName, []*fsnode{
			fileFSNode("main.go", withTemplate("cmd_main.go.tmpl")),
			dirFSNode(enginedomain.InternalDirName, []*fsnode{
				dirFSNode("cli", []*fsnode{
					fileFSNode("app.go", withTemplate("cmd_internal_app.go.tmpl")),
				}),
			}),
		}),
		dirFSNode(".config", []*fsnode{
			fileFSNode("domain.yaml", withTemplate("domain.yaml.tmpl")),
			dirFSNode("networks", []*fsnode{
				fileFSNode("mainnet.yaml", withTemplate("mainnet.yaml.tmpl")),
				fileFSNode("testnet.yaml", withTemplate("testnet.yaml.tmpl")),
			}),
			dirFSNode("local", []*fsnode{gitKeepFSNode()}),
			dirFSNode("ci", []*fsnode{
				fileFSNode("common.env", withTemplate("common.env.tmpl")),
			}),
		}),
	})

	// Scaffold the domain structure
	if err := scaffold(structure, domain.RootPath(), renderArgs); err != nil {
		return err
	}

	// Run go mod tidy in the newly created domain directory if requested
	if options.runModTidy {
		domainPath := domain.DirPath()
		if err := runGoModTidy(domainPath); err != nil {
			return fmt.Errorf("failed to run 'go mod tidy' in %s: %w", domainPath, err)
		}
	}

	return nil
}

// ScaffoldEnvDir creates a new environment directory structure within the specified base path.
func ScaffoldEnvDir(envdir enginedomain.EnvDir) error {
	// Check if the directory already exists or if there is an error accessing it
	if err := checkDirExists(envdir.DirPath()); err != nil {
		return fmt.Errorf("failed to create %s env directory: %w", envdir.String(), err)
	}

	// Setup all the args for the templates
	renderArgs := map[string]string{
		"package": sanitizePackageName(envdir.Key()),
	}

	// Define the structure
	structure := dirFSNode(envdir.Key(), []*fsnode{
		fileFSNode(enginedomain.AddressBookFileName, withTemplate("address_book.json.tmpl")),
		dirFSNode(enginedomain.DatastoreDirName, []*fsnode{
			fileFSNode(enginedomain.AddressRefsFileName, withTemplate("address_refs.json.tmpl")),
			fileFSNode(enginedomain.ChainMetadataFileName, withTemplate("chain_metadata.json.tmpl")),
			fileFSNode(enginedomain.ContractMetadataFileName, withTemplate("contract_metadata.json.tmpl")),
			fileFSNode(enginedomain.EnvMetadataFileName, withTemplate("env_metadata.json.tmpl")),
		}),
		fileFSNode(enginedomain.NodesFileName, withTemplate("nodes.json.tmpl")),
		fileFSNode(enginedomain.ViewStateFileName, withTemplate("view_state.json.tmpl")),
		fileFSNode(enginedomain.PipelinesFileName, withTemplate("pipelines.go.tmpl")),
		fileFSNode("pipelines_test.go", withTemplate("pipelines_test.go.tmpl")),
		dirFSNode(enginedomain.ArtifactsDirName, []*fsnode{gitKeepFSNode()}),
		dirFSNode(enginedomain.ProposalsDirName, []*fsnode{gitKeepFSNode()}),
		dirFSNode(enginedomain.ArchivedProposalsDirName, []*fsnode{gitKeepFSNode()}),
		dirFSNode(enginedomain.DecodedProposalsDirName, []*fsnode{gitKeepFSNode()}),
		dirFSNode(enginedomain.OperationsReportsDirName, []*fsnode{gitKeepFSNode()}),
		dirFSNode(enginedomain.DurablePipelineDirName, []*fsnode{
			gitKeepFSNode(),
			dirFSNode(enginedomain.DurablePipelineInputsDirName, []*fsnode{gitKeepFSNode()}),
		}),
	})

	return scaffold(structure, envdir.DomainDirPath(), renderArgs)
}

// fsnode represents a file system node, which can be either a directory or a file.
type fsnode struct {
	// name is the name of the file or directory.
	name string
	// isDir indicates whether this node is a directory or a file.
	isDir bool
	// children contains the child nodes of this node. It is only used if this node is a directory.
	children []*fsnode
	// templateName is the name of the template file that will be used to render the file. It is
	// only used if this node is a file.
	templateName string
}

// scaffold walks the file system tree starting from the given root node and creates the
// corresponding directories and files. It uses the provided basePath as the root directory
// for the scaffolded structure. The renderArgs map is used to pass arguments to the template
// rendering process for files that have a template associated with them.
//
// If an error occurs during the scaffolding process, the created directories and files will
// be cleaned up to avoid leaving behind any partially created files or directories.
func scaffold(root *fsnode, basePath string, renderArgs map[string]string) error {
	var err error

	currentPath := filepath.Join(basePath, root.name)

	// Clean up the directory if an error occurs
	defer func() {
		if err != nil {
			os.RemoveAll(currentPath)
		}
	}()

	if root.isDir {
		if err = os.MkdirAll(currentPath, os.ModePerm); err != nil {
			return err
		}

		for _, child := range root.children {
			if err := scaffold(child, currentPath, renderArgs); err != nil {
				return err
			}
		}
	} else {
		file, err := os.Create(currentPath)
		if err != nil {
			return err
		}
		defer file.Close()

		if root.templateName != "" {
			content, err := renderTemplate(root.templateName, renderArgs)
			if err != nil {
				return err
			}

			if _, err := file.WriteString(content); err != nil {
				return err
			}
		}
	}

	return nil
}

// dirFSNode constructs a directory fsnode with the given name and children.
func dirFSNode(name string, children []*fsnode) *fsnode {
	return &fsnode{
		name:     name,
		isDir:    true,
		children: children,
	}
}

// fileFSNode constructs a file fsnode with the given name.
func fileFSNode(name string, opts ...func(*fsnode)) *fsnode {
	n := &fsnode{
		name:  name,
		isDir: false,
	}

	for _, opt := range opts {
		opt(n)
	}

	return n
}

// withTemplate sets the template name for the fsnode.
func withTemplate(templateName string) func(*fsnode) {
	return func(n *fsnode) {
		n.templateName = templateName
	}
}

// gitKeepFSNode constructs a .gitkeep file fsnode to keep empty directories in git. Useful since
// we have many empty directories in the scaffolded structures.
func gitKeepFSNode() *fsnode {
	return fileFSNode(".gitkeep")
}

// renderTemplate renders a template with the given name and arguments.
func renderTemplate(name string, renderArgs any) (string, error) {
	tmpls, err := template.New("").ParseFS(templates, "templates/*")
	if err != nil {
		return "", err
	}

	b := &strings.Builder{}
	if err = tmpls.ExecuteTemplate(b, name, renderArgs); err != nil {
		return "", err
	}

	return b.String(), nil
}

// checkDirExists checks if the domain directory exists, returning os.ErrExist or any access error.
func checkDirExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return os.ErrExist
	}

	if !os.IsNotExist(err) {
		return err
	}

	return nil
}

// runGoModTidy runs 'go mod tidy' in the specified directory.
func runGoModTidy(dir string) error {
	cmd := exec.CommandContext(context.TODO(), "go", "mod", "tidy")
	cmd.Dir = dir

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// sanitizePackageName converts a string to a valid Go package name by replacing hyphens with underscores.
// Go package names cannot contain hyphens, so this function ensures compatibility.
func sanitizePackageName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}
