package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/jsonrpc2"
)

// Server represents the Crystal Language Server
type Server struct {
	conn   *jsonrpc2.Conn
	logger *log.Logger

	// Document management
	documents map[string]*TextDocumentItem

	// Crystal analyzer
	analyzer *CrystalAnalyzer
}

// NewServer creates a new Crystal Language Server
func NewServer() *Server {
	return &Server{
		logger:    log.New(os.Stderr, "[Crystal LSP] ", log.LstdFlags),
		documents: make(map[string]*TextDocumentItem),
		analyzer:  NewCrystalAnalyzer(),
	}
}

// Start starts the language server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Println("Crystal Language Server starting...")

	// Create JSON-RPC connection over stdio
	conn := jsonrpc2.NewConn(
		ctx,
		jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
		s,
	)

	s.conn = conn

	// Wait for connection to close
	<-conn.DisconnectNotify()
	s.logger.Println("Crystal Language Server stopped")

	return nil
}

// Handle implements jsonrpc2.Handler
func (s *Server) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(ctx, conn, req)
	case "initialized":
		s.handleInitialized(ctx, conn, req)
	case "textDocument/didOpen":
		s.handleTextDocumentDidOpen(ctx, conn, req)
	case "textDocument/didChange":
		s.handleTextDocumentDidChange(ctx, conn, req)
	case "textDocument/didClose":
		s.handleTextDocumentDidClose(ctx, conn, req)
	case "textDocument/completion":
		s.handleTextDocumentCompletion(ctx, conn, req)
	case "textDocument/hover":
		s.handleTextDocumentHover(ctx, conn, req)
	case "textDocument/signatureHelp":
		s.handleTextDocumentSignatureHelp(ctx, conn, req)
	case "textDocument/definition":
		s.handleTextDocumentDefinition(ctx, conn, req)
	case "textDocument/documentSymbol":
		s.handleTextDocumentSymbol(ctx, conn, req)
	case "textDocument/formatting":
		s.handleTextDocumentFormatting(ctx, conn, req)
	case "textDocument/foldingRange":
		s.handleTextDocumentFoldingRange(ctx, conn, req)
	case "textDocument/references":
		s.handleTextDocumentReferences(ctx, conn, req)
	case "textDocument/documentHighlight":
		s.handleTextDocumentHighlight(ctx, conn, req)
	case "shutdown":
		s.handleShutdown(ctx, conn, req)
	case "exit":
		s.handleExit(ctx, conn, req)
	case "workspace/didChangeConfiguration":
		s.handleWorkspaceDidChangeConfiguration(ctx, conn, req)
	case "$/setTrace":
		s.handleSetTrace(ctx, conn, req)
	case "$/cancelRequest":
		s.handleCancelRequest(ctx, conn, req)
	default:
		s.logger.Printf("Unhandled method: %s", req.Method)
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeMethodNotFound,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		})
	}
}

func (s *Server) handleInitialize(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		ProcessID             *int   `json:"processId"`
		RootPath              string `json:"rootPath"`
		RootURI               string `json:"rootUri"`
		InitializationOptions any    `json:"initializationOptions"`
		Capabilities          any    `json:"capabilities"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		})
		return
	}

	s.logger.Printf("Initializing with root: %s", params.RootURI)

	result := map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync": map[string]any{
				"openClose": true,
				"change":    2, // Incremental
			},
			"completionProvider": map[string]any{
				"resolveProvider":   false,
				"triggerCharacters": []string{".", ":"},
			},
			"hoverProvider":              true,
			"definitionProvider":         true,
			"referencesProvider":         true,
			"documentHighlightProvider":  true,
			"documentSymbolProvider":     true,
			"documentFormattingProvider": true,
			"foldingRangeProvider":       true,
			"signatureHelpProvider": map[string]any{
				"triggerCharacters": []string{"(", ","},
			},
		},
		"serverInfo": map[string]any{
			"name":    "Crystal Language Server",
			"version": "0.1.0",
		},
	}

	conn.Reply(ctx, req.ID, result)
}

func (s *Server) handleInitialized(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	s.logger.Println("Server initialized")
}

func (s *Server) handleTextDocumentDidOpen(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentItem `json:"textDocument"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		s.logger.Printf("Error unmarshaling didOpen params: %v", err)
		return
	}

	s.documents[params.TextDocument.URI] = &params.TextDocument
	s.logger.Printf("Opened document: %s", params.TextDocument.URI)

	// Analyze the document and send diagnostics
	diagnostics := s.analyzer.AnalyzeDocument(&params.TextDocument)
	s.publishDiagnostics(ctx, conn, params.TextDocument.URI, diagnostics)
}

func (s *Server) handleTextDocumentDidChange(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
		ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		s.logger.Printf("Error unmarshaling didChange params: %v", err)
		return
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		s.logger.Printf("Document not found: %s", params.TextDocument.URI)
		return
	}

	// Apply changes
	for _, change := range params.ContentChanges {
		if change.Range == nil {
			// Full document update
			doc.Text = change.Text
		} else {
			// Incremental update
			doc.Text = s.applyTextChange(doc.Text, change)
		}
	}

	doc.Version = params.TextDocument.Version

	// Re-analyze and send diagnostics
	diagnostics := s.analyzer.AnalyzeDocument(doc)
	s.publishDiagnostics(ctx, conn, params.TextDocument.URI, diagnostics)
}

