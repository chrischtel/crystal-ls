package lsp

import (
	"fmt"
	"regexp"
	"strings"
)

func (a *CrystalAnalyzer) getDiagnostics(doc *TextDocumentItem) []Diagnostic {
	var diagnostics []Diagnostic

	lines := strings.Split(doc.Text, "\n")

	// Check for structure balance issues across the entire document
	structureErrors := a.checkStructureBalance(lines)
	diagnostics = append(diagnostics, structureErrors...)

	for i, line := range lines {
		pos := Position{Line: i, Character: 0}

		syntaxErrors := a.checkSyntaxError(line, pos)
		diagnostics = append(diagnostics, syntaxErrors...)

		undefinedVarErrors := a.checkUndefinedVariable(line, pos, doc)
		diagnostics = append(diagnostics, undefinedVarErrors...)
	}

	return diagnostics
}

func (a *CrystalAnalyzer) checkStructureBalance(lines []string) []Diagnostic {
	var diagnostics []Diagnostic
	var stack []struct {
		keyword string
		line    int
	}

	openingKeywords := []string{"class", "def", "if", "while", "case", "begin", "module", "unless", "for"}

	endRegexp := regexp.MustCompile(`\bend\b`)

	for lineNum, line := range lines {
		// Find opening keywords
		for _, keyword := range openingKeywords {
			pattern := fmt.Sprintf(`\b%s\b`, keyword)
			keywordRegexp := regexp.MustCompile(pattern)
			if keywordRegexp.MatchString(line) {
				stack = append(stack, struct {
					keyword string
					line    int
				}{keyword, lineNum})
			}
		}

		// Find end keywords
		if endRegexp.MatchString(line) {
			if len(stack) == 0 {
				// Unexpected end
				diagnostic := Diagnostic{
					Range: Range{
						Start: Position{Line: lineNum, Character: 0},
						End:   Position{Line: lineNum, Character: len(line)},
					},
					Severity: DiagnosticSeverityError,
					Message:  "Unexpected 'end' keyword - no matching opening statement",
				}
				diagnostics = append(diagnostics, diagnostic)
			} else {
				// Pop from stack
				stack = stack[:len(stack)-1]
			}
		}
	}

	// Check for unclosed structures
	for _, item := range stack {
		diagnostic := Diagnostic{
			Range: Range{
				Start: Position{Line: item.line, Character: 0},
				End:   Position{Line: item.line, Character: len(lines[item.line])},
			},
			Severity: DiagnosticSeverityError,
			Message:  fmt.Sprintf("Unclosed '%s' statement - missing 'end'", item.keyword),
		}
		diagnostics = append(diagnostics, diagnostic)
	}

	return diagnostics
}

func (a *CrystalAnalyzer) checkSyntaxError(line string, pos Position) []Diagnostic {
	var diagnostics []Diagnostic

	quoteCount := strings.Count(line, "\"") - strings.Count(line, "\\\"")
	if quoteCount%2 != 0 {
		diagnostic := Diagnostic{
			Range: Range{
				Start: pos,
				End:   Position{Line: pos.Line, Character: len(line)},
			},
			Severity: DiagnosticSeverityError,
			Message:  "Unclosed string literal",
		}
		diagnostics = append(diagnostics, diagnostic)
	}

	if match := regexp.MustCompile(`(\w+)\s*\(\s*([^)]*)\s*\)`).FindStringSubmatch(line); match != nil {
		params := strings.Split(match[2], ",")
		for i, param := range params {
			param = strings.TrimSpace(param)
			if param != "" && !regexp.MustCompile(`^\w+(\s*:\s*\w+)?(\s*=\s*.+)?$`).MatchString(param) {
				paramStart := strings.Index(line, match[0])
				if paramStart != -1 {
					diagnostic := Diagnostic{
						Range: Range{
							Start: Position{Line: pos.Line, Character: paramStart},
							End:   Position{Line: pos.Line, Character: paramStart + len(match[0])},
						},
						Severity: DiagnosticSeverityWarning,
						Message:  fmt.Sprintf("Invalid parameter syntax: %s (parameter %d)", param, i+1),
					}
					diagnostics = append(diagnostics, diagnostic)
				}
			}
		}
	}

	return diagnostics
}

func (a *CrystalAnalyzer) checkUndefinedVariable(line string, pos Position, doc *TextDocumentItem) []Diagnostic {
	var diagnostics []Diagnostic

	varPattern := regexp.MustCompile(`\b([a-zA-Z_]\w*)\b`)
	matches := varPattern.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		varName := match[1]

		if a.isKeyword(varName) || a.isBuiltinType(varName) {
			continue
		}

		if a.isMethodCall(line, varName) {
			continue
		}

		if !a.isVariableDefined(varName, doc, pos) && !a.isClassDefined(varName) {
			varStart := strings.Index(line, varName)
			if varStart != -1 {
				diagnostic := Diagnostic{
					Range: Range{
						Start: Position{Line: pos.Line, Character: varStart},
						End:   Position{Line: pos.Line, Character: varStart + len(varName)},
					},
					Severity: DiagnosticSeverityWarning,
					Message:  fmt.Sprintf("Undefined variable or method: %s", varName),
				}
				diagnostics = append(diagnostics, diagnostic)
			}
		}
	}

	return diagnostics
}

func (a *CrystalAnalyzer) isKeyword(word string) bool {
	for _, keyword := range a.keywords {
		if keyword == word {
			return true
		}
	}
	return false
}

func (a *CrystalAnalyzer) isBuiltinType(word string) bool {
	for _, typ := range a.builtinTypes {
		if typ == word {
			return true
		}
	}
	return false
}

func (a *CrystalAnalyzer) isMethodCall(line, varName string) bool {
	pattern := fmt.Sprintf(`%s\s*\(`, regexp.QuoteMeta(varName))
	matched, _ := regexp.MatchString(pattern, line)
	if matched {
		return true
	}

	dotPattern := fmt.Sprintf(`\w+\.%s`, regexp.QuoteMeta(varName))
	matched, _ = regexp.MatchString(dotPattern, line)
	return matched
}

func (a *CrystalAnalyzer) isVariableDefined(varName string, doc *TextDocumentItem, pos Position) bool {
	if _, exists := a.context.Variables[varName]; exists {
		return true
	}

	lines := strings.Split(doc.Text, "\n")
	for i := 0; i <= pos.Line; i++ {
		line := lines[i]

		assignmentPattern := fmt.Sprintf(`\b%s\s*=`, regexp.QuoteMeta(varName))
		if matched, _ := regexp.MatchString(assignmentPattern, line); matched {
			return true
		}

		paramPattern := fmt.Sprintf(`def\s+\w+\([^)]*\b%s\b`, regexp.QuoteMeta(varName))
		if matched, _ := regexp.MatchString(paramPattern, line); matched {
			return true
		}

		blockPattern := fmt.Sprintf(`\|\s*[^|]*\b%s\b`, regexp.QuoteMeta(varName))
		if matched, _ := regexp.MatchString(blockPattern, line); matched {
			return true
		}
	}

	return false
}

func (a *CrystalAnalyzer) isClassDefined(className string) bool {
	_, exists := a.context.Classes[className]
	return exists
}
