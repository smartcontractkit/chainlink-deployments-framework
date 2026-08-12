package template

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

// commentProvider returns doc-comment lines for struct types and their fields.
// Implementations must be safe for concurrent use. A nil return (or nil slice)
// means no comments are available — callers should treat this as a no-op.
type commentProvider interface {
	// StructComments returns the doc-comment lines above a struct type
	// declaration (e.g. "// MyChangeset deploys ...").
	StructComments(t reflect.Type) []string

	// FieldComments returns the doc-comment lines above a named field of a
	// struct type (e.g. "// ChainSelector is the EVM chain to deploy to.").
	FieldComments(t reflect.Type, goFieldName string) []string
}

// structCommentData holds the extracted doc-comment lines for a single Go
// struct type: the struct-level doc comment (above the type declaration) and
// per-field doc comments keyed by the Go field name (not the yaml/json tag).
type structCommentData struct {
	structComments []string
	fieldComments  map[string][]string
}

// pkgComments holds all extracted struct comment data for a single Go package,
// keyed by the Go type name (e.g. "InputStruct", "MyConfig").
type pkgComments struct {
	structs map[string]*structCommentData
}

// commentExtractor implements commentProvider by parsing Go source files with
// golang.org/x/tools/go/packages to read // doc comments above struct fields.
//
// It caches results per package path so that repeated lookups for types in the
// same package only trigger one packages.Load call. All errors are swallowed —
// comment extraction is a best-effort enhancement and must never cause the
// template-input command to fail.
type commentExtractor struct {
	mu    sync.RWMutex
	cache map[string]*pkgComments
}

// newCommentExtractor returns a ready-to-use commentExtractor.
func newCommentExtractor() *commentExtractor {
	return &commentExtractor{
		cache: make(map[string]*pkgComments),
	}
}

// StructComments returns the doc-comment lines above the given struct type
// declaration, or nil if unavailable. Pointer types are dereferenced to their
// element type before lookup.
func (e *commentExtractor) StructComments(t reflect.Type) []string {
	if t == nil {
		return nil
	}

	// Dereference pointer types (e.g. *fixtureChangeset → fixtureChangeset).
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Name() == "" || t.PkgPath() == "" {
		return nil
	}

	pkgData := e.getPkgComments(t.PkgPath())
	if pkgData == nil {
		return nil
	}

	structData, ok := pkgData.structs[t.Name()]
	if !ok {
		return nil
	}

	return structData.structComments
}

// FieldComments returns the doc-comment lines for the given field on the given
// struct type, or nil if the type's package could not be loaded, the struct is
// not found, or the field has no doc comment. Pointer types are dereferenced
// to their element type before lookup, consistent with StructComments.
func (e *commentExtractor) FieldComments(t reflect.Type, goFieldName string) []string {
	if t == nil || goFieldName == "" {
		return nil
	}

	// Dereference pointer types (e.g. *MyStruct → MyStruct).
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Only named struct types that belong to a real package have source files
	// we can parse. Anonymous structs, primitives, etc. have no comments.
	if t.Name() == "" || t.PkgPath() == "" {
		return nil
	}

	pkgData := e.getPkgComments(t.PkgPath())
	if pkgData == nil {
		return nil
	}

	structData, ok := pkgData.structs[t.Name()]
	if !ok {
		return nil
	}

	return structData.fieldComments[goFieldName]
}

// getPkgComments returns the cached pkgComments for the given package path,
// loading it via packages.Load on first access. Returns nil on any error.
func (e *commentExtractor) getPkgComments(pkgPath string) *pkgComments {
	// Fast path: read lock for cached entries.
	e.mu.RLock()
	if data, ok := e.cache[pkgPath]; ok {
		e.mu.RUnlock()
		return data
	}
	e.mu.RUnlock()

	// Slow path: load and cache with a write lock.
	e.mu.Lock()
	defer e.mu.Unlock()

	// Double-check after acquiring write lock.
	if data, ok := e.cache[pkgPath]; ok {
		return data
	}

	data := e.loadPackage(pkgPath)
	e.cache[pkgPath] = data // may be nil on error — cached to avoid retrying

	return data
}

