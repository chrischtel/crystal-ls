package lsp

import (
	"fmt"
	"regexp"
	"strings"
)

// CrystalAnalyzer provides Crystal language analysis capabilities
type CrystalAnalyzer struct {
	// Crystal keywords for syntax highlighting and completion
	keywords []string

	// Built-in types and classes
	builtinTypes []string

	// Standard library methods
	stdlibMethods map[string][]string

	// Document-specific class and method tracking
	documentClasses map[string]*ClassInfo
	documentMethods map[string][]string
}

// ClassInfo holds information about a class
type ClassInfo struct {
	Name     string
	Methods  []string
	Location Position
}

// NewCrystalAnalyzer creates a new Crystal language analyzer
func NewCrystalAnalyzer() *CrystalAnalyzer {
	return &CrystalAnalyzer{
		keywords: []string{
			"abstract", "alias", "and", "as", "begin", "break", "case", "class",
			"def", "do", "else", "elsif", "end", "ensure", "enum", "extend",
			"false", "for", "fun", "if", "in", "include", "instance_sizeof",
			"is_a?", "lib", "macro", "module", "next", "nil", "not", "of",
			"or", "out", "pointerof", "private", "protected", "rescue", "return",
			"require", "select", "self", "sizeof", "struct", "super", "then",
			"true", "type", "typeof", "union", "unless", "until", "when",
			"while", "with", "yield", "puts", "print", "p", "pp", "gets",
		},
		builtinTypes: []string{
			"Array", "Bool", "Char", "Class", "Enum", "Float32", "Float64",
			"Hash", "Int8", "Int16", "Int32", "Int64", "Int128", "Module",
			"Nil", "Number", "Object", "Proc", "Range", "Regex", "Set",
			"String", "Symbol", "Tuple", "UInt8", "UInt16", "UInt32",
			"UInt64", "UInt128", "Union", "Value", "Void",
		},
		stdlibMethods: map[string][]string{
			"String": {
				"size", "length", "empty?", "blank?", "downcase", "upcase",
				"capitalize", "strip", "lstrip", "rstrip", "split", "gsub",
				"sub", "match", "includes?", "starts_with?", "ends_with?",
				"to_i", "to_f", "to_s", "chars", "bytes", "lines",
			},
			"Array": {
				"size", "length", "empty?", "first", "last", "push", "pop",
				"shift", "unshift", "insert", "delete", "delete_at", "clear",
				"concat", "join", "map", "select", "reject", "find", "each",
				"sort", "reverse", "shuffle", "uniq", "flatten", "compact",
			},
			"Hash": {
				"size", "length", "empty?", "keys", "values", "has_key?",
				"has_value?", "fetch", "merge", "delete", "clear", "each",
				"each_key", "each_value", "select", "reject", "transform_keys",
				"transform_values", "invert", "to_a",
			},
			"Int32": {
				"abs", "ceil", "floor", "round", "to_i", "to_f", "to_s",
				"times", "upto", "downto", "step", "even?", "odd?", "+", "-",
				"*", "/", "%", "**", "==", "!=", "<", ">", "<=", ">=",
			},
		},
		documentClasses: make(map[string]*ClassInfo),
		documentMethods: make(map[string][]string),
	}
}

// AnalyzeDocument analyzes a Crystal document and returns diagnostics
func (a *CrystalAnalyzer) AnalyzeDocument(doc *TextDocumentItem) []Diagnostic {
	var diagnostics []Diagnostic

	// Parse classes and methods in the document first
	a.parseDocumentStructure(doc)

	// Create lexer for this document
	lexer := NewCrystalLexer(doc.Text)
	tokens := lexer.Tokenize()

	lines := strings.Split(doc.Text, "\n")

	for lineNum, line := range lines {
		// Check for syntax errors
		if diag := a.checkSyntaxError(line, lineNum); diag != nil {
			diagnostics = append(diagnostics, *diag)
		}

		// Check for undefined variables (simple heuristic)
		if diag := a.checkUndefinedVariable(line, lineNum); diag != nil {
			diagnostics = append(diagnostics, *diag)
		}
	}

	// Use tokens for additional analysis
	diagnostics = append(diagnostics, a.analyzeTokens(tokens, doc.URI)...)

	return diagnostics
}

// parseDocumentStructure parses classes and methods in the document
func (a *CrystalAnalyzer) parseDocumentStructure(doc *TextDocumentItem) {
	// Clear previous data
	a.documentClasses = make(map[string]*ClassInfo)
	a.documentMethods = make(map[string][]string)

	lines := strings.Split(doc.Text, "\n")
	var currentClass string

	for lineNum, line := range lines {
		// Find class definitions
		if match := regexp.MustCompile(`^\s*class\s+(\w+)`).FindStringSubmatch(line); match != nil {
			className := match[1]
			currentClass = className
			a.documentClasses[className] = &ClassInfo{
				Name:     className,
				Methods:  []string{},
				Location: Position{Line: lineNum, Character: 0},
			}
		}

		// Find method definitions
		if match := regexp.MustCompile(`^\s*def\s+(\w+[\?!]?)`).FindStringSubmatch(line); match != nil {
			methodName := match[1]
			if currentClass != "" {
				// Add to current class
				if classInfo, exists := a.documentClasses[currentClass]; exists {
					classInfo.Methods = append(classInfo.Methods, methodName)
				}
			}
			// Also track globally
			if methods, exists := a.documentMethods[currentClass]; exists {
				a.documentMethods[currentClass] = append(methods, methodName)
			} else {
				a.documentMethods[currentClass] = []string{methodName}
			}
		}

		// Reset current class on 'end'
		if strings.TrimSpace(line) == "end" && currentClass != "" {
			currentClass = ""
		}
	}
}

