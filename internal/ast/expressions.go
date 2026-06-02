package ast

// ——— Literals ———

type Ident struct {
	Pos_     Position
	TokenLit string
}

func (i *Ident) expressionNode()      {}
func (i *Ident) patternNode()         {}
func (i *Ident) TokenLiteral() string { return i.TokenLit }
func (i *Ident) Pos() Position        { return i.Pos_ }

type NumberLit struct {
	Pos_     Position
	TokenLit string
	Value    float64
	IsInt    bool
}

func (n *NumberLit) expressionNode()      {}
func (n *NumberLit) TokenLiteral() string  { return n.TokenLit }
func (n *NumberLit) Pos() Position         { return n.Pos_ }

type StringLit struct {
	Pos_     Position
	TokenLit string
}

func (s *StringLit) expressionNode()      {}
func (s *StringLit) TokenLiteral() string { return s.TokenLit }
func (s *StringLit) Pos() Position        { return s.Pos_ }

type TemplateLit struct {
	Pos_     Position
	TokenLit string
}

func (t *TemplateLit) expressionNode()      {}
func (t *TemplateLit) TokenLiteral() string { return t.TokenLit }
func (t *TemplateLit) Pos() Position        { return t.Pos_ }

type BoolLit struct {
	Pos_     Position
	TokenLit string
	Value    bool
}

func (b *BoolLit) expressionNode()      {}
func (b *BoolLit) TokenLiteral() string { return b.TokenLit }
func (b *BoolLit) Pos() Position        { return b.Pos_ }

type NullLit struct {
	Pos_     Position
	TokenLit string
}

func (n *NullLit) expressionNode()      {}
func (n *NullLit) TokenLiteral() string { return n.TokenLit }
func (n *NullLit) Pos() Position        { return n.Pos_ }

type UndefinedLit struct {
	Pos_     Position
	TokenLit string
}

func (u *UndefinedLit) expressionNode()      {}
func (u *UndefinedLit) TokenLiteral() string { return u.TokenLit }
func (u *UndefinedLit) Pos() Position        { return u.Pos_ }

type ThisExpr struct {
	Pos_     Position
	TokenLit string
}

func (t *ThisExpr) expressionNode()      {}
func (t *ThisExpr) TokenLiteral() string { return t.TokenLit }
func (t *ThisExpr) Pos() Position        { return t.Pos_ }

type SuperExpr struct {
	Pos_     Position
	TokenLit string
	Method   string
}

func (s *SuperExpr) expressionNode()      {}
func (s *SuperExpr) TokenLiteral() string { return s.TokenLit }
func (s *SuperExpr) Pos() Position        { return s.Pos_ }

// ——— Composite ———

type ArrayLit struct {
	Pos_     Position
	TokenLit string
	Elements []Expression
}

func (a *ArrayLit) expressionNode()      {}
func (a *ArrayLit) TokenLiteral() string { return a.TokenLit }
func (a *ArrayLit) Pos() Position        { return a.Pos_ }

type ObjectLit struct {
	Pos_       Position
	TokenLit   string
	Properties []*Property
}

func (o *ObjectLit) expressionNode()      {}
func (o *ObjectLit) TokenLiteral() string { return o.TokenLit }
func (o *ObjectLit) Pos() Position        { return o.Pos_ }

type Property struct {
	Pos_       Position
	Key        Expression
	Value      Expression
	Computed   bool
	Shorthand  bool
	Spread     bool
	IsAccessor bool
}

// ——— Operators ———

type PrefixExpr struct {
	Pos_     Position
	TokenLit string // operator literal
	Op       string
	Right    Expression
}

func (p *PrefixExpr) expressionNode()      {}
func (p *PrefixExpr) TokenLiteral() string { return p.TokenLit }
func (p *PrefixExpr) Pos() Position        { return p.Pos_ }

type InfixExpr struct {
	Pos_     Position
	TokenLit string
	Op       string
	Left     Expression
	Right    Expression
}

func (ie *InfixExpr) expressionNode()      {}
func (ie *InfixExpr) TokenLiteral() string { return ie.TokenLit }
func (ie *InfixExpr) Pos() Position        { return ie.Pos_ }

