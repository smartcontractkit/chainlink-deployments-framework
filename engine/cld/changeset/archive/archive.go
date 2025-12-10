package archive

import (
	"context"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// Archivist is responsible for managing the archival of migration keys in a Go source file.
// It provides functionality to locate the migration file, parse its abstract syntax tree (AST),
// and remove specified migration keys and their associated registry.Add calls. The Archivist
// ensures that the migration keys are properly archived and removed from the migration file,
// maintaining the integrity of the migration process.
type Archivist struct {
	envdir              domain.EnvDir
	mainBranchSHAGetter GitSHAGetter
}

// NewArchivist creates a new Archivist instance with the provided EnvDir.
func NewArchivist(envdir domain.EnvDir) Archivist {
	return Archivist{
		envdir:              envdir,
		mainBranchSHAGetter: MainBranchSHAGetter{},
	}
}

// Archive parses the migrations.go and migrations_archive.go abstract syntax tree (AST), to remove
// specified migration keys and their associated registry.Add calls and adds Archive calls for the
// provided keys.
func (a Archivist) Archive(migKeys ...string) (archivalReport, error) {
	// Parse the migration migFile and create an AST
	migFilepath, err := a.migrationFilePath()
	if err != nil {
		return nil, err
	}

	migFset := token.NewFileSet()
	migFile, err := parser.ParseFile(migFset, migFilepath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Parse the migration archive file and create an AST
	archiveFilepath, err := a.archiveFilePath()
	if err != nil {
		return nil, err
	}

	archiveFset := token.NewFileSet()
	archiveFile, err := parser.ParseFile(archiveFset, archiveFilepath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Prepare the migration file for archival, generating a report
	report := a.prepare(migFile, migKeys)

	// Remove the migrations from the migration file
	a.removeMigrations(migFile, report)

	// Archive the migrations in the archive file
	if err := a.insertArchiveCalls(archiveFile, report); err != nil {
		return report, err
	}

	// Save the changes to the migration and archive files
	if err := saveASTToFile(migFilepath, migFset, migFile); err != nil {
		return report, err
	}

	if err := saveASTToFile(archiveFilepath, archiveFset, archiveFile); err != nil {
		return report, err
	}

	return report, nil
}

// migrationFilePath returns the path to the migrations.go file.
func (a Archivist) migrationFilePath() (string, error) {
	path := a.envdir.MigrationsFilePath()

	// Check the file exists
	_, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	return path, nil
}

// archiveFilePath returns the path to the migrations_archive.go file.
func (a Archivist) archiveFilePath() (string, error) {
	path := a.envdir.MigrationsArchiveFilePath()

	// Check the file exists
	if _, err := os.Stat(path); err != nil {
		return "", err
	}

	return path, nil
}

// prepare inspects the AST to retrieve data about the migrations for archival. Returns a map of
// migration keys to the corresponding constant names.
func (Archivist) prepare(f *ast.File, migKeys []string) archivalReport {
	report := make(archivalReport, len(migKeys))
	for _, key := range migKeys {
		report[key] = archivalReportEntry{}
	}

	ast.Inspect(f, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Tok == token.CONST {
				for _, spec := range n.Specs {
					vs := spec.(*ast.ValueSpec)

					for i, val := range vs.Values {
						if basicLit, ok := val.(*ast.BasicLit); ok {
							if basicLitStrContains(migKeys, basicLit) {
								key := strings.Trim(basicLit.Value, "\"")

								report.setConstantIfExists(key, vs.Names[i].Name)
							}
						}
					}
				}
			}
		}

		return true
	})

	return report
}

// archivalReport is a map of migration keys to the current status of the archival process for each
// migration key.
type archivalReport map[string]archivalReportEntry

// setDeletedIfExists sets the isDeleted flag for the migration key in the report.
func (r archivalReport) setDeletedIfExists(key string) {
	if entry, ok := r[key]; ok {
		entry.isDeleted = true

		r[key] = entry
	}
}

// setConstantIfExists sets the constant name for the migration key in the report.
func (r archivalReport) setConstantIfExists(key, constant string) {
	if entry, ok := r[key]; ok {
		entry.constant = &constant
		r[key] = entry
	}
}

// archivalReportEntry contains the data for a migration key in the archival report.
type archivalReportEntry struct {
	// constant is the name of the constant that has the value of the migration key. This may be
	// empty if the constant was not found in the migration.
	constant *string
	// isDeleted indicates whether the migration key was deleted from the migration file.
	isDeleted bool
}

// removeMigrations removes the migration constants (if they exist) and the registry.Add calls
// for the migrations in migKeys.
func (Archivist) removeMigrations(f *ast.File, report archivalReport) {
	ast.Inspect(f, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Tok == token.CONST {
				removeMigrationKeyConsts(n, report)
			}
		case *ast.FuncDecl:
			rkeys := removeMigrationAddCalls(n, report)

			for _, rkey := range rkeys {
				report.setDeletedIfExists(rkey)
			}
		}

		return true
	})
}

