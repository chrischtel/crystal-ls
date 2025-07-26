package lsp

import (
	"regexp"
	"strings"
)

func (a *CrystalAnalyzer) parseDocumentStructure(doc *TextDocumentItem) {
	a.context = &DocumentContext{
		Classes:   make(map[string]*ClassInfo),
		Variables: make(map[string]*VariableInfo),
		Imports:   []string{},
	}

	lines := strings.Split(doc.Text, "\n")
	var currentClass *ClassInfo

	for lineNum, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if match := regexp.MustCompile(`^\s*class\s+(\w+)(?:\s*<\s*(\w+))?`).FindStringSubmatch(line); match != nil {
			className := match[1]
			superClass := ""
			if len(match) > 2 && match[2] != "" {
				superClass = match[2]
			}

			currentClass = &ClassInfo{
				Name:       className,
				Methods:    make(map[string]*MethodInfo),
				Properties: make(map[string]*PropertyInfo),
				Location:   Position{Line: lineNum, Character: 0},
				SuperClass: superClass,
				Visibility: "public",
			}
			a.context.Classes[className] = currentClass
		}

		a.parseMethodDefinition(line, lineNum, currentClass)
		a.parsePropertyDefinition(line, lineNum, currentClass)
		a.parseVariableAssignment(line, lineNum)

		if strings.Contains(trimmedLine, "end") && currentClass != nil {
			if a.isClassEnd(lines, lineNum, currentClass.Location.Line) {
				currentClass = nil
			}
		}
	}
}

func (a *CrystalAnalyzer) parseMethodDefinition(line string, lineNum int, currentClass *ClassInfo) {
	// Check for static methods first (def self.method_name)
	if match := regexp.MustCompile(`^\s*def\s+self\.(\w+[\?!]?)(?:\((.*?)\))?(?:\s*:\s*(\w+))?`).FindStringSubmatch(line); match != nil {
		methodName := match[1]
		paramsStr := ""
		if len(match) > 2 {
			paramsStr = match[2]
		}
		returnType := "Void"
		if len(match) > 3 && match[3] != "" {
			returnType = match[3]
		}

		parameters := a.parseParameters(paramsStr)

		methodInfo := &MethodInfo{
			Name:          methodName,
			Parameters:    parameters,
			ReturnType:    returnType,
			Visibility:    "public",
			Location:      Position{Line: lineNum, Character: 0},
			Documentation: "Static method " + methodName,
			IsProperty:    false,
			IsInitializer: false,
			IsStatic:      true,
			Signature:     a.generateMethodSignature(methodName, parameters, returnType),
		}

		if currentClass != nil {
			currentClass.Methods[methodName] = methodInfo
		}
		return
	}

	// Check for regular instance methods (def method_name)
	if match := regexp.MustCompile(`^\s*def\s+(\w+[\?!]?)(?:\((.*?)\))?(?:\s*:\s*(\w+))?`).FindStringSubmatch(line); match != nil {
		methodName := match[1]
		paramsStr := ""
		if len(match) > 2 {
			paramsStr = match[2]
		}
		returnType := "Void"
		if len(match) > 3 && match[3] != "" {
			returnType = match[3]
		}

		parameters := a.parseParameters(paramsStr)

		methodInfo := &MethodInfo{
			Name:          methodName,
			Parameters:    parameters,
			ReturnType:    returnType,
			Visibility:    "public",
			Location:      Position{Line: lineNum, Character: 0},
			Documentation: "Method " + methodName,
			IsProperty:    false,
			IsInitializer: methodName == "initialize",
			IsStatic:      false,
			Signature:     a.generateMethodSignature(methodName, parameters, returnType),
		}

		if currentClass != nil {
			currentClass.Methods[methodName] = methodInfo
		}
	}
}

