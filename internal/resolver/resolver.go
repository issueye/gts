package resolver

import (
	"fmt"

	"github.com/issueye/goscript/internal/ast"
)

type BindingKind string

const (
	BindingLet      BindingKind = "let"
	BindingConst    BindingKind = "const"
	BindingVar      BindingKind = "var"
	BindingFunction BindingKind = "function"
	BindingClass    BindingKind = "class"
	BindingParam    BindingKind = "param"
	BindingImport   BindingKind = "import"
	BindingPattern  BindingKind = "pattern"
	BindingCatch    BindingKind = "catch"
	BindingImplicit BindingKind = "implicit"
)

type Binding struct {
	Name  string
	Kind  BindingKind
	Scope *Scope
	Pos   ast.Position
}

type Reference struct {
	Name    string
	Scope   *Scope
	Binding *Binding
	Depth   int
	Pos     ast.Position
}

type Error struct {
	Pos     ast.Position
	Message string
}

func (e Error) Error() string {
	if e.Pos.IsZero() {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Pos, e.Message)
}

type ScopeKind string

const (
	ScopeGlobal   ScopeKind = "global"
	ScopeBlock    ScopeKind = "block"
	ScopeFunction ScopeKind = "function"
	ScopeClass    ScopeKind = "class"
	ScopeMatchArm ScopeKind = "match-arm"
)

type Scope struct {
	ID       int
	Kind     ScopeKind
	Parent   *Scope
	Bindings map[string]*Binding
}

type Options struct {
	Predeclared []string
}

type Result struct {
	Global     *Scope
	Scopes     []*Scope
	Bindings   []*Binding
	References []*Reference
	Errors     []Error
}

type Resolver struct {
	opts   Options
	result *Result
	scope  *Scope
}

func New(opts Options) *Resolver {
	return &Resolver{opts: opts}
}

func Resolve(program *ast.Program, opts Options) *Result {
	return New(opts).Resolve(program)
}

func (r *Resolver) Resolve(program *ast.Program) *Result {
	global := r.newScope(ScopeGlobal, nil)
	r.result = &Result{Global: global, Scopes: []*Scope{global}}
	r.scope = global
	for _, name := range r.opts.Predeclared {
		r.declare(name, BindingImplicit, ast.Position{})
	}
	if program != nil {
		r.stmts(program.Body)
	}
	return r.result
}

func (r *Resolver) newScope(kind ScopeKind, parent *Scope) *Scope {
	scope := &Scope{Kind: kind, Parent: parent, Bindings: make(map[string]*Binding)}
	if r.result != nil {
		scope.ID = len(r.result.Scopes)
		r.result.Scopes = append(r.result.Scopes, scope)
	}
	return scope
}

func (r *Resolver) push(kind ScopeKind) *Scope {
	scope := r.newScope(kind, r.scope)
	r.scope = scope
	return scope
}

func (r *Resolver) pop() {
	if r.scope != nil {
		r.scope = r.scope.Parent
	}
}

func (r *Resolver) declare(name string, kind BindingKind, pos ast.Position) *Binding {
	if name == "" || r.scope == nil {
		return nil
	}
	if existing := r.scope.Bindings[name]; existing != nil {
		r.addError(pos, "duplicate declaration of %q", name)
		return existing
	}
	binding := &Binding{Name: name, Kind: kind, Scope: r.scope, Pos: pos}
	r.scope.Bindings[name] = binding
	r.result.Bindings = append(r.result.Bindings, binding)
	return binding
}

func (r *Resolver) reference(name string, pos ast.Position) {
	if name == "" || r.scope == nil {
		return
	}
	binding, depth := r.lookup(name)
	ref := &Reference{Name: name, Scope: r.scope, Binding: binding, Depth: depth, Pos: pos}
	r.result.References = append(r.result.References, ref)
	if binding == nil {
		r.addError(pos, "undefined identifier %q", name)
	}
}

