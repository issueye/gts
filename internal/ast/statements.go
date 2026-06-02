package ast

// Program is the root node of every AST.
type Program struct {
	Pos_   Position
	Body   []Statement
	Errors []string
}

func (p *Program) TokenLiteral() string {
	if len(p.Body) > 0 {
		return p.Body[0].TokenLiteral()
	}
	return ""
}
func (p *Program) Pos() Position { return p.Pos_ }

// LetStmt represents `let x: T = expr;`.
type LetStmt struct {
	Pos_     Position
	TokenLit string
	Name     string
	TypeAnno *TypeAnnotation
	Value    Expression
}

func (ls *LetStmt) statementNode()       {}
func (ls *LetStmt) TokenLiteral() string { return ls.TokenLit }
func (ls *LetStmt) Pos() Position        { return ls.Pos_ }

// ConstStmt represents `const x: T = expr;`.
type ConstStmt struct {
	Pos_     Position
	TokenLit string
	Name     string
	TypeAnno *TypeAnnotation
	Value    Expression
}

func (cs *ConstStmt) statementNode()       {}
func (cs *ConstStmt) TokenLiteral() string { return cs.TokenLit }
func (cs *ConstStmt) Pos() Position        { return cs.Pos_ }

// VarStmt represents `var x: T = expr;`.
type VarStmt struct {
	Pos_     Position
	TokenLit string
	Name     string
	TypeAnno *TypeAnnotation
	Value    Expression
}

func (vs *VarStmt) statementNode()       {}
func (vs *VarStmt) TokenLiteral() string { return vs.TokenLit }
func (vs *VarStmt) Pos() Position        { return vs.Pos_ }

// FuncDecl represents a function declaration or expression.
type FuncDecl struct {
	Pos_     Position
	TokenLit string
	Name     string
	Params   []*Param
	ReturnT  *TypeAnnotation
	Body     *BlockStmt
	IsAsync  bool
}

func (fd *FuncDecl) statementNode()       {}
func (fd *FuncDecl) expressionNode()      {}
func (fd *FuncDecl) TokenLiteral() string { return fd.TokenLit }
func (fd *FuncDecl) Pos() Position        { return fd.Pos_ }

// Param is a function parameter.
type Param struct {
	Pos_     Position
	Name     string
	TypeAnno *TypeAnnotation
	Default  Expression
	Spread   bool
}

// BlockStmt is a `{ ... }` block.
type BlockStmt struct {
	Pos_       Position
	TokenLit   string
	Statements []Statement
}

func (bs *BlockStmt) statementNode()       {}
func (bs *BlockStmt) TokenLiteral() string { return bs.TokenLit }
func (bs *BlockStmt) Pos() Position        { return bs.Pos_ }

// IfStmt is `if (cond) consequence else alternative`.
type IfStmt struct {
	Pos_        Position
	TokenLit    string
	Cond        Expression
	Consequence *BlockStmt
	Alternative Statement
}

func (is *IfStmt) statementNode()       {}
func (is *IfStmt) TokenLiteral() string { return is.TokenLit }
func (is *IfStmt) Pos() Position        { return is.Pos_ }

// WhileStmt is `while (cond) body`.
type WhileStmt struct {
	Pos_     Position
	TokenLit string
	Cond     Expression
	Body     *BlockStmt
}

func (ws *WhileStmt) statementNode()       {}
func (ws *WhileStmt) TokenLiteral() string { return ws.TokenLit }
func (ws *WhileStmt) Pos() Position        { return ws.Pos_ }

// ForStmt is `for (init; cond; post) body`.
type ForStmt struct {
	Pos_     Position
	TokenLit string
	Init     Statement
	Cond     Expression
	Post     Expression
	Body     *BlockStmt
}

func (fs *ForStmt) statementNode()       {}
func (fs *ForStmt) TokenLiteral() string { return fs.TokenLit }
func (fs *ForStmt) Pos() Position        { return fs.Pos_ }

// ForInStmt is `for (name in iterable) body`.
type ForInStmt struct {
	Pos_     Position
	TokenLit string
	Name     string
	Iterable Expression
	Body     *BlockStmt
}

func (fs *ForInStmt) statementNode()       {}
func (fs *ForInStmt) TokenLiteral() string { return fs.TokenLit }
func (fs *ForInStmt) Pos() Position        { return fs.Pos_ }

// ForOfStmt is `for (name of iterable) body`.
type ForOfStmt struct {
	Pos_     Position
	TokenLit string
	Name     string
	Iterable Expression
	Body     *BlockStmt
}

func (fs *ForOfStmt) statementNode()       {}
func (fs *ForOfStmt) TokenLiteral() string { return fs.TokenLit }
func (fs *ForOfStmt) Pos() Position        { return fs.Pos_ }

// ReturnStmt is `return expr;`.
type ReturnStmt struct {
	Pos_     Position
	TokenLit string
	Value    Expression
}

