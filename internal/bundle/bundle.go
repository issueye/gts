package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/parser"
)

func Bundle(entry string) (string, error) {
	seen := make(map[string]bool)
	order := make([]string, 0)
	baseDir := filepath.Dir(entry)

	var walk func(path string) error
	walk = func(path string) error {
		absPath := resolvePath(path, baseDir)
		if seen[absPath] {
			return nil
		}
		seen[absPath] = true

		src, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		deps := findRequires(string(src))
		for _, dep := range deps {
			depPath := resolvePath(dep, filepath.Dir(absPath))
			if err := walk(depPath); err != nil {
				return err
			}
		}
		order = append(order, absPath)
		return nil
	}

	if err := walk(entry); err != nil {
		return "", err
	}

	// Build module name map
	modNames := make(map[string]string)
	depGraph := make(map[string][]string) // module → its dependencies
	for _, p := range order {
		name := "__mod_" + sanitize(filepath.Base(p))
		modNames[p] = name

		src, _ := os.ReadFile(p)
		deps := findRequires(string(src))
		var depMods []string
		for _, dep := range deps {
			depAbs := resolvePath(dep, filepath.Dir(p))
			depMods = append(depMods, depAbs)
		}
		depGraph[p] = depMods
	}

	var b strings.Builder
	b.WriteString("// GoScript bundle\n\n")

	// Generate module functions
	for _, p := range order {
		src, _ := os.ReadFile(p)
		name := modNames[p]

		// Build require() replacement for dependencies
		rewritten := string(src)
		for _, depPath := range depGraph[p] {
			depName := modNames[depPath]
			rel, _ := filepath.Rel(filepath.Dir(p), depPath)
			rel = filepath.ToSlash(rel)
			if !strings.HasPrefix(rel, ".") {
				rel = "./" + rel
			}
			old := fmt.Sprintf("require(%q)", rel)
			rewritten = strings.ReplaceAll(rewritten, old, depName+"_exports")
		}

		fmt.Fprintf(&b, "// %s\n", filepath.Base(p))
		fmt.Fprintf(&b, "var %s_exports = {};\n", name)

		// Call dependency modules
		for _, depPath := range depGraph[p] {
			depName := modNames[depPath]
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
	entryName := modNames[resolvePath(entry, baseDir)]
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

func resolvePath(path, baseDir string) string {
	if filepath.IsAbs(path) {
		return path
	}
	resolved := filepath.Join(baseDir, path)
	if filepath.Ext(resolved) == "" {
		resolved += ".gs"
	}
	return resolved
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
