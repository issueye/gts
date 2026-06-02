package lexer

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	input    string
	offset   int // current byte offset in input
	readOff  int // reading offset (after current char)
	ch       rune
	line     int
	col      int
	prevCol  int // column before current ch
	errors   []string
}

func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		col:    0,
		offset: 0,
	}
	l.readChar()
	return l
}

func (l *Lexer) Errors() []string { return l.errors }

func (l *Lexer) addError(msg string) {
	l.errors = append(l.errors, fmt.Sprintf("Lexer error at line %d col %d: %s", l.line, l.col, msg))
}

func (l *Lexer) readChar() {
	if l.readOff >= len(l.input) {
		l.offset = l.readOff
		l.ch = 0
	} else {
		r, size := utf8.DecodeRuneInString(l.input[l.readOff:])
		l.prevCol = l.col
		if l.ch == '\n' {
			l.line++
			l.col = 0
		}
		l.offset = l.readOff
		l.readOff += size
		l.ch = r
		l.col++
	}
}

func (l *Lexer) peekChar() rune {
	if l.readOff >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readOff:])
	return r
}

func (l *Lexer) peekChar2() rune {
	off := l.readOff
	if off >= len(l.input) {
		return 0
	}
	_, sz := utf8.DecodeRuneInString(l.input[off:])
	off += sz
	if off >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[off:])
	return r
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	var tok Token
	startOff := l.offset
	startLine := l.line
	startCol := l.col

	switch l.ch {
	case '(':
		tok = l.newToken(TOKEN_LPAREN, "(")
	case ')':
		tok = l.newToken(TOKEN_RPAREN, ")")
	case '{':
		tok = l.newToken(TOKEN_LBRACE, "{")
	case '}':
		tok = l.newToken(TOKEN_RBRACE, "}")
	case '[':
		tok = l.newToken(TOKEN_LBRACK, "[")
	case ']':
		tok = l.newToken(TOKEN_RBRACK, "]")
	case ',':
		tok = l.newToken(TOKEN_COMMA, ",")
	case ';':
		tok = l.newToken(TOKEN_SEMI, ";")
	case ':':
		tok = l.newToken(TOKEN_COLON, ":")
	case '~':
		tok = l.newToken(TOKEN_TILDE, "~")
	case '^':
		if l.peekChar() == '=' {
			l.readChar()
			tok = l.newToken(TOKEN_CARET_EQ, "^=")
		} else {
			tok = l.newToken(TOKEN_CARET, "^")
		}
	case '?':
		next := l.peekChar()
		switch next {
		case '?':
			l.readChar()
			tok = l.newToken(TOKEN_QM_QM, "??")
		case '.':
			l.readChar()
			if l.peekChar() == '.' {
				l.readChar()
				tok = l.newToken(TOKEN_ELLIPSIS, "?..")
			} else {
				tok = l.newToken(TOKEN_QM_DOT, "?.")
			}
		default:
			tok = l.newToken(TOKEN_QUESTION, "?")
		}
	case '+':
		next := l.peekChar()
		switch next {
		case '+':
			l.readChar()
			tok = l.newToken(TOKEN_PLUS_PLUS, "++")
		case '=':
			l.readChar()
			tok = l.newToken(TOKEN_PLUS_EQ, "+=")
		default:
			tok = l.newToken(TOKEN_PLUS, "+")
		}
	case '-':
		next := l.peekChar()
		switch next {
		case '-':
			l.readChar()
			tok = l.newToken(TOKEN_MINUS_MINUS, "--")
		case '=':
			l.readChar()
			tok = l.newToken(TOKEN_MINUS_EQ, "-=")
		default:
			tok = l.newToken(TOKEN_MINUS, "-")
		}
	case '*':
		next := l.peekChar()
		switch next {
		case '*':
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = l.newToken(TOKEN_POW_EQ, "**=")
			} else {
				tok = l.newToken(TOKEN_POW, "**")
			}
		case '=':
			l.readChar()
			tok = l.newToken(TOKEN_STAR_EQ, "*=")
		default:
			tok = l.newToken(TOKEN_STAR, "*")
		}
	case '/':
		if l.peekChar() == '/' {
			l.skipLineComment()
			return l.NextToken()
		} else if l.peekChar() == '*' {
			l.skipBlockComment()
			return l.NextToken()
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = l.newToken(TOKEN_SLASH_EQ, "/=")
		} else {
			tok = l.newToken(TOKEN_SLASH, "/")
		}
	case '%':
		if l.peekChar() == '=' {
			l.readChar()
			tok = l.newToken(TOKEN_PERCENT_EQ, "%=")
		} else {
			tok = l.newToken(TOKEN_PERCENT, "%")
		}
	case '=':
		next := l.peekChar()
		switch next {
		case '=':
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = l.newToken(TOKEN_EQ_EQ_EQ, "===")
			} else {
				l.addError("'==' is not allowed in GoScript; use '===' for strict equality")
				tok = l.newToken(TOKEN_ILLEGAL, "==")
			}
		case '>':
			l.readChar()
			tok = l.newToken(TOKEN_ARROW, "=>")
		default:
			tok = l.newToken(TOKEN_EQ, "=")
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = l.newToken(TOKEN_NEQ_EQ, "!==")
			} else {
				l.addError("'!=' is not allowed in GoScript; use '!==' for strict inequality")
				tok = l.newToken(TOKEN_ILLEGAL, "!=")
			}
		} else {
			tok = l.newToken(TOKEN_BANG, "!")
		}
	case '<':
		next := l.peekChar()
		switch next {
		case '<':
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = l.newToken(TOKEN_LSHIFT_EQ, "<<=")
			} else {
				tok = l.newToken(TOKEN_LSHIFT, "<<")
			}
		case '=':
			l.readChar()
			tok = l.newToken(TOKEN_LT_EQ, "<=")
		default:
			tok = l.newToken(TOKEN_LT, "<")
		}
	case '>':
		next := l.peekChar()
		switch next {
		case '>':
			l.readChar()
			next2 := l.peekChar()
			switch next2 {
			case '>':
				l.readChar()
				if l.peekChar() == '=' {
					l.readChar()
					tok = l.newToken(TOKEN_URSHIFT_EQ, ">>>=")
				} else {
					tok = l.newToken(TOKEN_URSHIFT, ">>>")
				}
			case '=':
				l.readChar()
				tok = l.newToken(TOKEN_RSHIFT_EQ, ">>=")
			default:
				tok = l.newToken(TOKEN_RSHIFT, ">>")
			}
		case '=':
			l.readChar()
			tok = l.newToken(TOKEN_GT_EQ, ">=")
		default:
			tok = l.newToken(TOKEN_GT, ">")
		}
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok = l.newToken(TOKEN_AND_AND, "&&")
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = l.newToken(TOKEN_AMP_EQ, "&=")
		} else {
			tok = l.newToken(TOKEN_AMP, "&")
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = l.newToken(TOKEN_OR_OR, "||")
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = l.newToken(TOKEN_PIPE_EQ, "|=")
		} else {
			tok = l.newToken(TOKEN_PIPE, "|")
		}
	case '.':
		next := l.peekChar()
		switch next {
		case '.':
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = l.newToken(TOKEN_DOT_DOT_EQ, "..=")
			} else if l.peekChar() == '.' {
				l.readChar()
				tok = l.newToken(TOKEN_ELLIPSIS, "...")
			} else {
				tok = l.newToken(TOKEN_DOT_DOT, "..")
			}
		default:
			tok = l.newToken(TOKEN_DOT, ".")
		}
	case '"', '\'':
		tok = l.readString(l.ch)
	case '`':
		tok = l.readTemplate()
	case 0:
		tok = l.newToken(TOKEN_EOF, "")
	default:
		if isLetter(l.ch) {
			ident := l.readIdentifier()
			tok = l.makeIdentToken(ident, startLine, startCol)
			return tok
		} else if isDigit(l.ch) {
			num := l.readNumber()
			tok = l.makeToken(TOKEN_NUMBER, num, startLine, startCol, startOff)
			return tok
		} else {
			l.addError(fmt.Sprintf("unexpected character: %q (U+%04X)", l.ch, l.ch))
			tok = l.newToken(TOKEN_ILLEGAL, string(l.ch))
		}
	}
	tok.Offset = startOff
	tok.Line = startLine
	tok.Column = startCol
	l.readChar()
	return tok
}

