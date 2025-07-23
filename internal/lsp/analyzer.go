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

	// Ensure we always return a valid slice
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}

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

		// Find method definitions (including property methods)
		if match := regexp.MustCompile(`^\s*def\s+(\w+[\?!]?)`).FindStringSubmatch(line); match != nil {
			methodName := match[1]
			if currentClass != "" {
				// Add to current class
				if classInfo, exists := a.documentClasses[currentClass]; exists {
					// Check if method already exists to avoid duplicates
					exists := false
					for _, existing := range classInfo.Methods {
						if existing == methodName {
							exists = true
							break
						}
					}
					if !exists {
						classInfo.Methods = append(classInfo.Methods, methodName)
					}
				}
			}
			// Also track globally
			if methods, exists := a.documentMethods[currentClass]; exists {
				a.documentMethods[currentClass] = append(methods, methodName)
			} else {
				a.documentMethods[currentClass] = []string{methodName}
			}
		}

		// Find property definitions (property name : Type creates getter/setter)
		if match := regexp.MustCompile(`^\s*property\s+(\w+)`).FindStringSubmatch(line); match != nil {
			propertyName := match[1]
			if currentClass != "" {
				// Add to current class (getter method)
				if classInfo, exists := a.documentClasses[currentClass]; exists {
					// Check if property getter already exists
					getterExists := false
					setterExists := false
					for _, existing := range classInfo.Methods {
						if existing == propertyName {
							getterExists = true
						}
						if existing == propertyName+"=" {
							setterExists = true
						}
					}
					if !getterExists {
						classInfo.Methods = append(classInfo.Methods, propertyName)
					}
					if !setterExists {
						classInfo.Methods = append(classInfo.Methods, propertyName+"=") // setter
					}
				}
			}
		}

		// Reset current class on 'end' (but be careful about nested ends)
		if strings.TrimSpace(line) == "end" {
			if currentClass != "" {
				// Simple approach: any 'end' at top level resets class
				// In a real parser, we'd track nesting properly
				currentClass = ""
			}
		}
	}
}

