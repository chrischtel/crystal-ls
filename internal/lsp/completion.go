package lsp

import (
	"regexp"
	"strings"
)

func (a *CrystalAnalyzer) GetCompletions(doc *TextDocumentItem, pos Position) CompletionList {
	var items []CompletionItem

	a.parseDocumentStructure(doc)

	lines := strings.Split(doc.Text, "\n")
	if pos.Line >= len(lines) {
		return CompletionList{Items: items}
	}

	currentLine := lines[pos.Line]
	if pos.Character > len(currentLine) {
		pos.Character = len(currentLine)
	}

	prefix := currentLine[:pos.Character]

	context := a.analyzeCompletionContext(doc, pos, prefix)

	switch context.Type {
	case CompletionContextMethod:
		items = append(items, a.getMethodCompletions(context, doc)...)
	case CompletionContextKeyword:
		items = append(items, a.getKeywordCompletions(context)...)
	default:
		items = append(items, a.getGeneralCompletions(context)...)
	}

	return CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

type CompletionContext struct {
	Type       CompletionContextType
	Prefix     string
	ObjectType string
	ObjectName string
	LastWord   string
	InMethod   bool
	InClass    bool
	IsStatic   bool // true if we're looking for static methods on a class
}

type CompletionContextType int

const (
	CompletionContextGeneral CompletionContextType = iota
	CompletionContextMethod
	CompletionContextKeyword
)

func (a *CrystalAnalyzer) analyzeCompletionContext(doc *TextDocumentItem, pos Position, prefix string) CompletionContext {
	context := CompletionContext{
		Type:     CompletionContextGeneral,
		Prefix:   prefix,
		LastWord: getLastWord(prefix),
	}

	if dotIndex := strings.LastIndex(prefix, "."); dotIndex != -1 {
		beforeDot := strings.TrimSpace(prefix[:dotIndex])
		afterDot := prefix[dotIndex+1:]

		objectType := a.inferTypeOfExpression(beforeDot, doc, pos)

		// Check if we're dealing with a class name (static context)
		isStatic := a.isClassName(beforeDot)

		context.Type = CompletionContextMethod
		context.ObjectType = objectType
		context.ObjectName = a.extractObjectName(beforeDot)
		context.LastWord = afterDot
		context.IsStatic = isStatic
	}

	return context
}

func (a *CrystalAnalyzer) getMethodCompletions(context CompletionContext, doc *TextDocumentItem) []CompletionItem {
	var items []CompletionItem

	methods := a.getMethodsForType(context.ObjectType, context.IsStatic)

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
}

func (a *CrystalAnalyzer) getMethodsForType(typeName string, isStatic bool) []*MethodInfo {
	var methods []*MethodInfo

	if classInfo, exists := a.context.Classes[typeName]; exists {
		for _, method := range classInfo.Methods {
			// Filter methods based on whether we want static or instance methods
			if method.IsStatic == isStatic {
				methods = append(methods, method)
			}
		}
	}

	// Only add built-in methods for instance contexts (not static)
	if !isStatic {
		methods = append(methods, a.getBuiltInMethodsForType(typeName)...)
	}

	return methods
}

func (a *CrystalAnalyzer) isClassName(name string) bool {
	// Check if the name is a known class
	_, exists := a.context.Classes[name]
	return exists
}

func (a *CrystalAnalyzer) getBuiltInMethodsForType(typeName string) []*MethodInfo {
	var methods []*MethodInfo

	switch typeName {
	case "String":
		builtInMethods := []struct {
			name, signature, doc string
		}{
			{"size", "size : Int32", "Returns the size of the string"},
			{"length", "length : Int32", "Returns the length of the string"},
			{"empty?", "empty? : Bool", "Returns true if the string is empty"},
			{"upcase", "upcase : String", "Returns a new string with all characters uppercase"},
			{"downcase", "downcase : String", "Returns a new string with all characters lowercase"},
			{"strip", "strip : String", "Returns a new string with leading and trailing whitespace removed"},
			{"split", "split(delimiter : String) : Array(String)", "Splits the string by delimiter"},
			{"includes?", "includes?(substring : String) : Bool", "Returns true if string contains substring"},
			{"starts_with?", "starts_with?(prefix : String) : Bool", "Returns true if string starts with prefix"},
			{"ends_with?", "ends_with?(suffix : String) : Bool", "Returns true if string ends with suffix"},
		}

		for _, builtin := range builtInMethods {
			methods = append(methods, &MethodInfo{
				Name:          builtin.name,
				Signature:     builtin.signature,
				Documentation: builtin.doc,
			})
		}

	case "Array":
		builtInMethods := []struct {
			name, signature, doc string
		}{
			{"size", "size : Int32", "Returns the size of the array"},
			{"length", "length : Int32", "Returns the length of the array"},
			{"empty?", "empty? : Bool", "Returns true if the array is empty"},
			{"push", "push(element) : self", "Adds element to the end of array"},
			{"<<", "<<(element) : self", "Adds element to the end of array"},
			{"pop", "pop : T?", "Removes and returns the last element"},
			{"first", "first : T", "Returns the first element"},
			{"last", "last : T", "Returns the last element"},
			{"each", "each(&block) : Nil", "Iterates over each element"},
			{"map", "map(&block) : Array", "Returns a new array with transformed elements"},
			{"select", "select(&block) : Array", "Returns a new array with elements that match the block"},
		}

		for _, builtin := range builtInMethods {
			methods = append(methods, &MethodInfo{
				Name:          builtin.name,
				Signature:     builtin.signature,
				Documentation: builtin.doc,
			})
		}

	case "Hash":
		builtInMethods := []struct {
			name, signature, doc string
		}{
			{"size", "size : Int32", "Returns the size of the hash"},
			{"length", "length : Int32", "Returns the length of the hash"},
			{"empty?", "empty? : Bool", "Returns true if the hash is empty"},
			{"keys", "keys : Array", "Returns an array of all keys"},
			{"values", "values : Array", "Returns an array of all values"},
			{"has_key?", "has_key?(key) : Bool", "Returns true if hash contains key"},
			{"each", "each(&block) : Nil", "Iterates over each key-value pair"},
		}

		for _, builtin := range builtInMethods {
			methods = append(methods, &MethodInfo{
				Name:          builtin.name,
				Signature:     builtin.signature,
				Documentation: builtin.doc,
			})
		}
	}

	methods = append(methods, a.getBuiltInObjectMethods()...)

	return methods
}

func (a *CrystalAnalyzer) getBuiltInObjectMethods() []*MethodInfo {
	return []*MethodInfo{
		{Name: "class", Signature: "class : Class", Documentation: "Returns the class of the object"},
		{Name: "to_s", Signature: "to_s : String", Documentation: "Returns a string representation of the object"},
		{Name: "inspect", Signature: "inspect : String", Documentation: "Returns a detailed string representation of the object"},
		{Name: "nil?", Signature: "nil? : Bool", Documentation: "Returns true if the object is nil"},
		{Name: "responds_to?", Signature: "responds_to?(method : String) : Bool", Documentation: "Returns true if the object responds to the method"},
	}
}

func (a *CrystalAnalyzer) inferTypeOfExpression(expression string, doc *TextDocumentItem, pos Position) string {
	expression = strings.TrimSpace(expression)

	if match := regexp.MustCompile(`^(\w+)$`).FindStringSubmatch(expression); match != nil {
		varName := match[1]

		// Check if it's a class name first
		if a.isClassName(varName) {
			return varName
		}

		// Then check if it's a variable
		if varType, found := a.findVariableType(varName, doc, pos); found {
			return varType
		}
	}

	return "Object"
}

func (a *CrystalAnalyzer) findVariableType(varName string, doc *TextDocumentItem, pos Position) (string, bool) {
	if variable, exists := a.context.Variables[varName]; exists {
		return variable.Type, true
	}

	lines := strings.Split(doc.Text, "\n")
	startLine := pos.Line
	if startLine > 100 {
		startLine = pos.Line - 100
	} else {
		startLine = 0
	}

	for i := pos.Line; i >= startLine; i-- {
		line := lines[i]
		assignmentPattern := varName + " = "
		if strings.Contains(line, assignmentPattern) {
			parts := strings.Split(line, assignmentPattern)
			if len(parts) > 1 {
				afterEquals := strings.TrimSpace(parts[1])
				return a.inferTypeFromAssignment(afterEquals), true
			}
		}
	}

	return "", false
}

func (a *CrystalAnalyzer) extractObjectName(expression string) string {
	parts := strings.Split(expression, ".")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return expression
}

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

func (a *CrystalAnalyzer) getGeneralCompletions(context CompletionContext) []CompletionItem {
	var items []CompletionItem

	items = append(items, a.getKeywordCompletions(context)...)

	for _, typ := range a.builtinTypes {
		if context.LastWord == "" || strings.HasPrefix(strings.ToLower(typ), strings.ToLower(context.LastWord)) {
			items = append(items, CompletionItem{
				Label: typ,
				Kind:  CompletionItemKindClass,
			})
		}
	}

	for className := range a.context.Classes {
		if context.LastWord == "" || strings.HasPrefix(strings.ToLower(className), strings.ToLower(context.LastWord)) {
			items = append(items, CompletionItem{
				Label:  className,
				Kind:   CompletionItemKindClass,
				Detail: "Local class",
			})
		}
	}

	for varName := range a.context.Variables {
		if context.LastWord == "" || strings.HasPrefix(strings.ToLower(varName), strings.ToLower(context.LastWord)) {
			items = append(items, CompletionItem{
				Label:  varName,
				Kind:   CompletionItemKindVariable,
				Detail: a.context.Variables[varName].Type,
			})
		}
	}

	return items
}