func (r *Resolver) lookup(name string) (*Binding, int) {
	depth := 0
	for scope := r.scope; scope != nil; scope = scope.Parent {
		if binding := scope.Bindings[name]; binding != nil {
			return binding, depth
		}
		depth++
	}
	return nil, -1
}

func (r *Resolver) addError(pos ast.Position, format string, args ...interface{}) {
	r.result.Errors = append(r.result.Errors, Error{Pos: pos, Message: fmt.Sprintf(format, args...)})
}

func (r *Resolver) stmts(stmts []ast.Statement) {
	for _, stmt := range stmts {
		r.stmt(stmt)
	}
}

func (r *Resolver) stmt(stmt ast.Statement) {
	switch n := stmt.(type) {
	case *ast.LetStmt:
		r.expr(n.Value)
		r.declare(n.Name, BindingLet, n.Pos())
	case *ast.ConstStmt:
		r.expr(n.Value)
		r.declare(n.Name, BindingConst, n.Pos())
	case *ast.VarStmt:
		r.expr(n.Value)
		r.declare(n.Name, BindingVar, n.Pos())
	case *ast.FuncDecl:
		r.declare(n.Name, BindingFunction, n.Pos())
		r.function(n.Params, n.Body)
	case *ast.ClassDecl:
		r.declare(n.Name, BindingClass, n.Pos())
		r.expr(n.Super)
		r.push(ScopeClass)
		for _, member := range n.Body.Members {
			r.expr(member.DefaultVal)
			if member.Body != nil {
				r.function(member.Params, member.Body)
			}
		}
		r.pop()
	case *ast.BlockStmt:
		r.push(ScopeBlock)
		r.stmts(n.Statements)
		r.pop()
	case *ast.IfStmt:
		r.expr(n.Cond)
		r.stmt(n.Consequence)
		r.stmt(n.Alternative)
	case *ast.WhileStmt:
		r.expr(n.Cond)
		r.stmt(n.Body)
	case *ast.ForStmt:
		r.push(ScopeBlock)
		r.stmt(n.Init)
		r.expr(n.Cond)
		r.expr(n.Post)
		r.stmt(n.Body)
		r.pop()
	case *ast.ForInStmt:
		r.expr(n.Iterable)
		r.push(ScopeBlock)
		r.declare(n.Name, BindingLet, n.Pos())
		r.stmt(n.Body)
		r.pop()
	case *ast.ForOfStmt:
		r.expr(n.Iterable)
		r.push(ScopeBlock)
		r.declare(n.Name, BindingLet, n.Pos())
		r.stmt(n.Body)
		r.pop()
	case *ast.ReturnStmt:
		r.expr(n.Value)
	case *ast.ThrowStmt:
		r.expr(n.Value)
	case *ast.TryStmt:
		r.stmt(n.Block)
		if n.Catch != nil {
			r.push(ScopeBlock)
			if n.Catch.Name != "" {
				r.declare(n.Catch.Name, BindingCatch, n.Catch.Pos_)
			}
			r.stmt(n.Catch.Body)
			r.pop()
		}
		r.stmt(n.Finalizer)
	case *ast.MatchExpr:
		r.matchExpr(n)
	case *ast.ExprStmt:
		r.expr(n.Expr)
	case *ast.ImportDecl:
		if n.Default != "" {
			r.declare(n.Default, BindingImport, n.Pos())
		}
		if n.Namespace != "" {
			r.declare(n.Namespace, BindingImport, n.Pos())
		}
		for _, name := range n.Names {
			r.declare(name, BindingImport, n.Pos())
		}
		for _, local := range n.Aliases {
			r.declare(local, BindingImport, n.Pos())
		}
	case *ast.ExportDecl:
		if len(n.Specifiers) > 0 {
			for _, spec := range n.Specifiers {
				r.reference(spec.Name, n.Pos())
			}
			return
		}
		r.stmt(n.Decl)
	case *ast.LabeledStmt:
		r.stmt(n.Stmt)
	}
}

