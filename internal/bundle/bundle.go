package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/parser"
)

type bundledModule struct {
	path string
	deps []moduleDependency
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

	var walk func(path string) error
	walk = func(path string) error {
		if _, ok := seen[path]; ok {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mod := &bundledModule{path: path}
		seen[path] = mod
		for _, spec := range findRequires(string(src)) {
			resolved, err := resolver.Resolve(spec, module.ResolveOptions{
				ProjectRoot: projectRoot,
				BaseDir:     filepath.Dir(path),
				Referrer:    path,
			})
			if err != nil {
				return err
			}
			mod.deps = append(mod.deps, moduleDependency{spec: spec, resolved: resolved})
			if !resolved.External && resolved.Path != "" {
				if err := walk(resolved.Path); err != nil {
					return err
				}
			}
		}
		order = append(order, path)
		return nil
	}

	if err := walk(entryResolved.Path); err != nil {
		return "", err
	}

	// Build module name map
	modNames := make(map[string]string)
	for i, p := range order {
		name := fmt.Sprintf("__mod_%d_%s", i, sanitize(filepath.Base(p)))
		modNames[p] = name
	}

	var b strings.Builder
	b.WriteString("// GoScript bundle\n\n")

	// Generate module functions
	for _, p := range order {
		src, _ := os.ReadFile(p)
		name := modNames[p]

		// Build require() replacement for dependencies
		rewritten := string(src)
		for _, dep := range seen[p].deps {
			if dep.resolved.External {
				continue
			}
			depName := modNames[dep.resolved.Path]
			rewritten = strings.ReplaceAll(rewritten, fmt.Sprintf("require(%q)", dep.spec), depName+"_exports")
		}

		fmt.Fprintf(&b, "// %s\n", filepath.Base(p))
		fmt.Fprintf(&b, "var %s_exports = {};\n", name)

		// Call dependency modules
		for _, dep := range seen[p].deps {
			if dep.resolved.External {
				continue
			}
			depName := modNames[dep.resolved.Path]
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
	entryName := modNames[entryResolved.Path]
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

func findRequires(src string) []string {
	l := lexer.New(src)
	p := parser.New(l, "<bundle>")
	prog := p.ParseProgram()
	return collectRequires(prog.Body)
}

func collectRequires(stmts []ast.Statement) []string {
	var paths []string
	for _, stmt := range stmts {
		paths = append(paths, walkNodeForRequire(stmt)...)
	}
	return unique(paths)
}

func walkNodeForRequire(node ast.Node) []string {
	var paths []string
	switch n := node.(type) {
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
			paths = append(paths, walkNodeForRequire(v)...)
		}
	case *ast.ExprStmt:
		paths = append(paths, walkNodeForRequire(n.Expr)...)
	case *ast.CallExpr:
		if ident, ok := n.Callee.(*ast.Ident); ok && ident.TokenLit == "require" {
			if len(n.Args) > 0 {
				if str, ok := n.Args[0].(*ast.StringLit); ok {
					lit := str.TokenLit
					if len(lit) >= 2 {
						paths = append(paths, lit[1:len(lit)-1])
					}
				}
			}
		}
		for _, a := range n.Args {
			paths = append(paths, walkNodeForRequire(a)...)
		}
	case *ast.InfixExpr:
		paths = append(paths, walkNodeForRequire(n.Left)...)
		paths = append(paths, walkNodeForRequire(n.Right)...)
	case *ast.BlockStmt:
		paths = append(paths, collectRequires(n.Statements)...)
	case *ast.IfStmt:
		paths = append(paths, walkNodeForRequire(n.Cond)...)
		paths = append(paths, collectRequires(n.Consequence.Statements)...)
		if n.Alternative != nil {
			paths = append(paths, walkNodeForRequire(n.Alternative)...)
		}
	case *ast.ReturnStmt:
		if n.Value != nil {
			paths = append(paths, walkNodeForRequire(n.Value)...)
		}
	case *ast.FuncDecl:
		paths = append(paths, collectRequires(n.Body.Statements)...)
	}
	return paths
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
