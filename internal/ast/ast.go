package ast

import "fmt"

// Position marks a location in source code.
type Position struct {
	File   string // source file name
	Line   int    // 1-based line number
	Col    int    // 1-based column number
	Offset int    // byte offset from start of file
}

func (p Position) String() string {
	f := p.File
	if f == "" {
		f = "<source>"
	}
	return fmt.Sprintf("%s:%d:%d", f, p.Line, p.Col)
}

func (p Position) IsZero() bool { return p.Line == 0 }

// Node is the interface for all AST nodes.
type Node interface {
	TokenLiteral() string
	Pos() Position
}

// Statement is a node that represents a statement.
type Statement interface {
	Node
	statementNode()
}

// Expression is a node that represents an expression.
type Expression interface {
	Node
	expressionNode()
}

// Pattern is a node that represents a match arm pattern.
type Pattern interface {
	Node
	patternNode()
}