func (s *Server) handleTextDocumentDidClose(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		s.logger.Printf("Error unmarshaling didClose params: %v", err)
		return
	}

	delete(s.documents, params.TextDocument.URI)
	s.logger.Printf("Closed document: %s", params.TextDocument.URI)
}

func (s *Server) handleTextDocumentCompletion(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		})
		return
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		conn.Reply(ctx, req.ID, CompletionList{Items: []CompletionItem{}})
		return
	}

	completions := s.analyzer.GetCompletions(doc, params.Position)
	conn.Reply(ctx, req.ID, completions)
}

func (s *Server) handleTextDocumentHover(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		})
		return
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		conn.Reply(ctx, req.ID, nil)
		return
	}

	hover := s.analyzer.GetHover(doc, params.Position)
	conn.Reply(ctx, req.ID, hover)
}

func (s *Server) handleTextDocumentSignatureHelp(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		})
		return
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		conn.Reply(ctx, req.ID, nil)
		return
	}

	signatureHelp := s.analyzer.GetSignatureHelp(doc, params.Position)
	conn.Reply(ctx, req.ID, signatureHelp)
}

func (s *Server) handleTextDocumentDefinition(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		})
		return
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		conn.Reply(ctx, req.ID, []Location{})
		return
	}

	definitions := s.analyzer.GetDefinition(doc, params.Position)
	conn.Reply(ctx, req.ID, definitions)
}

func (s *Server) handleTextDocumentSymbol(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		})
		return
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		conn.Reply(ctx, req.ID, []SymbolInformation{})
		return
	}

	symbols := s.analyzer.GetDocumentSymbols(doc)
	conn.Reply(ctx, req.ID, symbols)
}

func (s *Server) handleTextDocumentFormatting(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Options      any                    `json:"options"` // FormattingOptions
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		})
		return
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		conn.Reply(ctx, req.ID, []TextEdit{})
		return
	}

	edits := s.analyzer.GetDocumentFormat(doc)
	conn.Reply(ctx, req.ID, edits)
}

func (s *Server) handleTextDocumentFoldingRange(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		})
		return
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		conn.Reply(ctx, req.ID, []FoldingRange{})
		return
	}

	ranges := s.analyzer.GetFoldingRanges(doc)
	conn.Reply(ctx, req.ID, ranges)
}

func (s *Server) handleTextDocumentReferences(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
		Context      struct {
			IncludeDeclaration bool `json:"includeDeclaration"`
		} `json:"context"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		})
		return
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		conn.Reply(ctx, req.ID, []Location{})
		return
	}

	references := s.analyzer.GetReferences(doc, params.Position, params.Context.IncludeDeclaration)
	conn.Reply(ctx, req.ID, references)
}

func (s *Server) handleTextDocumentHighlight(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
	}

	if err := json.Unmarshal(*req.Params, &params); err != nil {
		conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		})
		return
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		conn.Reply(ctx, req.ID, []DocumentHighlight{})
		return
	}

	highlights := s.analyzer.GetDocumentHighlights(doc, params.Position)
	conn.Reply(ctx, req.ID, highlights)
}

func (s *Server) handleShutdown(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	s.logger.Println("Shutdown requested")
	conn.Reply(ctx, req.ID, nil)
}

func (s *Server) handleExit(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	s.logger.Println("Exit requested")
	os.Exit(0)
}

func (s *Server) handleWorkspaceDidChangeConfiguration(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// Handle workspace configuration changes
	s.logger.Println("Workspace configuration changed")
}

func (s *Server) handleSetTrace(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// Handle trace level changes (for debugging)
	// This is a notification, so no response needed
}

func (s *Server) handleCancelRequest(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// Handle request cancellation
	// This is a notification, so no response needed
}

func (s *Server) publishDiagnostics(ctx context.Context, conn *jsonrpc2.Conn, uri string, diagnostics []Diagnostic) {
	// Always ensure we have a non-nil slice
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}

	params := map[string]any{
		"uri":         uri,
		"diagnostics": diagnostics,
	}

	// s.logger.Printf("Publishing %d diagnostics for %s", len(diagnostics), uri)
	conn.Notify(ctx, "textDocument/publishDiagnostics", params)
}

func (s *Server) applyTextChange(text string, change TextDocumentContentChangeEvent) string {
	// This is a simplified implementation
	// In a real implementation, you'd need to properly handle line/character positions
	if change.Range == nil {
		return change.Text
	}

	lines := splitLines(text)

	// Calculate start and end offsets
	startOffset := 0
	for i := 0; i < change.Range.Start.Line && i < len(lines); i++ {
		startOffset += len(lines[i]) + 1 // +1 for newline
	}
	startOffset += change.Range.Start.Character

	endOffset := 0
	for i := 0; i < change.Range.End.Line && i < len(lines); i++ {
		endOffset += len(lines[i]) + 1 // +1 for newline
	}
	endOffset += change.Range.End.Character

	if startOffset > len(text) {
		startOffset = len(text)
	}
	if endOffset > len(text) {
		endOffset = len(text)
	}

	return text[:startOffset] + change.Text + text[endOffset:]
}

func splitLines(text string) []string {
	var lines []string
	start := 0

	for i, r := range text {
		if r == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}

	if start < len(text) {
		lines = append(lines, text[start:])
	}

	return lines
}

// stdrwc implements io.ReadWriteCloser for stdio
type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}