func (r *Resolver) function(params []*ast.Param, body *ast.BlockStmt) {
	r.push(ScopeFunction)
	for _, param := range params {
		r.expr(param.Default)
		r.declare(param.Name, BindingParam, param.Pos_)
	}
	if body != nil {
		r.stmts(body.Statements)
	}
	r.pop()
}

func (r *Resolver) expr(expr ast.Expression) {
	switch n := expr.(type) {
	case nil:
	case *ast.Ident:
		r.reference(n.TokenLit, n.Pos())
	case *ast.NumberLit, *ast.StringLit, *ast.TemplateLit, *ast.BoolLit, *ast.NullLit, *ast.UndefinedLit, *ast.ThisExpr, *ast.SuperExpr:
	case *ast.ArrayLit:
		for _, elem := range n.Elements {
			r.expr(elem)
		}
	case *ast.ObjectLit:
		for _, prop := range n.Properties {
			if prop.Computed {
				r.expr(prop.Key)
			}
			r.expr(prop.Value)
		}
	case *ast.PrefixExpr:
		r.expr(n.Right)
	case *ast.InfixExpr:
		r.expr(n.Left)
		r.expr(n.Right)
	case *ast.TernaryExpr:
		r.expr(n.Cond)
		r.expr(n.Consequent)
		r.expr(n.Alternate)
	case *ast.AssignExpr:
		r.expr(n.Left)
		r.expr(n.Right)
	case *ast.CallExpr:
		r.expr(n.Callee)
		for _, arg := range n.Args {
			r.expr(arg)
		}
	case *ast.MemberExpr:
		r.expr(n.Object)
		if n.Computed {
			r.expr(n.Property)
		}
	case *ast.IndexExpr:
		r.expr(n.Left)
		r.expr(n.Index)
	case *ast.OptionalExpr:
		r.expr(n.Object)
		if n.Computed {
			r.expr(n.Property)
		}
		for _, arg := range n.Args {
			r.expr(arg)
		}
	case *ast.FuncExpr:
		r.function(n.Params, n.Body)
	case *ast.ArrowFuncExpr:
		r.push(ScopeFunction)
		for _, param := range n.Params {
			r.expr(param.Default)
			r.declare(param.Name, BindingParam, param.Pos_)
		}
		if body, ok := n.Body.(*ast.BlockStmt); ok {
			r.stmts(body.Statements)
		} else if body, ok := n.Body.(ast.Expression); ok {
			r.expr(body)
		}
		r.pop()
	case *ast.NewExpr:
		r.expr(n.Callee)
		for _, arg := range n.Args {
			r.expr(arg)
		}
	case *ast.AwaitExpr:
		r.expr(n.Value)
	case *ast.SpreadExpr:
		r.expr(n.Value)
	case *ast.MatchExpr:
		r.matchExpr(n)
	}
}

func (r *Resolver) matchExpr(n *ast.MatchExpr) {
	r.expr(n.Expr)
	for _, arm := range n.Arms {
		r.push(ScopeMatchArm)
		r.pattern(arm.Pattern)
		r.expr(arm.Guard)
		switch body := arm.Body.(type) {
		case ast.Expression:
			r.expr(body)
		case ast.Statement:
			r.stmt(body)
		}
		r.pop()
	}
}

func (r *Resolver) pattern(pattern ast.Pattern) {
	switch n := pattern.(type) {
	case nil:
	case *ast.LiteralPattern:
		r.expr(n.Value)
	case *ast.IdentPattern:
		r.declare(n.Name, BindingPattern, n.Pos())
	case *ast.WildcardPattern:
	case *ast.OrPattern:
		for _, alt := range n.Alternatives {
			r.pattern(alt)
		}
	case *ast.RangePattern:
		r.expr(n.Start)
		r.expr(n.End)
	}
}