func (l *Lexer) newToken(tpe TokenType, lit string) Token {
	return Token{Type: tpe, Literal: lit, Line: l.line, Column: l.col, Offset: l.offset}
}

func (l *Lexer) makeToken(tpe TokenType, lit string, line, col, off int) Token {
	return Token{Type: tpe, Literal: lit, Line: line, Column: col, Offset: off}
}

func (l *Lexer) makeIdentToken(ident string, line, col int) Token {
	return Token{Type: LookupIdent(ident), Literal: ident, Line: line, Column: col, Offset: l.offset}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipLineComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) skipBlockComment() {
	l.readChar() // consume '*'
	depth := 1
	for depth > 0 && l.ch != 0 {
		if l.ch == '/' && l.peekChar() == '*' {
			l.readChar()
			depth++
		} else if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			depth--
		}
		l.readChar()
	}
	if depth > 0 {
		l.addError("unterminated block comment")
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.offset
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start:l.offset]
}

func (l *Lexer) readNumber() string {
	start := l.offset
	isFloat := false

	if l.ch == '0' {
		next := l.peekChar()
		switch next {
		case 'x', 'X':
			l.readChar()
			l.readChar()
			for isHexDigit(l.ch) {
				l.readChar()
			}
			return l.input[start:l.offset]
		case 'b', 'B':
			l.readChar()
			l.readChar()
			for l.ch == '0' || l.ch == '1' {
				l.readChar()
			}
			return l.input[start:l.offset]
		case 'o', 'O':
			l.readChar()
			l.readChar()
			for l.ch >= '0' && l.ch <= '7' {
				l.readChar()
			}
			return l.input[start:l.offset]
		case '.':
			l.readChar()
			isFloat = true
		case 'e', 'E':
			l.readChar()
			isFloat = true
		}
	}

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' {
		if isDigit(l.peekChar()) {
			l.readChar()
			isFloat = true
			for isDigit(l.ch) {
				l.readChar()
			}
		}
	}

	if l.ch == 'e' || l.ch == 'E' {
		l.readChar()
		isFloat = true
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	_ = isFloat
	return l.input[start:l.offset]
}

