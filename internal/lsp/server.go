package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/parser"
	_ "github.com/issueye/goscript/internal/stdlib"
)

// Server implements a small Language Server Protocol endpoint for GoScript.
type Server struct {
	in  *bufio.Reader
	out io.Writer
	log io.Writer

	mu        sync.Mutex
	nextID    int64
	shutdown  bool
	documents map[string]string
}

// NewServer creates a server that speaks LSP over in/out.
func NewServer(in io.Reader, out io.Writer, log io.Writer) *Server {
	if log == nil {
		log = io.Discard
	}
	return &Server{
		in:        bufio.NewReader(in),
		out:       out,
		log:       log,
		documents: make(map[string]string),
	}
}

// Run serves requests until the client sends exit or the input stream closes.
func (s *Server) Run() error {
	for {
		msg, err := readMessage(s.in)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if errors.Is(err, errEmptyMessage) {
				continue
			}
			return err
		}
		if len(bytes.TrimSpace(msg)) == 0 {
			continue
		}
		if err := s.handle(msg); err != nil {
			fmt.Fprintln(s.log, err)
		}
	}
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) handle(msg []byte) error {
	var req rpcMessage
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	switch req.Method {
	case "initialize":
		return s.reply(req.ID, initializeResult())
	case "initialized":
		return nil
	case "shutdown":
		s.mu.Lock()
		s.shutdown = true
		s.mu.Unlock()
		return s.reply(req.ID, nil)
	case "exit":
		os.Exit(0)
		return nil
	case "textDocument/didOpen":
		var p didOpenTextDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return err
		}
		s.setDocument(p.TextDocument.URI, p.TextDocument.Text)
		return s.publishDiagnostics(p.TextDocument.URI, p.TextDocument.Text)
	case "textDocument/didChange":
		var p didChangeTextDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return err
		}
		if len(p.ContentChanges) == 0 {
			return nil
		}
		text := p.ContentChanges[len(p.ContentChanges)-1].Text
		s.setDocument(p.TextDocument.URI, text)
		return s.publishDiagnostics(p.TextDocument.URI, text)
	case "textDocument/didSave":
		var p didSaveTextDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return err
		}
		text := p.Text
		if text == "" {
			text = s.document(p.TextDocument.URI)
		}
		return s.publishDiagnostics(p.TextDocument.URI, text)
	case "textDocument/didClose":
		var p didCloseTextDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return err
		}
		s.removeDocument(p.TextDocument.URI)
		return s.notify("textDocument/publishDiagnostics", publishDiagnosticsParams{URI: p.TextDocument.URI, Diagnostics: []diagnostic{}})
	case "textDocument/completion":
		return s.reply(req.ID, completionItems())
	case "textDocument/hover":
		var p hoverParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return err
		}
		return s.reply(req.ID, s.hover(p.TextDocument.URI, p.Position))
	default:
		if len(req.ID) > 0 {
			return s.replyError(req.ID, -32601, "method not found: "+req.Method)
		}
		return nil
	}
}

func (s *Server) setDocument(uri, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[uri] = text
}

func (s *Server) document(uri string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.documents[uri]
}

func (s *Server) removeDocument(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.documents, uri)
}

func (s *Server) publishDiagnostics(uri, text string) error {
	return s.notify("textDocument/publishDiagnostics", publishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnosticsFor(uri, text),
	})
}

func (s *Server) hover(uri string, pos position) any {
	text := s.document(uri)
	word := wordAtPosition(text, pos)
	if word == "" {
		return nil
	}
	if doc, ok := hoverDocs[word]; ok {
		return hover{Contents: markupContent{Kind: "markdown", Value: doc}}
	}
	if strings.HasPrefix(word, "@std/") {
		if signatures, ok := module.GetNativeAPIDoc(word); ok {
			return hover{Contents: markupContent{Kind: "markdown", Value: "```goscript\nimport ... from \"" + word + "\"\n```\n\n" + strings.Join(signatures, "\n\n")}}
		}
	}
	return nil
}

func (s *Server) reply(id json.RawMessage, result any) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return writeMessage(s.out, rpcResponse{JSONRPC: "2.0", ID: id, Result: data})
}

func (s *Server) replyError(id json.RawMessage, code int, message string) error {
	return writeMessage(s.out, rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}})
}

func (s *Server) notify(method string, params any) error {
	msg := struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{JSONRPC: "2.0", Method: method, Params: params}
	return writeMessage(s.out, msg)
}

var errEmptyMessage = errors.New("empty message")

func readMessage(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	sawHeader := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		sawHeader = true
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, err
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		if !sawHeader {
			return nil, errEmptyMessage
		}
		return nil, errors.New("missing Content-Length")
	}
	body := make([]byte, contentLength)
	_, err := io.ReadFull(r, body)
	return body, err
}

func writeMessage(w io.Writer, msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type documentRange struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type diagnostic struct {
	Range    documentRange `json:"range"`
	Severity int           `json:"severity"`
	Source   string        `json:"source"`
	Message  string        `json:"message"`
}

type publishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []diagnostic `json:"diagnostics"`
}