func (a *CrystalAnalyzer) parsePropertyDefinition(line string, lineNum int, currentClass *ClassInfo) {
	if match := regexp.MustCompile(`^\s*property\s+(\w+)(?:\s*:\s*(\w+))?`).FindStringSubmatch(line); match != nil {
		propertyName := match[1]
		propertyType := "Object"
		if len(match) > 2 && match[2] != "" {
			propertyType = match[2]
		}

		if currentClass != nil {
			currentClass.Properties[propertyName] = &PropertyInfo{
				Name:       propertyName,
				Type:       propertyType,
				Visibility: "public",
				Location:   Position{Line: lineNum, Character: 0},
				HasGetter:  true,
				HasSetter:  true,
				IsReadOnly: false,
			}

			currentClass.Methods[propertyName] = &MethodInfo{
				Name:          propertyName,
				Parameters:    []ParameterInfo{},
				ReturnType:    propertyType,
				Visibility:    "public",
				Location:      Position{Line: lineNum, Character: 0},
				Documentation: "Property getter for " + propertyName,
				IsProperty:    true,
				IsInitializer: false,
				Signature:     propertyName + " : " + propertyType,
			}

			currentClass.Methods[propertyName+"="] = &MethodInfo{
				Name: propertyName + "=",
				Parameters: []ParameterInfo{{
					Name:       "value",
					Type:       propertyType,
					IsOptional: false,
				}},
				ReturnType:    propertyType,
				Visibility:    "public",
				Location:      Position{Line: lineNum, Character: 0},
				Documentation: "Property setter for " + propertyName,
				IsProperty:    true,
				IsInitializer: false,
				Signature:     propertyName + "=(" + propertyType + ") : " + propertyType,
			}
		}
	}
}

func (a *CrystalAnalyzer) parseVariableAssignment(line string, lineNum int) {
	if match := regexp.MustCompile(`^\s*(\w+)\s*=\s*(.+)`).FindStringSubmatch(line); match != nil {
		varName := match[1]
		assignment := strings.TrimSpace(match[2])

		varType := a.inferTypeFromAssignment(assignment)

		variable := &VariableInfo{
			Name:     varName,
			Type:     varType,
			Location: Position{Line: lineNum, Character: 0},
			Scope:    "global",
		}

		a.context.Variables[varName] = variable
	}
}

func (a *CrystalAnalyzer) parseParameters(paramsStr string) []ParameterInfo {
	var parameters []ParameterInfo
	if paramsStr == "" {
		return parameters
	}

	paramParts := strings.Split(paramsStr, ",")
	for _, paramPart := range paramParts {
		paramPart = strings.TrimSpace(paramPart)
		if paramPart == "" {
			continue
		}

		paramName := paramPart
		paramType := "Object"
		defaultValue := ""

		if strings.HasPrefix(paramPart, "@") {
			paramName = paramPart[1:]
		}

		if strings.Contains(paramPart, ":") {
			parts := strings.Split(paramPart, ":")
			if len(parts) >= 2 {
				paramName = strings.TrimSpace(parts[0])
				if strings.HasPrefix(paramName, "@") {
					paramName = paramName[1:]
				}
				typeAndDefault := strings.TrimSpace(parts[1])
				if strings.Contains(typeAndDefault, "=") {
					typeParts := strings.Split(typeAndDefault, "=")
					paramType = strings.TrimSpace(typeParts[0])
					defaultValue = strings.TrimSpace(typeParts[1])
				} else {
					paramType = typeAndDefault
				}
			}
		}

		parameters = append(parameters, ParameterInfo{
			Name:         paramName,
			Type:         paramType,
			DefaultValue: defaultValue,
			IsOptional:   defaultValue != "",
		})
	}

	return parameters
}

func (a *CrystalAnalyzer) inferTypeFromAssignment(assignment string) string {
	assignment = strings.TrimSpace(assignment)

	if match := regexp.MustCompile(`^(\w+)\.new\b`).FindStringSubmatch(assignment); match != nil {
		return match[1]
	}

	if strings.HasPrefix(assignment, `"`) && strings.HasSuffix(assignment, `"`) {
		return "String"
	}

	if strings.HasPrefix(assignment, "[") && strings.HasSuffix(assignment, "]") {
		return "Array"
	}

	if strings.HasPrefix(assignment, "{") && strings.HasSuffix(assignment, "}") {
		return "Hash"
	}

	if regexp.MustCompile(`^\d+\.\d+$`).MatchString(assignment) {
		return "Float64"
	}
	if regexp.MustCompile(`^\d+$`).MatchString(assignment) {
		return "Int32"
	}

	if assignment == "true" || assignment == "false" {
		return "Bool"
	}

	return "Object"
}

func (a *CrystalAnalyzer) isClassEnd(lines []string, endLine, classStartLine int) bool {
	openBlocks := 0
	for i := classStartLine + 1; i < endLine; i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "if ") ||
			strings.HasPrefix(line, "unless ") || strings.HasPrefix(line, "case ") ||
			strings.HasPrefix(line, "while ") || strings.HasPrefix(line, "for ") {
			openBlocks++
		} else if line == "end" {
			openBlocks--
		}
	}
	return openBlocks == 0
}