// GetCompletions provides completion suggestions
func (a *CrystalAnalyzer) GetCompletions(doc *TextDocumentItem, pos Position) CompletionList {
	var items []CompletionItem

	// Parse document structure first
	a.parseDocumentStructure(doc)

	// Get the current line
	lines := strings.Split(doc.Text, "\n")
	if pos.Line >= len(lines) {
		return CompletionList{Items: items}
	}

	currentLine := lines[pos.Line]
	if pos.Character > len(currentLine) {
		pos.Character = len(currentLine)
	}

	prefix := currentLine[:pos.Character]

	// Check if we're completing after a dot (method completion)
	if strings.Contains(prefix, ".") {
		items = append(items, a.getMethodCompletions(prefix, doc)...)
	} else {
		// Get the word being typed
		lastWord := getLastWord(prefix)

		// Add keywords
		for _, keyword := range a.keywords {
			if lastWord == "" || strings.HasPrefix(keyword, lastWord) {
				items = append(items, CompletionItem{
					Label: keyword,
					Kind:  CompletionItemKindKeyword,
				})
			}
		}

		// Add built-in types
		for _, typ := range a.builtinTypes {
			if lastWord == "" || strings.HasPrefix(strings.ToLower(typ), strings.ToLower(lastWord)) {
				items = append(items, CompletionItem{
					Label: typ,
					Kind:  CompletionItemKindClass,
				})
			}
		}

		// Add local class names
		for className := range a.documentClasses {
			if lastWord == "" || strings.HasPrefix(strings.ToLower(className), strings.ToLower(lastWord)) {
				items = append(items, CompletionItem{
					Label:  className,
					Kind:   CompletionItemKindClass,
					Detail: "Local class",
				})
			}
		}
	}

	return CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

// GetHover provides hover information
func (a *CrystalAnalyzer) GetHover(doc *TextDocumentItem, pos Position) *Hover {
	lines := strings.Split(doc.Text, "\n")
	if pos.Line >= len(lines) {
		return nil
	}

	currentLine := lines[pos.Line]
	word := getWordAtPosition(currentLine, pos.Character)

	if word == "" {
		return nil
	}

	// Parse document structure
	a.parseDocumentStructure(doc)

	// Check if it's a local class
	if classInfo, exists := a.documentClasses[word]; exists {
		methodList := strings.Join(classInfo.Methods, ", ")
		return &Hover{
			Contents: []string{fmt.Sprintf("**%s** - Local class\n\nMethods: %s", word, methodList)},
		}
	}

	// Check if it's a keyword
	for _, keyword := range a.keywords {
		if word == keyword {
			return &Hover{
				Contents: []string{fmt.Sprintf("**%s** - Crystal keyword", keyword)},
			}
		}
	}

	// Check if it's a built-in type
	for _, typ := range a.builtinTypes {
		if word == typ {
			return &Hover{
				Contents: []string{fmt.Sprintf("**%s** - Built-in Crystal type", typ)},
			}
		}
	}

	return nil
}

// GetSignatureHelp provides signature help
func (a *CrystalAnalyzer) GetSignatureHelp(doc *TextDocumentItem, pos Position) *SignatureHelp {
	lines := strings.Split(doc.Text, "\n")
	if pos.Line >= len(lines) {
		return nil
	}

	currentLine := lines[pos.Line]
	prefix := currentLine[:pos.Character]

	// Simple heuristic: look for method calls
	methodCall := a.findMethodCall(prefix)
	if methodCall != "" {
		return &SignatureHelp{
			Signatures: []SignatureInformation{
				{
					Label:         fmt.Sprintf("%s(args)", methodCall),
					Documentation: fmt.Sprintf("Method call: %s", methodCall),
				},
			},
			ActiveSignature: 0,
			ActiveParameter: 0,
		}
	}

	return nil
}

// GetDefinition provides go-to-definition
func (a *CrystalAnalyzer) GetDefinition(doc *TextDocumentItem, pos Position) []Location {
	lines := strings.Split(doc.Text, "\n")
	if pos.Line >= len(lines) {
		return []Location{}
	}

	currentLine := lines[pos.Line]
	word := getWordAtPosition(currentLine, pos.Character)

	// Parse document structure
	a.parseDocumentStructure(doc)

	// Check if it's a local class
	if classInfo, exists := a.documentClasses[word]; exists {
		return []Location{
			{
				URI: doc.URI,
				Range: Range{
					Start: classInfo.Location,
					End:   Position{Line: classInfo.Location.Line, Character: classInfo.Location.Character + len(word)},
				},
			},
		}
	}

	return []Location{}
}

// GetDocumentSymbols provides document symbols
func (a *CrystalAnalyzer) GetDocumentSymbols(doc *TextDocumentItem) []SymbolInformation {
	var symbols []SymbolInformation

	lines := strings.Split(doc.Text, "\n")

	for lineNum, line := range lines {
		// Find class definitions
		if match := regexp.MustCompile(`^\s*class\s+(\w+)`).FindStringSubmatch(line); match != nil {
			symbols = append(symbols, SymbolInformation{
				Name: match[1],
				Kind: SymbolKindClass,
				Location: Location{
					URI: doc.URI,
					Range: Range{
						Start: Position{Line: lineNum, Character: 0},
						End:   Position{Line: lineNum, Character: len(line)},
					},
				},
			})
		}

		// Find method definitions
		if match := regexp.MustCompile(`^\s*def\s+(\w+[\?!]?)`).FindStringSubmatch(line); match != nil {
			symbols = append(symbols, SymbolInformation{
				Name: match[1],
				Kind: SymbolKindMethod,
				Location: Location{
					URI: doc.URI,
					Range: Range{
						Start: Position{Line: lineNum, Character: 0},
						End:   Position{Line: lineNum, Character: len(line)},
					},
				},
			})
		}

		// Find module definitions
		if match := regexp.MustCompile(`^\s*module\s+(\w+)`).FindStringSubmatch(line); match != nil {
			symbols = append(symbols, SymbolInformation{
				Name: match[1],
				Kind: SymbolKindModule,
				Location: Location{
					URI: doc.URI,
					Range: Range{
						Start: Position{Line: lineNum, Character: 0},
						End:   Position{Line: lineNum, Character: len(line)},
					},
				},
			})
		}
	}

	return symbols
}

// Helper methods

func (a *CrystalAnalyzer) checkSyntaxError(line string, lineNum int) *Diagnostic {
	// Simple syntax checks
	trimmed := strings.TrimSpace(line)

	// Check for mismatched quotes
	if strings.Count(trimmed, `"`)%2 != 0 && !strings.Contains(trimmed, `\"`) {
		return &Diagnostic{
			Range: Range{
				Start: Position{Line: lineNum, Character: 0},
				End:   Position{Line: lineNum, Character: len(line)},
			},
			Severity: DiagnosticSeverityError,
			Message:  "Mismatched quotes",
			Source:   "crystal-lsp",
		}
	}

	return nil
}

func (a *CrystalAnalyzer) checkUndefinedVariable(line string, lineNum int) *Diagnostic {
	// This is a very basic check - in practice you'd need proper scope analysis
	return nil
}

// analyzeTokens performs token-based analysis
func (a *CrystalAnalyzer) analyzeTokens(tokens []Token, uri string) []Diagnostic {
	var diagnostics []Diagnostic

	// Example: Check for unused variables (very basic)
	// In a real implementation, you'd track variable declarations and usage

	return diagnostics
}

func (a *CrystalAnalyzer) getMethodCompletions(prefix string, doc *TextDocumentItem) []CompletionItem {
	var items []CompletionItem

	// Extract the variable name before the dot
	parts := strings.Split(prefix, ".")
	if len(parts) < 2 {
		return items
	}

	// Get the variable name (last part before the dot)
	varName := strings.TrimSpace(parts[len(parts)-2])

	// Try to determine the type by looking for variable assignments
	lines := strings.Split(doc.Text, "\n")
	for _, line := range lines {
		// Look for patterns like: varName = ClassName.new
		if match := regexp.MustCompile(varName + `\s*=\s*(\w+)\.new`).FindStringSubmatch(line); match != nil {
			className := match[1]

			// Check if we have this class in our document
			if classInfo, exists := a.documentClasses[className]; exists {
				for _, method := range classInfo.Methods {
					items = append(items, CompletionItem{
						Label:  method,
						Kind:   CompletionItemKindMethod,
						Detail: fmt.Sprintf("Method of %s", className),
					})
				}
				return items
			}
		}
	}

	// Fallback to standard library methods for common patterns
	if methods, exists := a.stdlibMethods["String"]; exists {
		for _, method := range methods {
			items = append(items, CompletionItem{
				Label: method,
				Kind:  CompletionItemKindMethod,
			})
		}
	}

	return items
}

func (a *CrystalAnalyzer) findMethodCall(text string) string {
	// Look for method calls like "method_name("
	re := regexp.MustCompile(`(\w+)\s*\($`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func getLastWord(text string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	return words[len(words)-1]
}

func getWordAtPosition(line string, char int) string {
	if len(line) == 0 || char < 0 {
		return ""
	}

	if char >= len(line) {
		char = len(line) - 1
	}

	// Find word boundaries
	start := char
	for start > 0 && isWordChar(rune(line[start-1])) {
		start--
	}

	end := char
	for end < len(line) && isWordChar(rune(line[end])) {
		end++
	}

	if start >= end {
		return ""
	}

	return line[start:end]
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '?' || r == '!'
}