// GetCompletions provides intelligent completion suggestions
func (a *CrystalAnalyzer) GetCompletions(doc *TextDocumentItem, pos Position) CompletionList {
	var items []CompletionItem

	// Parse document structure first for better context
	a.parseDocumentStructure(doc)

	// Get the current line and context
	lines := strings.Split(doc.Text, "\n")
	if pos.Line >= len(lines) {
		return CompletionList{Items: items}
	}

	currentLine := lines[pos.Line]
	if pos.Character > len(currentLine) {
		pos.Character = len(currentLine)
	}

	prefix := currentLine[:pos.Character]

	// Analyze completion context
	context := a.analyzeCompletionContext(doc, pos, prefix)

	switch context.Type {
	case CompletionContextMethod:
		items = append(items, a.getAdvancedMethodCompletions(context, doc)...)
	case CompletionContextKeyword:
		items = append(items, a.getKeywordCompletions(context)...)
	default:
		// General completions (keywords, types, local classes)
		items = append(items, a.getGeneralCompletions(context)...)
	}

	return CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

// Enhanced completion types and structures

type CompletionContext struct {
	Type       CompletionContextType
	Prefix     string
	ObjectType string
	ObjectName string
	LastWord   string
	InMethod   bool
	InClass    bool
	ScopeInfo  *ScopeInfo
}

type CompletionContextType int

const (
	CompletionContextGeneral CompletionContextType = iota
	CompletionContextMethod
	CompletionContextKeyword
)

type ScopeInfo struct {
	Variables map[string]string // variable name -> type
	Methods   []string
	ClassName string
}

type MethodInfo struct {
	Name          string
	Signature     string
	Documentation string
}

// analyzeCompletionContext determines what kind of completion we need
func (a *CrystalAnalyzer) analyzeCompletionContext(doc *TextDocumentItem, pos Position, prefix string) CompletionContext {
	context := CompletionContext{
		Type:      CompletionContextGeneral,
		Prefix:    prefix,
		LastWord:  getLastWord(prefix),
		ScopeInfo: a.analyzeScopeAt(doc, pos),
	}

	// Check if we're completing after a dot (method completion)
	if dotIndex := strings.LastIndex(prefix, "."); dotIndex != -1 {
		beforeDot := strings.TrimSpace(prefix[:dotIndex])
		afterDot := prefix[dotIndex+1:]

		// Determine the type of the object before the dot
		objectType := a.inferTypeOfExpression(beforeDot, doc, pos)

		context.Type = CompletionContextMethod
		context.ObjectType = objectType
		context.ObjectName = a.extractObjectName(beforeDot)
		context.LastWord = afterDot
	}

	return context
}

// getAdvancedMethodCompletions provides intelligent method completions
func (a *CrystalAnalyzer) getAdvancedMethodCompletions(context CompletionContext, doc *TextDocumentItem) []CompletionItem {
	var items []CompletionItem

	// Get methods for the inferred type (don't re-parse to avoid performance issues)
	methods := a.getMethodsForType(context.ObjectType)

	for _, method := range methods {
		if context.LastWord == "" || strings.HasPrefix(strings.ToLower(method.Name), strings.ToLower(context.LastWord)) {
			items = append(items, CompletionItem{
				Label:         method.Name,
				Kind:          CompletionItemKindMethod,
				Detail:        method.Signature,
				Documentation: method.Documentation,
			})
		}
	}

	return items
} // getKeywordCompletions provides keyword completions
func (a *CrystalAnalyzer) getKeywordCompletions(context CompletionContext) []CompletionItem {
	var items []CompletionItem

	for _, keyword := range a.keywords {
		if context.LastWord == "" || strings.HasPrefix(keyword, context.LastWord) {
			items = append(items, CompletionItem{
				Label: keyword,
				Kind:  CompletionItemKindKeyword,
			})
		}
	}

	return items
}

// getGeneralCompletions provides general completions (keywords, types, classes)
func (a *CrystalAnalyzer) getGeneralCompletions(context CompletionContext) []CompletionItem {
	var items []CompletionItem

	// Add keywords
	items = append(items, a.getKeywordCompletions(context)...)

	// Add built-in types
	for _, typ := range a.builtinTypes {
		if context.LastWord == "" || strings.HasPrefix(strings.ToLower(typ), strings.ToLower(context.LastWord)) {
			items = append(items, CompletionItem{
				Label: typ,
				Kind:  CompletionItemKindClass,
			})
		}
	}

	// Add local class names
	for className := range a.documentClasses {
		if context.LastWord == "" || strings.HasPrefix(strings.ToLower(className), strings.ToLower(context.LastWord)) {
			items = append(items, CompletionItem{
				Label:  className,
				Kind:   CompletionItemKindClass,
				Detail: "Local class",
			})
		}
	}

	return items
}

// inferTypeOfExpression tries to determine the type of an expression
func (a *CrystalAnalyzer) inferTypeOfExpression(expr string, doc *TextDocumentItem, pos Position) string {
	expr = strings.TrimSpace(expr)

	// Check for literal types first (simple patterns)
	if len(expr) >= 2 && ((strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) ||
		(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'"))) {
		return "String"
	}

	// Simple number check
	if len(expr) > 0 && expr[0] >= '0' && expr[0] <= '9' {
		if strings.Contains(expr, ".") {
			return "Float64"
		}
		return "Int32"
	}

	if expr == "true" || expr == "false" {
		return "Bool"
	}
	if strings.HasPrefix(expr, "[") && strings.HasSuffix(expr, "]") {
		return "Array"
	}
	if strings.HasPrefix(expr, "{") && strings.HasSuffix(expr, "}") {
		return "Hash"
	}

	// Check for variable assignments in scope (simplified)
	if varType, exists := a.findVariableType(expr, doc, pos); exists {
		return varType
	}

	return "String" // Default to String for safety
} // findVariableType looks for variable type definitions
func (a *CrystalAnalyzer) findVariableType(varName string, doc *TextDocumentItem, pos Position) (string, bool) {
	lines := strings.Split(doc.Text, "\n")

	// Limit search to avoid infinite loops - only look back 50 lines maximum
	startLine := pos.Line
	if startLine > 50 {
		startLine = pos.Line - 50
	} else {
		startLine = 0
	}

	// Look backwards from current position for variable assignments
	for i := pos.Line; i >= startLine; i-- {
		line := lines[i]

		// Simple string matching instead of complex regex to avoid performance issues
		// Pattern: varName = Type.new (most common in Crystal)
		if strings.Contains(line, varName+" = ") && strings.Contains(line, ".new") {
			// Extract type name between = and .new
			parts := strings.Split(line, varName+" = ")
			if len(parts) > 1 {
				afterEquals := strings.TrimSpace(parts[1])
				if strings.Contains(afterEquals, ".new") {
					typeParts := strings.Split(afterEquals, ".new")
					if len(typeParts) > 0 {
						typeName := strings.TrimSpace(typeParts[0])
						if typeName != "" && isValidIdentifier(typeName) {
							return typeName, true
						}
					}
				}
			}
		}

		// Pattern: varName = "string"
		if strings.Contains(line, varName+" = \"") {
			return "String", true
		}

		// Pattern: varName = [...]
		if strings.Contains(line, varName+" = [") {
			return "Array", true
		}

		// Pattern: varName = {...}
		if strings.Contains(line, varName+" = {") {
			return "Hash", true
		}
	}

	return "", false
}

// isValidIdentifier checks if a string is a valid Crystal identifier
func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Must start with letter or underscore
	if !(s[0] >= 'A' && s[0] <= 'Z') && !(s[0] >= 'a' && s[0] <= 'z') && s[0] != '_' {
		return false
	}
	// Rest can be letters, digits, or underscores
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !(c >= 'A' && c <= 'Z') && !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '_' {
			return false
		}
	}
	return true
}

