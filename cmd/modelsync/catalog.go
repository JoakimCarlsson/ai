package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
)

// entry is one model as it exists in a catalog file on disk: the Go constant
// naming it, the constant's string value and every struct field rendered as
// the literal source text it was written with.
type entry struct {
	constName string
	constVal  string
	apiModel  string
	fields    map[string]string
}

// catalog is the parsed state of a models.go file, keyed by APIModel so it can
// be matched against what a provider API returns.
type catalog struct {
	entries map[string]*entry
	names   map[string]bool
}

func newCatalog() *catalog {
	return &catalog{
		entries: make(map[string]*entry),
		names:   make(map[string]bool),
	}
}

// readCatalog parses an existing models.go. A missing file yields an empty
// catalog so a brand new provider can be generated from scratch.
func readCatalog(path string) (*catalog, error) {
	cat := newCatalog()

	src, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cat, nil
	}
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	consts := make(map[string]string)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
				continue
			}
			lit, ok := vs.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			val, err := strconv.Unquote(lit.Value)
			if err != nil {
				continue
			}
			consts[vs.Names[0].Name] = val
			cat.names[vs.Names[0].Name] = true
		}
	}

	models := findModelsLiteral(file)
	if models == nil {
		return cat, nil
	}

	for _, elt := range models.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		lit, ok := kv.Value.(*ast.CompositeLit)
		if !ok {
			continue
		}

		e := &entry{fields: make(map[string]string)}
		if ident, ok := kv.Key.(*ast.Ident); ok {
			e.constName = ident.Name
			e.constVal = consts[ident.Name]
		}

		for _, f := range lit.Elts {
			fkv, ok := f.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			name, ok := fkv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			e.fields[name.Name] = exprText(fset, src, fkv.Value)
		}

		key := unquoteField(e.fields["APIModel"])
		if key == "" {
			key = e.constVal
		}
		if key == "" {
			continue
		}
		e.apiModel = key
		cat.entries[key] = e
	}

	return cat, nil
}

func findModelsLiteral(file *ast.File) *ast.CompositeLit {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "Models" {
				continue
			}
			if len(vs.Values) != 1 {
				continue
			}
			if lit, ok := vs.Values[0].(*ast.CompositeLit); ok {
				return lit
			}
		}
	}
	return nil
}

// exprText returns the source text an expression was written with, so field
// values the API does not describe survive a regeneration byte for byte.
func exprText(fset *token.FileSet, src []byte, e ast.Expr) string {
	start := fset.Position(e.Pos()).Offset
	end := fset.Position(e.End()).Offset
	if start < 0 || end > len(src) || start >= end {
		return ""
	}
	return string(src[start:end])
}

func unquoteField(lit string) string {
	if lit == "" {
		return ""
	}
	val, err := strconv.Unquote(lit)
	if err != nil {
		return ""
	}
	return val
}