// loadPackage uses packages.Load to find the Go source files for the given
// package, then manually parses each file with parser.ParseComments to extract
// struct field doc comments. Returns nil on any error.
//
// We use NeedFiles (not NeedSyntax) because packages.Load does not guarantee
// that comments are retained in the pre-parsed syntax trees. By re-parsing the
// files ourselves with parser.ParseComments, we ensure doc comments are
// available in the AST.
func (e *commentExtractor) loadPackage(pkgPath string) *pkgComments {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles,
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil
	}

	result := &pkgComments{
		structs: make(map[string]*structCommentData),
	}

	fset := token.NewFileSet()
	for _, pkg := range pkgs {
		for _, filePath := range pkg.GoFiles {
			file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
			if err != nil {
				continue
			}

			extractStructComments(file, result)
		}
	}

	if len(result.structs) == 0 {
		return nil
	}

	return result
}

// extractStructComments walks an AST file and populates result with doc-comment
// lines for every field of every named struct type declaration.
func extractStructComments(file *ast.File, result *pkgComments) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name == nil {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok || structType.Fields == nil {
				continue
			}

			structData := &structCommentData{
				fieldComments: make(map[string][]string),
			}

			// Extract the struct-level doc comment (the // block above the
			// type declaration, e.g. "// MyChangeset deploys ...").
			// The doc comment can be on either typeSpec.Doc or genDecl.Doc
			// depending on whether the type is declared alone or grouped.
			var doc *ast.CommentGroup
			if typeSpec.Doc != nil {
				doc = typeSpec.Doc
			} else if len(genDecl.Specs) == 1 && genDecl.Doc != nil {
				doc = genDecl.Doc
			}
			if doc != nil {
				structData.structComments = splitCommentGroup(doc)
			}

			for _, field := range structType.Fields.List {
				// Embedded fields have no names — skip them.
				if len(field.Names) == 0 {
					continue
				}

				var comments []string

				// Doc comment group: the // comment block directly above the field.
				if field.Doc != nil {
					comments = append(comments, splitCommentGroup(field.Doc)...)
				}

				// Line comment: a trailing // comment on the same line as the field.
				if field.Comment != nil {
					comments = append(comments, splitCommentGroup(field.Comment)...)
				}

				if len(comments) == 0 {
					continue
				}

				// A single ast.Field can declare multiple names (e.g. `a, b int`),
				// so apply the same comments to each named field.
				for _, name := range field.Names {
					structData.fieldComments[name.Name] = comments
				}
			}

			result.structs[typeSpec.Name.Name] = structData
		}
	}
}

// splitCommentGroup converts an ast.CommentGroup into a slice of individual
// comment lines, with the leading "//" markers and surrounding whitespace
// stripped. Empty lines are removed. Block comments (/* ... */) are split
// into individual lines with leading "*" prefixes stripped.
func splitCommentGroup(group *ast.CommentGroup) []string {
	if group == nil {
		return nil
	}

	var lines []string
	for _, comment := range group.List {
		text := comment.Text
		isBlock := false

		// Strip the "//" prefix (single-line comments).
		if strings.HasPrefix(text, "//") {
			text = strings.TrimPrefix(text, "//")
		} else if strings.HasPrefix(text, "/*") && strings.HasSuffix(text, "*/") {
			// Block comment: strip /* and */ delimiters.
			text = strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
			isBlock = true
		}

		// Trim a single leading space that Go convention adds after "//".
		text = strings.TrimPrefix(text, " ")

		// Split multi-line comments into individual lines.
		for _, line := range strings.Split(text, "\n") {
			trimmed := strings.TrimSpace(line)

			// In block comments, strip a leading "*" that is commonly used
			// as a line prefix (e.g. " * This is a line").
			if isBlock && strings.HasPrefix(trimmed, "*") {
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "*"))
			}

			if trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
	}

	return lines
}