// getMethodsForType returns available methods for a given type
func (a *CrystalAnalyzer) getMethodsForType(typeName string) []MethodInfo {
	var methods []MethodInfo

	// Local class methods FIRST (so they appear at the top)
	if classInfo, exists := a.documentClasses[typeName]; exists {
		for _, methodName := range classInfo.Methods {
			detail := fmt.Sprintf("%s() : %s", methodName, typeName)
			documentation := fmt.Sprintf("Method of %s class", typeName)

			// Better labeling for properties
			if strings.HasSuffix(methodName, "=") {
				detail = fmt.Sprintf("%s : %s", methodName, "setter")
				documentation = fmt.Sprintf("Property setter of %s class", typeName)
			} else if !strings.Contains(methodName, "(") && !strings.Contains(methodName, "?") && !strings.Contains(methodName, "!") {
				// Likely a property getter (no special characters, no parentheses)
				detail = fmt.Sprintf("%s : property", methodName)
				documentation = fmt.Sprintf("Property getter of %s class", typeName)
			}

			methods = append(methods, MethodInfo{
				Name:          methodName,
				Signature:     detail,
				Documentation: documentation,
			})
		}
	}

	// Built-in type methods with enhanced information
	switch typeName {
	case "String":
		methods = append(methods, []MethodInfo{
			{Name: "size", Signature: "size : Int32", Documentation: "Returns the size of the string"},
			{Name: "length", Signature: "length : Int32", Documentation: "Returns the length of the string"},
			{Name: "empty?", Signature: "empty? : Bool", Documentation: "Returns true if the string is empty"},
			{Name: "upcase", Signature: "upcase : String", Documentation: "Returns a new string with all characters uppercase"},
			{Name: "downcase", Signature: "downcase : String", Documentation: "Returns a new string with all characters lowercase"},
			{Name: "strip", Signature: "strip : String", Documentation: "Returns a new string with leading and trailing whitespace removed"},
			{Name: "split", Signature: "split(delimiter : String) : Array(String)", Documentation: "Splits the string by delimiter"},
			{Name: "gsub", Signature: "gsub(pattern, replacement) : String", Documentation: "Replaces all occurrences of pattern with replacement"},
			{Name: "includes?", Signature: "includes?(substring : String) : Bool", Documentation: "Returns true if string contains substring"},
			{Name: "starts_with?", Signature: "starts_with?(prefix : String) : Bool", Documentation: "Returns true if string starts with prefix"},
			{Name: "ends_with?", Signature: "ends_with?(suffix : String) : Bool", Documentation: "Returns true if string ends with suffix"},
		}...)

	case "Array":
		methods = append(methods, []MethodInfo{
			{Name: "size", Signature: "size : Int32", Documentation: "Returns the size of the array"},
			{Name: "length", Signature: "length : Int32", Documentation: "Returns the length of the array"},
			{Name: "empty?", Signature: "empty? : Bool", Documentation: "Returns true if the array is empty"},
			{Name: "push", Signature: "push(element) : self", Documentation: "Adds element to the end of array"},
			{Name: "<<", Signature: "<<(element) : self", Documentation: "Adds element to the end of array"},
			{Name: "pop", Signature: "pop : T?", Documentation: "Removes and returns the last element"},
			{Name: "first", Signature: "first : T", Documentation: "Returns the first element"},
			{Name: "last", Signature: "last : T", Documentation: "Returns the last element"},
			{Name: "each", Signature: "each(&block) : Nil", Documentation: "Iterates over each element"},
			{Name: "map", Signature: "map(&block) : Array", Documentation: "Returns a new array with transformed elements"},
			{Name: "select", Signature: "select(&block) : Array", Documentation: "Returns a new array with elements that match the block"},
			{Name: "reject", Signature: "reject(&block) : Array", Documentation: "Returns a new array without elements that match the block"},
		}...)

	case "Hash":
		methods = append(methods, []MethodInfo{
			{Name: "size", Signature: "size : Int32", Documentation: "Returns the size of the hash"},
			{Name: "length", Signature: "length : Int32", Documentation: "Returns the length of the hash"},
			{Name: "empty?", Signature: "empty? : Bool", Documentation: "Returns true if the hash is empty"},
			{Name: "keys", Signature: "keys : Array", Documentation: "Returns an array of all keys"},
			{Name: "values", Signature: "values : Array", Documentation: "Returns an array of all values"},
			{Name: "has_key?", Signature: "has_key?(key) : Bool", Documentation: "Returns true if hash contains key"},
			{Name: "each", Signature: "each(&block) : Nil", Documentation: "Iterates over each key-value pair"},
		}...)

	case "Int32", "Int64", "Float32", "Float64":
		methods = append(methods, []MethodInfo{
			{Name: "abs", Signature: "abs : self", Documentation: "Returns the absolute value"},
			{Name: "round", Signature: "round : Int32", Documentation: "Returns the rounded value"},
			{Name: "ceil", Signature: "ceil : Int32", Documentation: "Returns the ceiling value"},
			{Name: "floor", Signature: "floor : Int32", Documentation: "Returns the floor value"},
			{Name: "to_s", Signature: "to_s : String", Documentation: "Converts to string"},
			{Name: "+", Signature: "+(other) : self", Documentation: "Addition"},
			{Name: "-", Signature: "-(other) : self", Documentation: "Subtraction"},
			{Name: "*", Signature: "*(other) : self", Documentation: "Multiplication"},
			{Name: "/", Signature: "/(other) : self", Documentation: "Division"},
		}...)
	}

	// Standard library methods that apply to all objects
	methods = append(methods, []MethodInfo{
		{Name: "to_s", Signature: "to_s : String", Documentation: "Returns string representation"},
		{Name: "inspect", Signature: "inspect : String", Documentation: "Returns detailed string representation"},
		{Name: "class", Signature: "class : Class", Documentation: "Returns the class of the object"},
		{Name: "nil?", Signature: "nil? : Bool", Documentation: "Returns true if object is nil"},
	}...)

	return methods
}