type TernaryExpr struct {
	Pos_       Position
	TokenLit   string
	Cond       Expression
	Consequent Expression
	Alternate  Expression
}

func (te *TernaryExpr) expressionNode()      {}
func (te *TernaryExpr) TokenLiteral() string { return te.TokenLit }
func (te *TernaryExpr) Pos() Position        { return te.Pos_ }

type AssignExpr struct {
	Pos_     Position
	TokenLit string
	Op       string
	Left     Expression
	Right    Expression
}

func (ae *AssignExpr) expressionNode()      {}
func (ae *AssignExpr) TokenLiteral() string { return ae.TokenLit }
func (ae *AssignExpr) Pos() Position        { return ae.Pos_ }

// ——— Call / Member / Index ———

type CallExpr struct {
	Pos_     Position
	TokenLit string
	Callee   Expression
	Args     []Expression
}

func (ce *CallExpr) expressionNode()      {}
func (ce *CallExpr) TokenLiteral() string { return ce.TokenLit }
func (ce *CallExpr) Pos() Position        { return ce.Pos_ }

type MemberExpr struct {
	Pos_     Position
	TokenLit string
	Object   Expression
	Property Expression
	Computed bool
}

func (me *MemberExpr) expressionNode()      {}
func (me *MemberExpr) TokenLiteral() string { return me.TokenLit }
func (me *MemberExpr) Pos() Position        { return me.Pos_ }

type IndexExpr struct {
	Pos_     Position
	TokenLit string
	Left     Expression
	Index    Expression
}

func (ie *IndexExpr) expressionNode()      {}
func (ie *IndexExpr) TokenLiteral() string { return ie.TokenLit }
func (ie *IndexExpr) Pos() Position        { return ie.Pos_ }

type OptionalExpr struct {
	Pos_     Position
	TokenLit string
	Object   Expression
	Property Expression
	Computed bool
	IsCall   bool
	Args     []Expression
}

func (oe *OptionalExpr) expressionNode()      {}
func (oe *OptionalExpr) TokenLiteral() string { return oe.TokenLit }
func (oe *OptionalExpr) Pos() Position        { return oe.Pos_ }

// ——— Function / Class expressions ———

type FuncExpr struct {
	Pos_     Position
	TokenLit string
	Name     string
	Params   []*Param
	ReturnT  *TypeAnnotation
	Body     *BlockStmt
	IsAsync  bool
}

func (fe *FuncExpr) expressionNode()      {}
func (fe *FuncExpr) TokenLiteral() string { return fe.TokenLit }
func (fe *FuncExpr) Pos() Position        { return fe.Pos_ }

type ArrowFuncExpr struct {
	Pos_     Position
	TokenLit string
	Params   []*Param
	ReturnT  *TypeAnnotation
	Body     Node // Expression or *BlockStmt
	IsAsync  bool
}

func (af *ArrowFuncExpr) expressionNode()      {}
func (af *ArrowFuncExpr) TokenLiteral() string { return af.TokenLit }
func (af *ArrowFuncExpr) Pos() Position        { return af.Pos_ }

type NewExpr struct {
	Pos_     Position
	TokenLit string
	Callee   Expression
	Args     []Expression
}

func (ne *NewExpr) expressionNode()      {}
func (ne *NewExpr) TokenLiteral() string { return ne.TokenLit }
func (ne *NewExpr) Pos() Position        { return ne.Pos_ }

// ——— Misc ———

type AwaitExpr struct {
	Pos_     Position
	TokenLit string
	Value    Expression
}

func (ae *AwaitExpr) expressionNode()      {}
func (ae *AwaitExpr) TokenLiteral() string { return ae.TokenLit }
func (ae *AwaitExpr) Pos() Position        { return ae.Pos_ }

type SpreadExpr struct {
	Pos_     Position
	TokenLit string
	Value    Expression
}

func (se *SpreadExpr) expressionNode()      {}
func (se *SpreadExpr) TokenLiteral() string { return se.TokenLit }
func (se *SpreadExpr) Pos() Position        { return se.Pos_ }
