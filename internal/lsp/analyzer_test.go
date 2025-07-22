package lsp

import (
	"testing"
)

func TestCrystalAnalyzer_GetCompletions(t *testing.T) {
	analyzer := NewCrystalAnalyzer()

	doc := &TextDocumentItem{
		URI:  "test.cr",
		Text: "def hello\n  puts \"H",
	}

	pos := Position{Line: 1, Character: 2} // Position just before "puts"
	completions := analyzer.GetCompletions(doc, pos)

	// Should have keyword completions including puts
	found := false
	for _, item := range completions.Items {
		if item.Label == "puts" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find 'puts' in completions. Got %d items:", len(completions.Items))
		for _, item := range completions.Items {
			t.Logf("  - %s (%d)", item.Label, item.Kind)
		}
	}

	// Test completion with partial word
	pos = Position{Line: 1, Character: 4} // Position after "pu"
	completions = analyzer.GetCompletions(doc, pos)

	found = false
	for _, item := range completions.Items {
		if item.Label == "puts" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find 'puts' in completions when typing 'pu'")
	}
}

func TestCrystalAnalyzer_GetDocumentSymbols(t *testing.T) {
	analyzer := NewCrystalAnalyzer()

	doc := &TextDocumentItem{
		URI: "test.cr",
		Text: `class MyClass
  def my_method
  end
end

module MyModule
  def module_method
  end
end`,
	}

	symbols := analyzer.GetDocumentSymbols(doc)

	if len(symbols) != 4 {
		t.Errorf("Expected 4 symbols, got %d", len(symbols))
	}

	// Check for class
	found := false
	for _, symbol := range symbols {
		if symbol.Name == "MyClass" && symbol.Kind == SymbolKindClass {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find MyClass symbol")
	}
}

func TestCrystalAnalyzer_AnalyzeDocument(t *testing.T) {
	analyzer := NewCrystalAnalyzer()

	doc := &TextDocumentItem{
		URI:  "test.cr",
		Text: `puts "hello world"`,
	}

	diagnostics := analyzer.AnalyzeDocument(doc)

	// Should have no diagnostics for valid code
	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}

	// Test with syntax error
	doc.Text = `puts "unclosed string`
	diagnostics = analyzer.AnalyzeDocument(doc)

	if len(diagnostics) == 0 {
		t.Error("Expected diagnostics for unclosed string")
	}
}

func TestGetWordAtPosition(t *testing.T) {
	tests := []struct {
		line     string
		pos      int
		expected string
	}{
		{"hello world", 3, "hello"},
		{"hello world", 7, "world"},
		{"foo.bar", 5, "bar"},
		{"", 0, ""},
	}

	for _, test := range tests {
		result := getWordAtPosition(test.line, test.pos)
		if result != test.expected {
			t.Errorf("getWordAtPosition(%q, %d) = %q, expected %q",
				test.line, test.pos, result, test.expected)
		}
	}
}

func TestSplitLines(t *testing.T) {
	text := "line1\nline2\nline3"
	lines := splitLines(text)

	expected := []string{"line1", "line2", "line3"}
	if len(lines) != len(expected) {
		t.Errorf("Expected %d lines, got %d", len(expected), len(lines))
	}

	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("Line %d: expected %q, got %q", i, expected[i], line)
		}
	}
}