// analyzeScopeAt analyzes the scope at a given position
func (a *CrystalAnalyzer) analyzeScopeAt(doc *TextDocumentItem, pos Position) *ScopeInfo {
	scope := &ScopeInfo{
		Variables: make(map[string]string),
		Methods:   []string{},
	}

	lines := strings.Split(doc.Text, "\n")

	// Analyze from beginning to current position
	inClass := ""

	for i := 0; i <= pos.Line && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Track class scope
		if match := regexp.MustCompile(`^class\s+(\w+)`).FindStringSubmatch(line); match != nil {
			inClass = match[1]
			scope.ClassName = inClass
		}

		// Track variable assignments
		if match := regexp.MustCompile(`(\w+)\s*=\s*(.+)`).FindStringSubmatch(line); match != nil {
			varName := match[1]
			expr := match[2]
			varType := a.inferTypeFromExpression(expr)
			scope.Variables[varName] = varType
		}

		// Track instance variables
		if match := regexp.MustCompile(`(@\w+)\s*=\s*(.+)`).FindStringSubmatch(line); match != nil {
			varName := match[1]
			expr := match[2]
			varType := a.inferTypeFromExpression(expr)
			scope.Variables[varName] = varType
		}

		if line == "end" && inClass != "" {
			inClass = ""
		}
	}

	return scope
}

