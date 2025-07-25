package lsp

import (
	"fmt"
	"strings"
)

type CrystalAnalyzer struct {
	keywords      []string
	builtinTypes  []string
	stdlibMethods map[string][]string
	context       *DocumentContext
}

type DocumentContext struct {
	Classes   map[string]*ClassInfo
	Variables map[string]*VariableInfo
	Imports   []string
}

type ClassInfo struct {
	Name       string
	Methods    map[string]*MethodInfo
	Properties map[string]*PropertyInfo
	Location   Position
	SuperClass string
	Visibility string
}

type MethodInfo struct {
	Name          string
	Parameters    []ParameterInfo
	ReturnType    string
	Visibility    string
	Location      Position
	Documentation string
	IsProperty    bool
	IsInitializer bool
	Signature     string
}

type PropertyInfo struct {
	Name       string
	Type       string
	Visibility string
	Location   Position
	HasGetter  bool
	HasSetter  bool
	IsReadOnly bool
}

type ParameterInfo struct {
	Name         string
	Type         string
	DefaultValue string
	IsOptional   bool
}

type VariableInfo struct {
	Name     string
	Type     string
	Location Position
	Scope    string
}

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
		context: &DocumentContext{
			Classes:   make(map[string]*ClassInfo),
			Variables: make(map[string]*VariableInfo),
			Imports:   []string{},
		},
	}
}

func (a *CrystalAnalyzer) AnalyzeDocument(doc *TextDocumentItem) []Diagnostic {
	a.parseDocumentStructure(doc)
	return a.getDiagnostics(doc)
}

func (a *CrystalAnalyzer) GetHover(doc *TextDocumentItem, pos Position) *Hover {
	lines := strings.Split(doc.Text, "\n")
	if pos.Line >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	word := a.getWordAtPosition(line, pos.Character)
	if word == "" {
		return nil
	}

	return &Hover{
		Contents: []string{fmt.Sprintf("**%s**\n\nCrystal symbol", word)},
	}
}

func (a *CrystalAnalyzer) GetSignatureHelp(doc *TextDocumentItem, pos Position) *SignatureHelp {
	return &SignatureHelp{
		Signatures: []SignatureInformation{},
	}
}

func (a *CrystalAnalyzer) GetDefinition(doc *TextDocumentItem, pos Position) []Location {
	return []Location{}
}

func (a *CrystalAnalyzer) GetDocumentFormat(doc *TextDocumentItem) []TextEdit {
	return []TextEdit{}
}

func (a *CrystalAnalyzer) GetFoldingRanges(doc *TextDocumentItem) []FoldingRange {
	return []FoldingRange{}
}

func (a *CrystalAnalyzer) GetReferences(doc *TextDocumentItem, pos Position, includeDeclaration bool) []Location {
	return []Location{}
}

func (a *CrystalAnalyzer) GetDocumentHighlights(doc *TextDocumentItem, pos Position) []DocumentHighlight {
	return []DocumentHighlight{}
}

func (a *CrystalAnalyzer) GetDocumentSymbols(doc *TextDocumentItem) []SymbolInformation {
	a.parseDocumentStructure(doc)

	var symbols []SymbolInformation

	for _, class := range a.context.Classes {
		symbols = append(symbols, SymbolInformation{
			Name: class.Name,
			Kind: SymbolKindClass,
			Location: Location{
				URI: doc.URI,
				Range: Range{
					Start: class.Location,
					End:   Position{Line: class.Location.Line + 1, Character: 0},
				},
			},
		})

		for _, method := range class.Methods {
			if !method.IsProperty {
				symbols = append(symbols, SymbolInformation{
					Name: method.Name,
					Kind: SymbolKindMethod,
					Location: Location{
						URI: doc.URI,
						Range: Range{
							Start: method.Location,
							End:   Position{Line: method.Location.Line + 1, Character: 0},
						},
					},
				})
			}
		}
	}

	return symbols
}

func (a *CrystalAnalyzer) getWordAtPosition(line string, character int) string {
	if character >= len(line) {
		character = len(line) - 1
	}
	if character < 0 {
		return ""
	}

	start := character
	for start > 0 && isWordCharacter(line[start-1]) {
		start--
	}

	end := character
	for end < len(line) && isWordCharacter(line[end]) {
		end++
	}

	if start >= end {
		return ""
	}

	return line[start:end]
}

func isWordCharacter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '?' || c == '!'
}

func (a *CrystalAnalyzer) generateMethodSignature(name string, params []ParameterInfo, returnType string) string {
	if len(params) == 0 {
		return fmt.Sprintf("%s : %s", name, returnType)
	}

	var paramStrs []string
	for _, param := range params {
		paramStr := fmt.Sprintf("%s : %s", param.Name, param.Type)
		if param.DefaultValue != "" {
			paramStr += " = " + param.DefaultValue
		}
		paramStrs = append(paramStrs, paramStr)
	}

	return fmt.Sprintf("%s(%s) : %s", name, strings.Join(paramStrs, ", "), returnType)
}

func getLastWord(text string) string {
	words := strings.Fields(text)
	if len(words) > 0 {
		return words[len(words)-1]
	}
	return ""
}

func getWordAtPosition(line string, char int) string {
	if len(line) == 0 || char < 0 {
		return ""
	}

	if char >= len(line) {
		char = len(line) - 1
	}

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
