package lsp

import (
	"strings"
)

// TokenType represents different types of Crystal tokens
type TokenType int

const (
	TokenKeyword TokenType = iota
	TokenIdentifier
	TokenString
	TokenNumber
	TokenComment
	TokenOperator
	TokenSymbol
	TokenConstant
)

// Token represents a Crystal language token
type Token struct {
	Type     TokenType
	Value    string
	Position Position
	Length   int
}

// CrystalLexer provides basic lexical analysis for Crystal code
type CrystalLexer struct {
	text     string
	position int
	line     int
	column   int
	tokens   []Token
}

// NewCrystalLexer creates a new Crystal lexer
func NewCrystalLexer(text string) *CrystalLexer {
	return &CrystalLexer{
		text:   text,
		line:   0,
		column: 0,
	}
}

// Tokenize analyzes the text and returns a list of tokens
func (l *CrystalLexer) Tokenize() []Token {
	l.tokens = []Token{}
	l.position = 0
	l.line = 0
	l.column = 0

	for l.position < len(l.text) {
		l.skipWhitespace()

		if l.position >= len(l.text) {
			break
		}

		ch := l.text[l.position]

		switch {
		case ch == '#':
			l.readComment()
		case ch == '"' || ch == '\'':
			l.readString()
		case isDigit(ch):
			l.readNumber()
		case isLetter(ch) || ch == '_':
			l.readIdentifierOrKeyword()
		case isOperator(ch):
			l.readOperator()
		case ch == ':':
			l.readSymbol()
		default:
			l.advance()
		}
	}

	return l.tokens
}

// GetTokenAtPosition returns the token at the given position
func (l *CrystalLexer) GetTokenAtPosition(pos Position) *Token {
	for _, token := range l.tokens {
		if token.Position.Line == pos.Line &&
			pos.Character >= token.Position.Character &&
			pos.Character < token.Position.Character+token.Length {
			return &token
		}
	}
	return nil
}

func (l *CrystalLexer) skipWhitespace() {
	for l.position < len(l.text) {
		ch := l.text[l.position]
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
		} else if ch == '\n' {
			l.line++
			l.column = 0
			l.position++
		} else {
			break
		}
	}
}

func (l *CrystalLexer) readComment() {
	start := l.position
	startCol := l.column

	for l.position < len(l.text) && l.text[l.position] != '\n' {
		l.advance()
	}

	value := l.text[start:l.position]
	l.addToken(TokenComment, value, startCol, len(value))
}

func (l *CrystalLexer) readString() {
	start := l.position
	startCol := l.column
	quote := l.text[l.position]
	l.advance()

	for l.position < len(l.text) {
		ch := l.text[l.position]
		if ch == quote {
			l.advance()
			break
		}
		if ch == '\\' && l.position+1 < len(l.text) {
			l.advance() // Skip escape character
		}
		l.advance()
	}

	value := l.text[start:l.position]
	l.addToken(TokenString, value, startCol, len(value))
}

func (l *CrystalLexer) readNumber() {
	start := l.position
	startCol := l.column

	for l.position < len(l.text) && (isDigit(l.text[l.position]) || l.text[l.position] == '.') {
		l.advance()
	}

	value := l.text[start:l.position]
	l.addToken(TokenNumber, value, startCol, len(value))
}

func (l *CrystalLexer) readIdentifierOrKeyword() {
	start := l.position
	startCol := l.column

	for l.position < len(l.text) && (isAlphaNumeric(l.text[l.position]) || l.text[l.position] == '_' || l.text[l.position] == '?' || l.text[l.position] == '!') {
		l.advance()
	}

	value := l.text[start:l.position]
	tokenType := TokenIdentifier

	// Check if it's a keyword
	keywords := []string{
		"abstract", "alias", "and", "as", "begin", "break", "case", "class",
		"def", "do", "else", "elsif", "end", "ensure", "enum", "extend",
		"false", "for", "fun", "if", "in", "include", "instance_sizeof",
		"is_a?", "lib", "macro", "module", "next", "nil", "not", "of",
		"or", "out", "pointerof", "private", "protected", "rescue", "return",
		"require", "select", "self", "sizeof", "struct", "super", "then",
		"true", "type", "typeof", "union", "unless", "until", "when",
		"while", "with", "yield", "puts", "print", "p", "pp", "gets",
	}

	for _, keyword := range keywords {
		if value == keyword {
			tokenType = TokenKeyword
			break
		}
	}

	// Check if it's a constant (starts with uppercase)
	if len(value) > 0 && isUppercase(value[0]) {
		tokenType = TokenConstant
	}

	l.addToken(tokenType, value, startCol, len(value))
}

func (l *CrystalLexer) readOperator() {
	start := l.position
	startCol := l.column
	l.advance()

	value := l.text[start:l.position]
	l.addToken(TokenOperator, value, startCol, len(value))
}

func (l *CrystalLexer) readSymbol() {
	start := l.position
	startCol := l.column
	l.advance()

	// Read the symbol name
	for l.position < len(l.text) && (isAlphaNumeric(l.text[l.position]) || l.text[l.position] == '_') {
		l.advance()
	}

	value := l.text[start:l.position]
	l.addToken(TokenSymbol, value, startCol, len(value))
}

func (l *CrystalLexer) advance() {
	if l.position < len(l.text) {
		l.position++
		l.column++
	}
}

func (l *CrystalLexer) addToken(tokenType TokenType, value string, startCol, length int) {
	token := Token{
		Type:  tokenType,
		Value: value,
		Position: Position{
			Line:      l.line,
			Character: startCol,
		},
		Length: length,
	}
	l.tokens = append(l.tokens, token)
}

// Helper functions
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isAlphaNumeric(ch byte) bool {
	return isLetter(ch) || isDigit(ch)
}

func isUppercase(ch byte) bool {
	return ch >= 'A' && ch <= 'Z'
}

func isOperator(ch byte) bool {
	operators := "+-*/%=<>!&|^~.,:;()[]{}@"
	return strings.ContainsRune(operators, rune(ch))
}