// inferTypeFromExpression infers type from an expression
func (a *CrystalAnalyzer) inferTypeFromExpression(expr string) string {
	expr = strings.TrimSpace(expr)

	// String literals
	if strings.HasPrefix(expr, "\"") || strings.HasPrefix(expr, "'") {
		return "String"
	}

	// Number literals
	if matched, _ := regexp.MatchString(`^\d+$`, expr); matched {
		return "Int32"
	}
	if matched, _ := regexp.MatchString(`^\d+\.\d+$`, expr); matched {
		return "Float64"
	}

	// Boolean literals
	if expr == "true" || expr == "false" {
		return "Bool"
	}

	// Array literals
	if strings.HasPrefix(expr, "[") {
		return "Array"
	}

	// Hash literals
	if strings.HasPrefix(expr, "{") {
		return "Hash"
	}

	// Constructor calls
	if match := regexp.MustCompile(`(\w+)\.new`).FindStringSubmatch(expr); match != nil {
		return match[1]
	}

	return "Object"
}

// getLastWord extracts the last word from a string
func getLastWord(text string) string {
	words := strings.Fields(text)
	if len(words) > 0 {
		return words[len(words)-1]
	}
	return text
}

// extractObjectName extracts the base object name from an expression
func (a *CrystalAnalyzer) extractObjectName(expr string) string {
	expr = strings.TrimSpace(expr)

	// Handle simple variable names
	if matched, _ := regexp.MatchString(`^\w+$`, expr); matched {
		return expr
	}

	// Handle method chains: obj.method1.method2 -> obj
	parts := strings.Split(expr, ".")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}

	// Extract the base object name from an expression
	words := strings.Fields(expr)
	if len(words) > 0 {
		return words[len(words)-1]
	}
	return expr
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

	// Check if it's a method definition
	for className, classInfo := range a.documentClasses {
		for _, methodName := range classInfo.Methods {
			if word == methodName {
				// Find the method definition in the document
				for lineNum, line := range lines {
					if match := regexp.MustCompile(`^\s*def\s+` + regexp.QuoteMeta(methodName) + `\b`).FindStringSubmatch(line); match != nil {
						return []Location{
							{
								URI: doc.URI,
								Range: Range{
									Start: Position{Line: lineNum, Character: strings.Index(line, methodName)},
									End:   Position{Line: lineNum, Character: strings.Index(line, methodName) + len(methodName)},
								},
							},
						}
					}
				}
			}
		}
		_ = className // Use the variable to avoid unused error
	}

	// Check if it's a variable assignment
	for i := pos.Line; i >= 0; i-- {
		line := lines[i]
		// Look for variable assignments like: word = something
		if match := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\s*=`).FindStringSubmatch(line); match != nil {
			return []Location{
				{
					URI: doc.URI,
					Range: Range{
						Start: Position{Line: i, Character: strings.Index(line, word)},
						End:   Position{Line: i, Character: strings.Index(line, word) + len(word)},
					},
				},
			}
		}
	}

	return []Location{}
}

// GetReferences finds all references to a symbol
func (a *CrystalAnalyzer) GetReferences(doc *TextDocumentItem, pos Position, includeDeclaration bool) []Location {
	lines := strings.Split(doc.Text, "\n")
	if pos.Line >= len(lines) {
		return []Location{}
	}

	currentLine := lines[pos.Line]
	word := getWordAtPosition(currentLine, pos.Character)
	var locations []Location

	// Search through all lines for references
	for lineNum, line := range lines {
		// Find all occurrences of the word
		index := 0
		for {
			foundIndex := strings.Index(line[index:], word)
			if foundIndex == -1 {
				break
			}

			actualIndex := index + foundIndex

			// Check if it's a whole word (not part of another word)
			isWholeWord := true
			if actualIndex > 0 {
				prev := rune(line[actualIndex-1])
				if isWordChar(prev) {
					isWholeWord = false
				}
			}
			if actualIndex+len(word) < len(line) {
				next := rune(line[actualIndex+len(word)])
				if isWordChar(next) {
					isWholeWord = false
				}
			}

			if isWholeWord {
				// Skip declaration if not requested
				isDeclation := false
				if regexp.MustCompile(`^\s*(class|def|module)\s+`+regexp.QuoteMeta(word)+`\b`).MatchString(line) ||
					regexp.MustCompile(`\b`+regexp.QuoteMeta(word)+`\s*=`).MatchString(line) {
					isDeclation = true
				}

				if includeDeclaration || !isDeclation {
					locations = append(locations, Location{
						URI: doc.URI,
						Range: Range{
							Start: Position{Line: lineNum, Character: actualIndex},
							End:   Position{Line: lineNum, Character: actualIndex + len(word)},
						},
					})
				}
			}

			index = actualIndex + len(word)
		}
	}

	return locations
}

// GetDocumentHighlights highlights all instances of a symbol in the document
func (a *CrystalAnalyzer) GetDocumentHighlights(doc *TextDocumentItem, pos Position) []DocumentHighlight {
	lines := strings.Split(doc.Text, "\n")
	if pos.Line >= len(lines) {
		return []DocumentHighlight{}
	}

	currentLine := lines[pos.Line]
	word := getWordAtPosition(currentLine, pos.Character)
	var highlights []DocumentHighlight

	// Search through all lines for the same word
	for lineNum, line := range lines {
		index := 0
		for {
			foundIndex := strings.Index(line[index:], word)
			if foundIndex == -1 {
				break
			}

			actualIndex := index + foundIndex

			// Check if it's a whole word
			isWholeWord := true
			if actualIndex > 0 && isWordChar(rune(line[actualIndex-1])) {
				isWholeWord = false
			}
			if actualIndex+len(word) < len(line) && isWordChar(rune(line[actualIndex+len(word)])) {
				isWholeWord = false
			}

			if isWholeWord {
				kind := DocumentHighlightKindText

				// Determine highlight kind
				if regexp.MustCompile(`^\s*(class|def|module)\s+` + regexp.QuoteMeta(word) + `\b`).MatchString(line) {
					kind = DocumentHighlightKindWrite // Declaration
				} else if regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\s*=`).MatchString(line) {
					kind = DocumentHighlightKindWrite // Assignment
				} else {
					kind = DocumentHighlightKindRead // Usage
				}

				highlights = append(highlights, DocumentHighlight{
					Range: Range{
						Start: Position{Line: lineNum, Character: actualIndex},
						End:   Position{Line: lineNum, Character: actualIndex + len(word)},
					},
					Kind: kind,
				})
			}

			index = actualIndex + len(word)
		}
	}

	return highlights
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