// insertArchiveCalls inserts the Archive calls for the migration keys in the report that have been
// deleted.
func (a Archivist) insertArchiveCalls(f *ast.File, report archivalReport) error {
	commitSHA, err := a.mainBranchSHAGetter.Get()
	if err != nil {
		return err
	}

	ast.Inspect(f, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			// 'archive' only matches legacy calls and can be removed once domains have been
			// migrated to use the Archive method.
			if n.Name.Name == "archive" || n.Name.Name == "Archive" {
				for _, k := range slices.Collect(maps.Keys(report)) {
					if !report[k].isDeleted {
						continue
					}

					archiveCall := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("registry"),
								Sel: ast.NewIdent("Archive"),
							},
							Args: []ast.Expr{
								ast.NewIdent(strconv.Quote(k)),
								ast.NewIdent(strconv.Quote(commitSHA)),
							},
						},
					}

					n.Body.List = append(n.Body.List, archiveCall)
				}
			}
		}

		return true
	})

	return nil
}

// removeMigrationKeyConsts removes the migration key constants which contain a value matching
// one of the strings in migKeys.
func removeMigrationKeyConsts(d *ast.GenDecl, report archivalReport) {
	d.Specs = slices.DeleteFunc(d.Specs, func(spec ast.Spec) bool {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			return false
		}

		return vsValueContains(slices.Collect(maps.Keys(report)), valueSpec)
	})
}

// removeMigrationAddCalls removes the registry.Add calls in the registry.Init function which
// contain an argument value matching one of the strings in migKeys. If the argument is an
// identifier, it is removed if its name matches one of the values in migKeys, otherwise if it is
// a basic literal, it is removed if its value matches one of the keys in migKeys.
func removeMigrationAddCalls(d *ast.FuncDecl, report archivalReport) []string {
	removedKeys := make([]string, 0)

	// Add calls are only in the Init function
	if d.Name.Name != "Init" {
		return removedKeys
	}

	// Iterate over all the statements in the function, adding only the ones that are not
	// registry.Add calls which match the migration
	d.Body.List = slices.DeleteFunc(d.Body.List, func(stmt ast.Stmt) bool {
		callExpr, mthd, err := getCallSelectorExpr(stmt)
		if err != nil {
			return false
		}

		ident, ok := mthd.X.(*ast.Ident)
		if !ok {
			return false
		}

		if ident.Name != "registry" || mthd.Sel.Name != "Add" || len(callExpr.Args) != 2 {
			return false
		}

		switch v := callExpr.Args[0].(type) {
		// When the argument is an identifier, check if it is a migration key matches one of the constants
		case *ast.Ident:
			for k, e := range report {
				if e.constant != nil && *e.constant == v.Name {
					removedKeys = append(removedKeys, k)

					return true
				}
			}

			return false
		// When the argument is a string, check if it is a migration key
		case *ast.BasicLit:
			if basicLitStrContains(slices.Collect(maps.Keys(report)), v) {
				removedKeys = append(removedKeys, strings.Trim(v.Value, "\""))

				return true
			}

			return false
		}

		return false
	})

	return removedKeys
}

// basicLitStrContains checks if a basic literal contains a string matching one of the strings in ss
func basicLitStrContains(ss []string, b *ast.BasicLit) bool {
	return b.Kind == token.STRING &&
		slices.Contains(ss, strings.Trim(b.Value, "\""))
}

// vsValueContains checks if a value spec contains a string matching one of the strings in ss
func vsValueContains(ss []string, vs *ast.ValueSpec) bool {
	for _, val := range vs.Values {
		if basicLit, ok := val.(*ast.BasicLit); ok {
			if basicLitStrContains(ss, basicLit) {
				return true
			}
		}
	}

	return false
}

// getCallSelectorExpr extracts the call expression and selector expression from a statement.
func getCallSelectorExpr(stmt ast.Stmt) (*ast.CallExpr, *ast.SelectorExpr, error) {
	exprStmt, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return nil, nil, errors.New("stmt is not an expression statement")
	}

	callExpr, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return nil, nil, errors.New("expression statement is not a call expression")
	}

	mthd, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, nil, errors.New("call expression is not a selector expression")
	}

	return callExpr, mthd, nil
}

// saveASTToFile saves the AST to a file at the provided filepath.
func saveASTToFile(fp string, fset *token.FileSet, f *ast.File) error {
	migFileWriter, err := os.OpenFile(fp, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer migFileWriter.Close()

	return format.Node(migFileWriter, fset, f)
}

// GitSHAGetter is an interface for getting a Git SHA.
type GitSHAGetter interface {
	Get() (string, error)
}

// MainBranchSHAGetter is a GitSHAGetter implementation that gets the SHA of
// the main branch.
type MainBranchSHAGetter struct{}

// Get returns the SHA of the main branch
func (m MainBranchSHAGetter) Get() (string, error) {
	cmd := exec.CommandContext(context.TODO(), "git", "rev-parse", "origin/main")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