type textDocumentIdentifier struct {
	URI string `json:"uri"`
}

type versionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version,omitempty"`
}

type textDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type textDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type didOpenTextDocumentParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

type didChangeTextDocumentParams struct {
	TextDocument   versionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []textDocumentContentChangeEvent `json:"contentChanges"`
}

type didSaveTextDocumentParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

type didCloseTextDocumentParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type hoverParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     position               `json:"position"`
}

type hover struct {
	Contents markupContent `json:"contents"`
}

type markupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type completionItem struct {
	Label         string        `json:"label"`
	Kind          int           `json:"kind,omitempty"`
	Detail        string        `json:"detail,omitempty"`
	Documentation markupContent `json:"documentation,omitempty"`
}

func initializeResult() any {
	return map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync": map[string]any{
				"openClose": true,
				"change":    1,
				"save": map[string]any{
					"includeText": true,
				},
			},
			"completionProvider": map[string]any{
				"triggerCharacters": []string{".", "\"", "'", "@", "/"},
			},
			"hoverProvider": true,
		},
		"serverInfo": map[string]string{
			"name":    "goscript-lsp",
			"version": "0.1.0",
		},
	}
}

func diagnosticsFor(uri, text string) []diagnostic {
	file := uriToPath(uri)
	l := lexer.New(text)
	p := parser.New(l, file)
	prog := p.ParseProgram()

	var out []diagnostic
	for _, errText := range l.Errors() {
		out = append(out, diagnosticFromError(text, "goscript lexer", errText))
	}
	for _, errText := range prog.Errors {
		out = append(out, diagnosticFromError(text, "goscript parser", errText))
	}
	return out
}

var (
	lexerErrRE = regexp.MustCompile(`^Lexer error at line (\d+) col (\d+):\s*(.*)$`)
	parseErrRE = regexp.MustCompile(`^(.+):(\d+):(\d+):\s*(.*)$`)
)

func diagnosticFromError(text, source, errText string) diagnostic {
	line, col, msg := 1, 1, errText
	if match := lexerErrRE.FindStringSubmatch(errText); len(match) == 4 {
		line = atoiDefault(match[1], 1)
		col = atoiDefault(match[2], 1)
		msg = match[3]
	} else if match := parseErrRE.FindStringSubmatch(errText); len(match) == 5 {
		line = atoiDefault(match[2], 1)
		col = atoiDefault(match[3], 1)
		msg = match[4]
	}
	start := position{Line: max(0, line-1), Character: max(0, col-1)}
	end := start
	end.Character = min(lineLength(text, start.Line), start.Character+1)
	return diagnostic{
		Range:    documentRange{Start: start, End: end},
		Severity: 1,
		Source:   source,
		Message:  msg,
	}
}

func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return uri
	}
	path, err := url.PathUnescape(u.Path)
	if err != nil {
		path = u.Path
	}
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.FromSlash(path)
}

func completionItems() []completionItem {
	items := make([]completionItem, 0, len(keywordDocs)+len(globalDocs)+len(module.ListNative()))
	for _, kw := range keywordOrder {
		items = append(items, completionItem{Label: kw, Kind: 14, Detail: "GoScript keyword", Documentation: markdown(keywordDocs[kw])})
	}
	for _, name := range globalOrder {
		items = append(items, completionItem{Label: name, Kind: 3, Detail: "GoScript global", Documentation: markdown(globalDocs[name])})
	}
	for _, path := range module.ListNative() {
		detail := "native module"
		doc := "Native module `" + path + "`."
		if signatures, ok := module.GetNativeAPIDoc(path); ok && len(signatures) > 0 {
			doc = strings.Join(signatures, "\n\n")
		}
		items = append(items, completionItem{Label: path, Kind: 9, Detail: detail, Documentation: markdown(doc)})
	}
	return items
}

func markdown(value string) markupContent {
	return markupContent{Kind: "markdown", Value: value}
}

func wordAtPosition(text string, pos position) string {
	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return ""
	}
	line := strings.TrimRight(lines[pos.Line], "\r")
	if pos.Character < 0 {
		return ""
	}
	bytePos := utf16CharacterToByte(line, pos.Character)
	if bytePos > len(line) {
		bytePos = len(line)
	}
	if bytePos > 0 && bytePos == len(line) {
		_, size := utf8.DecodeLastRuneInString(line)
		bytePos -= size
	}
	start := bytePos
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(line[:start])
		if !isWordRune(r) {
			break
		}
		start -= size
	}
	end := bytePos
	for end < len(line) {
		r, size := utf8.DecodeRuneInString(line[end:])
		if !isWordRune(r) {
			break
		}
		end += size
	}
	return line[start:end]
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$' || r == '@' || r == '/'
}

func utf16CharacterToByte(s string, character int) int {
	if character <= 0 {
		return 0
	}
	units := 0
	for i, r := range s {
		if units >= character {
			return i
		}
		if r <= 0xFFFF {
			units++
		} else {
			units += 2
		}
	}
	return len(s)
}

