package ast

// LiteralPattern matches a literal value.
type LiteralPattern struct {
	Pos_     Position
	TokenLit string
	Value    Expression
}

func (lp *LiteralPattern) patternNode()       {}
func (lp *LiteralPattern) TokenLiteral() string { return lp.TokenLit }
func (lp *LiteralPattern) Pos() Position        { return lp.Pos_ }

// IdentPattern binds the matched value to a name.
type IdentPattern struct {
	Pos_     Position
	TokenLit string
	Name     string
}

func (ip *IdentPattern) patternNode()       {}
func (ip *IdentPattern) TokenLiteral() string { return ip.TokenLit }
func (ip *IdentPattern) Pos() Position        { return ip.Pos_ }

// WildcardPattern matches any value.
type WildcardPattern struct {
	Pos_     Position
	TokenLit string
}

func (wp *WildcardPattern) patternNode()       {}
func (wp *WildcardPattern) TokenLiteral() string { return wp.TokenLit }
func (wp *WildcardPattern) Pos() Position        { return wp.Pos_ }

// OrPattern matches any of its alternatives.
type OrPattern struct {
	Pos_         Position
	TokenLit     string
	Alternatives []Pattern
}

func (op *OrPattern) patternNode()       {}
func (op *OrPattern) TokenLiteral() string { return op.TokenLit }
func (op *OrPattern) Pos() Position        { return op.Pos_ }

// RangePattern matches a range of values.
type RangePattern struct {
	Pos_      Position
	TokenLit  string
	Start     Expression
	End       Expression
	Inclusive bool // true for ..= , false for ..
}

func (rp *RangePattern) patternNode()       {}
func (rp *RangePattern) TokenLiteral() string { return rp.TokenLit }
func (rp *RangePattern) Pos() Position        { return rp.Pos_ }