func (l *Lexer) readString(quote rune) Token {
	start := l.offset
	startLine := l.line
	startCol := l.col
	l.readChar() // consume opening quote
	for l.ch != quote && l.ch != 0 && l.ch != '\n' {
		if l.ch == '\\' {
			l.readChar() // skip escaped char
		}
		l.readChar()
	}
	if l.ch != quote {
		l.addError("unterminated string literal")
	}
	lit := l.input[start:l.readOff] // includes both quotes
	// do NOT readChar here — outer NextToken does it
	return Token{Type: TOKEN_STRING, Literal: lit, Line: startLine, Column: startCol, Offset: start}
}

func (l *Lexer) readTemplate() Token {
	start := l.offset
	startLine := l.line
	startCol := l.col
	l.readChar() // consume opening backtick
	depth := 1
	for depth > 0 && l.ch != 0 {
		if l.ch == '`' {
			depth--
			if depth == 0 {
				break
			}
		} else if l.ch == '$' && l.peekChar() == '{' {
			depth++
			l.readChar()
		}
		l.readChar()
	}
	if depth > 0 {
		l.addError("unterminated template literal")
	}
	lit := l.input[start:l.readOff] // includes both backticks
	// do NOT readChar here — outer NextToken does it
	return Token{Type: TOKEN_TEMPLATE, Literal: lit, Line: startLine, Column: startCol, Offset: start}
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_' || ch == '$'
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func isHexDigit(ch rune) bool {
	return isDigit(ch) || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}
