package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/packagefile"
	"github.com/issueye/goscript/internal/parser"
)

type bundledModule struct {
	resolved module.ResolvedModule
	source   string
	deps     []moduleDependency
}

type moduleDependency struct {
	spec     string
	resolved module.ResolvedModule
}

func Bundle(entry string) (string, error) {
	projectRoot := module.FindProjectRoot(filepath.Dir(entry))
	resolver := module.NewResolver(projectRoot)
	entryResolved, err := resolver.Resolve(entry, module.ResolveOptions{
		ProjectRoot: projectRoot,
		BaseDir:     filepath.Dir(entry),
	})
	if err != nil {
		return "", err
	}
	if entryResolved.External || entryResolved.Path == "" {
		return "", fmt.Errorf("entry %q resolved to external module", entry)
	}

	seen := make(map[string]*bundledModule)
	order := make([]string, 0)

	var walk func(resolved module.ResolvedModule) error
	walk = func(resolved module.ResolvedModule) error {
		id := moduleID(resolved)
		if _, ok := seen[id]; ok {
			return nil
		}

		src, err := readResolvedSource(resolved)
		if err != nil {
			return err
		}
		mod := &bundledModule{resolved: resolved, source: src}
		seen[id] = mod
		for _, spec := range findStaticDependencies(src) {
			resolved, err := resolver.Resolve(spec, module.ResolveOptions{
				ProjectRoot: projectRoot,
				BaseDir:     resolvedModuleDir(mod.resolved),
				Referrer:    mod.resolved.Path,
			})
			if err != nil {
				return err
			}
			mod.deps = append(mod.deps, moduleDependency{spec: spec, resolved: resolved})
			if !resolved.External && resolved.Path != "" {
				if err := walk(resolved); err != nil {
					return err
				}
			}
		}
		order = append(order, id)
		return nil
	}

	if err := walk(entryResolved); err != nil {
		return "", err
	}

	// Build module name map
	modNames := make(map[string]string)
	for i, id := range order {
		name := fmt.Sprintf("__mod_%d_%s", i, sanitize(moduleDisplayName(seen[id].resolved)))
		modNames[id] = name
	}

	var b strings.Builder
	b.WriteString("// GoScript bundle\n\n")

	// Generate module functions
	for _, id := range order {
		mod := seen[id]
		src := mod.source
		name := modNames[id]

		// Build require() replacement for dependencies
		rewritten := src
		for _, dep := range mod.deps {
			if dep.resolved.External {
				continue
			}
			depName := modNames[moduleID(dep.resolved)]
			rewritten = strings.ReplaceAll(rewritten, fmt.Sprintf("require(%q)", dep.spec), depName+"_exports")
		}

		fmt.Fprintf(&b, "// %s\n", moduleDisplayName(mod.resolved))
		fmt.Fprintf(&b, "var %s_exports = {};\n", name)

		// Call dependency modules
		for _, dep := range mod.deps {
			if dep.resolved.External {
				continue
			}
			depName := modNames[moduleID(dep.resolved)]
			fmt.Fprintf(&b, "__run_%s(%s_exports);\n", depName, depName)
		}

		fmt.Fprintf(&b, "function __run_%s(exports) {\n", name)
		for _, line := range strings.Split(rewritten, "\n") {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("}\n")
		fmt.Fprintf(&b, "__run_%s(%s_exports);\n\n", name, name)
	}

	// Call entry
	entryName := modNames[moduleID(entryResolved)]
	fmt.Fprintf(&b, "%s_exports;\n", entryName)

	return b.String(), nil
}

func sanitize(path string) string {
	s := filepath.Base(path)
	s = strings.TrimSuffix(s, ".gs")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func moduleID(resolved module.ResolvedModule) string {
	if resolved.ID != "" {
		return resolved.ID
	}
	return resolved.Path
}

func moduleDisplayName(resolved module.ResolvedModule) string {
	if resolved.PackageFile != "" {
		return filepath.Base(resolved.PackageFile) + "!" + resolved.ArchivePath
	}
	return filepath.Base(resolved.Path)
}

func resolvedModuleDir(resolved module.ResolvedModule) string {
	if resolved.PackageFile != "" {
		return filepath.ToSlash(resolved.PackageFile) + "!" + filepath.ToSlash(filepath.Dir(resolved.ArchivePath))
	}
	return filepath.Dir(resolved.Path)
}

func readResolvedSource(resolved module.ResolvedModule) (string, error) {
	if resolved.PackageFile != "" {
		return packagefile.ReadNestedText(resolved.PackageFile, resolved.ArchivePath)
	}
	data, err := os.ReadFile(resolved.Path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func findStaticDependencies(src string) []string {
	l := lexer.New(src)
	p := parser.New(l, "<bundle>")
	prog := p.ParseProgram()
	return collectDependencies(prog.Body)
}

func collectDependencies(stmts []ast.Statement) []string {
	var paths []string
	for _, stmt := range stmts {
		paths = append(paths, walkNodeForDependency(stmt)...)
	}
	return unique(paths)
}

func walkNodeForDependency(node ast.Node) []string {
	var paths []string
	switch n := node.(type) {
	case *ast.ImportDecl:
		paths = append(paths, unquoteModulePath(n.Source))
	case *ast.LetStmt, *ast.ConstStmt, *ast.VarStmt:
		var v ast.Expression
		switch s := n.(type) {
		case *ast.LetStmt:
			v = s.Value
		case *ast.ConstStmt:
			v = s.Value
		case *ast.VarStmt:
			v = s.Value
		}
		if v != nil {
			paths = append(paths, walkNodeForDependency(v)...)
		}
	case *ast.ExprStmt:
		paths = append(paths, walkNodeForDependency(n.Expr)...)
	case *ast.CallExpr:
		if ident, ok := n.Callee.(*ast.Ident); ok && ident.TokenLit == "require" {
			if len(n.Args) > 0 {
				if str, ok := n.Args[0].(*ast.StringLit); ok {
					paths = append(paths, unquoteModulePath(str.TokenLit))
				}
			}
		}
		for _, a := range n.Args {
			paths = append(paths, walkNodeForDependency(a)...)
		}
	case *ast.InfixExpr:
		paths = append(paths, walkNodeForDependency(n.Left)...)
		paths = append(paths, walkNodeForDependency(n.Right)...)
	case *ast.BlockStmt:
		paths = append(paths, collectDependencies(n.Statements)...)
	case *ast.IfStmt:
		paths = append(paths, walkNodeForDependency(n.Cond)...)
		paths = append(paths, collectDependencies(n.Consequence.Statements)...)
		if n.Alternative != nil {
			paths = append(paths, walkNodeForDependency(n.Alternative)...)
		}
	case *ast.ReturnStmt:
		if n.Value != nil {
			paths = append(paths, walkNodeForDependency(n.Value)...)
		}
	case *ast.FuncDecl:
		paths = append(paths, collectDependencies(n.Body.Statements)...)
	}
	return paths
}

func unquoteModulePath(path string) string {
	if len(path) >= 2 {
		first := path[0]
		last := path[len(path)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return path[1 : len(path)-1]
		}
	}
	return path
}

func unique(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