// GetDocumentFormat provides basic document formatting
func (a *CrystalAnalyzer) GetDocumentFormat(doc *TextDocumentItem) []TextEdit {
	var edits []TextEdit
	lines := strings.Split(doc.Text, "\n")

	for lineNum, line := range lines {
		// Basic formatting rules for Crystal
		formatted := line

		// Remove trailing whitespace
		trimmed := strings.TrimRightFunc(line, func(r rune) bool {
			return r == ' ' || r == '\t'
		})

		// Fix indentation (convert tabs to spaces, ensure consistent spacing)
		if strings.Contains(line, "\t") || strings.TrimSpace(line) != "" {
			content := strings.TrimSpace(line)
			if content != "" {
				// Apply consistent 2-space indentation for basic cases
				// This is a simplified version - a full formatter would need more context
				if strings.HasPrefix(content, "class ") || strings.HasPrefix(content, "module ") {
					formatted = content // Top level, no indent
				} else if strings.HasPrefix(content, "def ") {
					formatted = "  " + content // Method inside class
				} else if content == "end" {
					formatted = content // Same level as opening
				} else {
					// Keep existing indentation for now, just clean it up
					formatted = trimmed
				}
			} else {
				formatted = "" // Empty line
			}
		}

		if formatted != line {
			edits = append(edits, TextEdit{
				Range: Range{
					Start: Position{Line: lineNum, Character: 0},
					End:   Position{Line: lineNum, Character: len(line)},
				},
				NewText: formatted,
			})
		}
	}

	return edits
}