func lineLength(text string, line int) int {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return 0
	}
	return len([]rune(strings.TrimRight(lines[line], "\r")))
}

func atoiDefault(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var keywordOrder = []string{
	"let", "const", "var", "function", "class", "extends", "if", "else", "while", "for",
	"in", "of", "return", "break", "continue", "try", "catch", "finally", "throw",
	"async", "await", "import", "export", "from", "as", "new", "this", "super",
	"match", "default", "true", "false", "null", "undefined", "typeof", "delete", "void",
}

var keywordDocs = map[string]string{
	"let":       "Declares a block-scoped variable.",
	"const":     "Declares a block-scoped constant.",
	"var":       "Declares a variable.",
	"function":  "Declares a named function.",
	"class":     "Declares a class.",
	"extends":   "Adds a superclass to a class declaration.",
	"if":        "Starts a conditional branch.",
	"else":      "Starts the fallback branch for an `if` statement.",
	"while":     "Repeats a block while a condition is truthy.",
	"for":       "Starts a loop.",
	"in":        "Iterates keys or checks membership depending on context.",
	"of":        "Iterates values in a `for ... of` loop.",
	"return":    "Returns from the current function.",
	"break":     "Exits a loop or labeled statement.",
	"continue":  "Continues with the next loop iteration.",
	"try":       "Starts exception handling.",
	"catch":     "Handles a thrown error.",
	"finally":   "Runs cleanup after `try`/`catch`.",
	"throw":     "Throws an error value.",
	"async":     "Declares an async function.",
	"await":     "Waits for a promise inside async code.",
	"import":    "Imports values from another module.",
	"export":    "Exports declarations from a module.",
	"from":      "Introduces the source module in an import/export statement.",
	"as":        "Aliases an imported or exported name.",
	"new":       "Constructs an object.",
	"this":      "References the current receiver.",
	"super":     "References superclass behavior.",
	"match":     "Starts a pattern match expression.",
	"default":   "Default export or fallback marker.",
	"true":      "Boolean true.",
	"false":     "Boolean false.",
	"null":      "Null value.",
	"undefined": "Undefined value.",
	"typeof":    "Returns the runtime type name.",
	"delete":    "Deletes a property.",
	"void":      "Evaluates an expression and returns undefined.",
}

var globalOrder = []string{
	"console", "println", "print", "require", "String", "Number", "Boolean", "Date", "RegExp",
	"Error", "TypeError", "RangeError", "ReferenceError", "SyntaxError", "parseInt",
	"parseFloat", "isNaN", "isFinite", "encodeURI", "decodeURI", "encodeURIComponent",
	"decodeURIComponent", "Math", "JSON", "Object", "Array", "Map", "Set", "Promise",
	"setTimeout", "clearTimeout", "setInterval", "clearInterval", "queueMicrotask",
	"sleep", "go", "makeChannel", "makeWaitGroup",
}

var globalDocs = map[string]string{
	"console":            "`console.log(...)` and related console helpers.",
	"println":            "Prints values followed by a newline.",
	"print":              "Prints values without adding a newline.",
	"require":            "Loads a module by path.",
	"String":             "String constructor and string helpers.",
	"Number":             "Number constructor and numeric helpers.",
	"Boolean":            "Boolean constructor.",
	"Date":               "Date constructor.",
	"RegExp":             "Regular expression constructor.",
	"Error":              "Creates a generic error.",
	"TypeError":          "Creates a type error.",
	"RangeError":         "Creates a range error.",
	"ReferenceError":     "Creates a reference error.",
	"SyntaxError":        "Creates a syntax error.",
	"parseInt":           "Parses an integer from a string.",
	"parseFloat":         "Parses a number from a string.",
	"isNaN":              "Checks whether a value is NaN-like.",
	"isFinite":           "Checks whether a value is finite.",
	"encodeURI":          "Encodes a URI.",
	"decodeURI":          "Decodes a URI.",
	"encodeURIComponent": "Encodes a URI component.",
	"decodeURIComponent": "Decodes a URI component.",
	"Math":               "Math constants and functions.",
	"JSON":               "JSON parse and stringify helpers.",
	"Object":             "Object constructor and object helpers.",
	"Array":              "Array constructor and array helpers.",
	"Map":                "Map constructor.",
	"Set":                "Set constructor.",
	"Promise":            "Promise constructor.",
	"setTimeout":         "Schedules a callback after a delay.",
	"clearTimeout":       "Cancels a timeout.",
	"setInterval":        "Schedules a repeated callback.",
	"clearInterval":      "Cancels an interval.",
	"queueMicrotask":     "Schedules a microtask callback.",
	"sleep":              "Blocks for the specified milliseconds.",
	"go":                 "Runs a function asynchronously.",
	"makeChannel":        "Creates a channel.",
	"makeWaitGroup":      "Creates a wait group.",
}

var hoverDocs = func() map[string]string {
	out := make(map[string]string, len(keywordDocs)+len(globalDocs))
	for k, v := range keywordDocs {
		out[k] = v
	}
	for k, v := range globalDocs {
		out[k] = v
	}
	return out
}()
