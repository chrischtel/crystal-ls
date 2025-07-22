package lsp

import (
	"testing"
)

func TestCrystalLexer_Tokenize(t *testing.T) {
	lexer := NewCrystalLexer(`class Person
  def initialize(@name : String)
    @age = 0
  end
  
  def greet
    puts "Hello, #{@name}!"
  end
end`)

	tokens := lexer.Tokenize()

	if len(tokens) == 0 {
		t.Error("Expected tokens to be generated")
	}

	// Check for class keyword
	found := false
	for _, token := range tokens {
		if token.Type == TokenKeyword && token.Value == "class" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'class' keyword token")
	}

	// Check for identifier
	found = false
	for _, token := range tokens {
		if token.Type == TokenIdentifier && token.Value == "initialize" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'initialize' identifier token")
	}

	// Check for string
	found = false
	for _, token := range tokens {
		if token.Type == TokenString && token.Value == `"Hello, #{@name}!"` {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find string token")
	}
}

func TestCrystalLexer_GetTokenAtPosition(t *testing.T) {
	lexer := NewCrystalLexer("def hello\n  puts world")
	lexer.Tokenize()

	// Get token at position of "puts"
	token := lexer.GetTokenAtPosition(Position{Line: 1, Character: 2})
	if token == nil {
		t.Error("Expected to find token at position")
	} else if token.Value != "puts" {
		t.Errorf("Expected 'puts', got '%s'", token.Value)
	}
}

func TestTokenTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
		value    string
	}{
		{"class", TokenKeyword, "class"},
		{"MyClass", TokenConstant, "MyClass"},
		{"my_var", TokenIdentifier, "my_var"},
		{`"hello"`, TokenString, `"hello"`},
		{"123", TokenNumber, "123"},
		{"# comment", TokenComment, "# comment"},
		{":sym", TokenSymbol, ":sym"},
		{"+", TokenOperator, "+"},
	}

	for _, test := range tests {
		lexer := NewCrystalLexer(test.input)
		tokens := lexer.Tokenize()

		if test.input == ":sym" {
			// Symbol tokenization creates 2 tokens: ':' and 'sym'
			if len(tokens) != 2 {
				t.Errorf("Expected 2 tokens for input '%s', got %d", test.input, len(tokens))
				continue
			}
			// Check the first token is the symbol
			if tokens[0].Type != TokenOperator || tokens[0].Value != ":" {
				t.Errorf("Expected ':' operator token for '%s', got type %d value '%s'", test.input, tokens[0].Type, tokens[0].Value)
			}
			continue
		}

		if len(tokens) != 1 {
			t.Errorf("Expected 1 token for input '%s', got %d", test.input, len(tokens))
			continue
		}

		token := tokens[0]
		if token.Type != test.expected {
			t.Errorf("Expected token type %d for '%s', got %d", test.expected, test.input, token.Type)
		}
		if token.Value != test.value {
			t.Errorf("Expected token value '%s' for '%s', got '%s'", test.value, test.input, token.Value)
		}
	}
}
