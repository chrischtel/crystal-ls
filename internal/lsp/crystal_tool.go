package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CrystalTool provides integration with Crystal compiler tools
type CrystalTool struct {
	crystalPath   string
	workspaceRoot string
}

// NewCrystalTool creates a new Crystal tool integration
func NewCrystalTool(workspaceRoot string) *CrystalTool {
	crystalPath := findCrystalExecutable()
	return &CrystalTool{
		crystalPath:   crystalPath,
		workspaceRoot: workspaceRoot,
	}
}

// ContextInfo represents context information from Crystal
type ContextInfo struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Location    string   `json:"location"`
	Methods     []string `json:"methods,omitempty"`
	Ancestors   []string `json:"ancestors,omitempty"`
	Description string   `json:"description,omitempty"`
}

// GetContext uses `crystal tool context` to get type information at a position
func (ct *CrystalTool) GetContext(filename string, line, column int) (*ContextInfo, error) {
	if ct.crystalPath == "" {
		return nil, fmt.Errorf("crystal executable not found")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	// Run crystal tool context
	cmd := exec.Command(ct.crystalPath, "tool", "context",
		fmt.Sprintf("--cursor=%d:%d", line+1, column+1), absPath)
	cmd.Dir = ct.workspaceRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("crystal tool context failed: %v", err)
	}

	// Parse the output (Crystal returns JSON-like format)
	var contextInfo ContextInfo
	if err := json.Unmarshal(output, &contextInfo); err != nil {
		// If JSON parsing fails, try to parse text output
		return ct.parseTextContext(string(output))
	}

	return &contextInfo, nil
}

// GetImplementations uses `crystal tool implementations` to find implementations
func (ct *CrystalTool) GetImplementations(filename string, line, column int) ([]Location, error) {
	if ct.crystalPath == "" {
		return nil, fmt.Errorf("crystal executable not found")
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(ct.crystalPath, "tool", "implementations",
		fmt.Sprintf("--cursor=%d:%d", line+1, column+1), absPath)
	cmd.Dir = ct.workspaceRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("crystal tool implementations failed: %v", err)
	}

	return ct.parseLocations(string(output))
}

// FormatCode uses `crystal tool format` to format Crystal code
func (ct *CrystalTool) FormatCode(filename string) (string, error) {
	if ct.crystalPath == "" {
		return "", fmt.Errorf("crystal executable not found")
	}

	cmd := exec.Command(ct.crystalPath, "tool", "format", filename)
	cmd.Dir = ct.workspaceRoot

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("crystal tool format failed: %v", err)
	}

	return string(output), nil
}

// GetTypeHierarchy uses `crystal tool hierarchy` to get type hierarchy
func (ct *CrystalTool) GetTypeHierarchy(filename string, line, column int) ([]string, error) {
	if ct.crystalPath == "" {
		return nil, fmt.Errorf("crystal executable not found")
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(ct.crystalPath, "tool", "hierarchy",
		fmt.Sprintf("--cursor=%d:%d", line+1, column+1), absPath)
	cmd.Dir = ct.workspaceRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("crystal tool hierarchy failed: %v", err)
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

// parseTextContext parses text-based context output
func (ct *CrystalTool) parseTextContext(output string) (*ContextInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("no context information")
	}

	info := &ContextInfo{
		Description: strings.Join(lines, "\n"),
	}

	// Try to extract type information from text
	for _, line := range lines {
		if strings.Contains(line, "class") || strings.Contains(line, "struct") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				info.Type = "class"
				info.Name = parts[1]
			}
		} else if strings.Contains(line, "def") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				info.Type = "method"
				info.Name = parts[1]
			}
		}
	}

	return info, nil
}

// parseLocations parses location output from Crystal tools
func (ct *CrystalTool) parseLocations(output string) ([]Location, error) {
	var locations []Location
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		// Parse format: filename:line:column
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			// Convert to URI format
			uri := "file://" + filepath.ToSlash(parts[0])

			location := Location{
				URI: uri,
				Range: Range{
					Start: Position{Line: 0, Character: 0}, // Would need to parse line/column
					End:   Position{Line: 0, Character: 0},
				},
			}
			locations = append(locations, location)
		}
	}

	return locations, nil
}

// findCrystalExecutable finds the Crystal executable in PATH
func findCrystalExecutable() string {
	// Try common Crystal executable names
	names := []string{"crystal", "crystal.exe"}

	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}

	// Try common installation paths on Windows
	commonPaths := []string{
		"C:\\crystal\\bin\\crystal.exe",
		"C:\\Program Files\\crystal\\bin\\crystal.exe",
		"C:\\scoop\\apps\\crystal\\current\\bin\\crystal.exe",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// IsCrystalAvailable checks if Crystal compiler is available
func (ct *CrystalTool) IsCrystalAvailable() bool {
	return ct.crystalPath != ""
}