func (rs *ReturnStmt) statementNode()       {}
func (rs *ReturnStmt) TokenLiteral() string { return rs.TokenLit }
func (rs *ReturnStmt) Pos() Position        { return rs.Pos_ }

// BreakStmt is `break;` or `break label;`.
type BreakStmt struct {
	Pos_     Position
	TokenLit string
	Label    string
}

func (bs *BreakStmt) statementNode()       {}
func (bs *BreakStmt) TokenLiteral() string { return bs.TokenLit }
func (bs *BreakStmt) Pos() Position        { return bs.Pos_ }

// ContinueStmt is `continue;` or `continue label;`.
type ContinueStmt struct {
	Pos_     Position
	TokenLit string
	Label    string
}

func (cs *ContinueStmt) statementNode()       {}
func (cs *ContinueStmt) TokenLiteral() string { return cs.TokenLit }
func (cs *ContinueStmt) Pos() Position        { return cs.Pos_ }

// TryStmt is `try { } catch (e) { } finally { }`.
type TryStmt struct {
	Pos_      Position
	TokenLit  string
	Block     *BlockStmt
	Catch     *CatchClause
	Finalizer *BlockStmt
}

func (ts *TryStmt) statementNode()       {}
func (ts *TryStmt) TokenLiteral() string { return ts.TokenLit }
func (ts *TryStmt) Pos() Position        { return ts.Pos_ }

// CatchClause is the catch part of a try.
type CatchClause struct {
	Pos_     Position
	Name     string
	TypeAnno *TypeAnnotation
	Body     *BlockStmt
}

// ThrowStmt is `throw expr;`.
type ThrowStmt struct {
	Pos_     Position
	TokenLit string
	Value    Expression
}

func (ts *ThrowStmt) statementNode()       {}
func (ts *ThrowStmt) TokenLiteral() string { return ts.TokenLit }
func (ts *ThrowStmt) Pos() Position        { return ts.Pos_ }

// MatchStmt is `match expr { arms }`.
type MatchExpr struct {
	Pos_     Position
	TokenLit string
	Expr     Expression
	Arms     []*MatchArm
}

func (me *MatchExpr) statementNode()       {}
func (me *MatchExpr) expressionNode()      {}
func (me *MatchExpr) TokenLiteral() string { return me.TokenLit }
func (me *MatchExpr) Pos() Position        { return me.Pos_ }

// MatchArm is one arm of a match expression.
type MatchArm struct {
	Pos_    Position
	Pattern Pattern
	Guard   Expression
	Body    Node // Expression or *BlockStmt
}

// ClassDecl is a class declaration.
type ClassDecl struct {
	Pos_     Position
	TokenLit string
	Name     string
	Super    Expression
	Body     *ClassBody
}

func (cd *ClassDecl) statementNode()       {}
func (cd *ClassDecl) expressionNode()      {}
func (cd *ClassDecl) TokenLiteral() string { return cd.TokenLit }
func (cd *ClassDecl) Pos() Position        { return cd.Pos_ }

// ClassBody holds class members.
type ClassBody struct {
	Pos_    Position
	Members []*ClassMember
}

// ClassMember is a member of a class.
type ClassMember struct {
	Pos_       Position
	IsStatic   bool
	IsAsync    bool
	Name       string
	Params     []*Param
	Body       *BlockStmt
	TypeAnno   *TypeAnnotation
	DefaultVal Expression
	Kind       string // "method" | "field" | "constructor"
}

// ExprStmt wraps an expression as a statement.
type ExprStmt struct {
	Pos_ Position
	Expr Expression
}

func (es *ExprStmt) statementNode()       {}
func (es *ExprStmt) TokenLiteral() string { return es.Expr.TokenLiteral() }
func (es *ExprStmt) Pos() Position        { return es.Pos_ }

// ImportDecl is `import x from "mod"`.
type ImportDecl struct {
	Pos_      Position
	TokenLit  string
	Default   string
	Namespace string
	Names     []string
	Aliases   map[string]string
	Source    string
}

func (id *ImportDecl) statementNode()       {}
func (id *ImportDecl) TokenLiteral() string { return id.TokenLit }
func (id *ImportDecl) Pos() Position        { return id.Pos_ }

// ExportDecl is `export x` or `export default expr`.
type ExportDecl struct {
	Pos_       Position
	TokenLit   string
	IsDefault  bool
	Decl       Statement
	Specifiers []ExportSpec
}

type ExportSpec struct {
	Name  string
	Alias string
}

func (ed *ExportDecl) statementNode()       {}
func (ed *ExportDecl) TokenLiteral() string { return ed.TokenLit }
func (ed *ExportDecl) Pos() Position        { return ed.Pos_ }

// LabeledStmt is `label: statement`.
type LabeledStmt struct {
	Pos_  Position
	Label string
	Stmt  Statement
}

func (ls *LabeledStmt) statementNode()       {}
func (ls *LabeledStmt) TokenLiteral() string { return ls.Stmt.TokenLiteral() }
func (ls *LabeledStmt) Pos() Position        { return ls.Pos_ }