// GetFoldingRanges provides code folding ranges
func (a *CrystalAnalyzer) GetFoldingRanges(doc *TextDocumentItem) []FoldingRange {
	var ranges []FoldingRange
	lines := strings.Split(doc.Text, "\n")

	var stack []FoldingRangeStart

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of foldable regions
		if regexp.MustCompile(`^\s*(class|module|def|if|unless|case|while|until|begin)\s`).MatchString(line) {
			stack = append(stack, FoldingRangeStart{
				Line: lineNum,
				Kind: "region",
			})
		}

		// End of foldable regions
		if trimmed == "end" || strings.HasPrefix(trimmed, "end ") {
			if len(stack) > 0 {
				start := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				if lineNum > start.Line {
					ranges = append(ranges, FoldingRange{
						StartLine: start.Line,
						EndLine:   lineNum,
						Kind:      start.Kind,
					})
				}
			}
		}
	}

	return ranges
}

// Helper methods

func (a *CrystalAnalyzer) checkSyntaxError(line string, lineNum int) *Diagnostic {
	// Simple syntax checks
	trimmed := strings.TrimSpace(line)

	// Skip empty lines and comments
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}

	// Check for mismatched quotes (improved logic)
	inString := false
	inSingleQuote := false
	escaped := false

	for _, char := range trimmed {
		if escaped {
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' && !inSingleQuote {
			inString = !inString
		} else if char == '\'' && !inString {
			inSingleQuote = !inSingleQuote
		}
	}

	// Only report error if we have an unclosed quote on this specific line
	// and the line contains quote characters
	if (inString || inSingleQuote) && (strings.Contains(trimmed, `"`) || strings.Contains(trimmed, `'`)) {
		// Make sure this isn't a string interpolation or multi-line string
		if !strings.Contains(trimmed, "#{") && !strings.HasSuffix(trimmed, "\\") {
			return &Diagnostic{
				Range: Range{
					Start: Position{Line: lineNum, Character: 0},
					End:   Position{Line: lineNum, Character: len(line)},
				},
				Severity: DiagnosticSeverityError,
				Message:  "Unclosed quote on this line",
				Source:   "crystal-lsp",
			}
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

func (a *CrystalAnalyzer) findMethodCall(text string) string {
	// Look for method calls like "method_name("
	re := regexp.MustCompile(`(\w+)\s*\($`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
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
