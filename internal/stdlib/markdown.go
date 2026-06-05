package stdlib

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type markdownDiagnostic struct {
	message string
	line    int
}

func init() {
	module.RegisterNative("@std/markdown", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initMarkdownModule(exports)
		return exports, nil
	})
}

func initMarkdownModule(exports *object.Hash) {
	setHashMember(exports, "parse", &object.Builtin{Name: "markdown.parse", Fn: markdownParse})
	setHashMember(exports, "renderTerminal", &object.Builtin{Name: "markdown.renderTerminal", Fn: markdownRenderTerminal})
	setHashMember(exports, "createStream", &object.Builtin{Name: "markdown.createStream", Fn: markdownCreateStream})
	setHashMember(exports, "fromHTML", &object.Builtin{Name: "markdown.fromHTML", Fn: markdownFromHTML})
}

type markdownStreamState struct {
	source strings.Builder
	width  int
}

func markdownParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	source, errObj := requiredString(pos, "markdown.parse", args, 0, "source")
	if errObj != nil {
		return errObj
	}
	doc := parseMarkdownDocument(source)
	return markdownDocObject(doc.children, doc.diagnostics)
}

func markdownRenderTerminal(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	source, errObj := requiredString(pos, "markdown.renderTerminal", args, 0, "source")
	if errObj != nil {
		return errObj
	}
	width, errObj := markdownWidthOption(pos, "markdown.renderTerminal", args, 1)
	if errObj != nil {
		return errObj
	}
	return markdownRenderTerminalObject(source, width)
}

func markdownCreateStream(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	width, errObj := markdownWidthOption(pos, "markdown.createStream", args, 0)
	if errObj != nil {
		return errObj
	}
	state := &markdownStreamState{width: width}
	extra := &object.GoObject{Value: state}
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "__markdownStream", extra)
	setHashMember(out, "append", &object.Builtin{Name: "markdown.stream.append", Fn: markdownStreamAppend, Extra: extra})
	setHashMember(out, "snapshot", &object.Builtin{Name: "markdown.stream.snapshot", Fn: markdownStreamSnapshot, Extra: extra})
	setHashMember(out, "finalize", &object.Builtin{Name: "markdown.stream.finalize", Fn: markdownStreamFinalize, Extra: extra})
	return out
}

func markdownFromHTML(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	source, errObj := requiredString(pos, "markdown.fromHTML", args, 0, "html")
	if errObj != nil {
		return errObj
	}
	opts := markdownHTMLOptions{includeLinks: true, maxChars: 20000}
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		hash, ok := args[1].(*object.Hash)
		if !ok {
			return object.NewError(pos, "markdown.fromHTML: options must be an object")
		}
		if value, ok := hashValue(hash, "includeLinks"); ok && value != object.UNDEFINED && value != object.NULL {
			include, ok := value.(*object.Boolean)
			if !ok {
				return object.NewError(pos, "markdown.fromHTML: includeLinks must be a boolean")
			}
			opts.includeLinks = include.Value
		}
		if value, ok := hashValue(hash, "maxChars"); ok && value != object.UNDEFINED && value != object.NULL {
			maxChars, ok := value.(*object.Number)
			if !ok {
				return object.NewError(pos, "markdown.fromHTML: maxChars must be a number")
			}
			opts.maxChars = int(maxChars.Value)
		}
		if value, ok := hashValue(hash, "baseUrl"); ok && value != object.UNDEFINED && value != object.NULL {
			baseURL, ok := value.(*object.String)
			if !ok {
				return object.NewError(pos, "markdown.fromHTML: baseUrl must be a string")
			}
			opts.baseURL = baseURL.Value
		}
	}
	return &object.String{Value: htmlToMarkdown(source, opts)}
}

type markdownHTMLOptions struct {
	baseURL      string
	includeLinks bool
	maxChars     int
}

func markdownStreamAppend(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	state, errObj := boundMarkdownStream(pos, env, "markdown.stream.append")
	if errObj != nil {
		return errObj
	}
	delta, errObj := requiredString(pos, "markdown.stream.append", args, 0, "delta")
	if errObj != nil {
		return errObj
	}
	state.source.WriteString(delta)
	return &object.Number{Value: float64(state.source.Len())}
}

func markdownStreamSnapshot(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	state, errObj := boundMarkdownStream(pos, env, "markdown.stream.snapshot")
	if errObj != nil {
		return errObj
	}
	return markdownRenderTerminalObject(state.source.String(), state.width)
}

func markdownStreamFinalize(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	state, errObj := boundMarkdownStream(pos, env, "markdown.stream.finalize")
	if errObj != nil {
		return errObj
	}
	doc := parseMarkdownDocument(state.source.String())
	out := markdownRenderTerminalObject(state.source.String(), state.width)
	if hash, ok := out.(*object.Hash); ok {
		setHashMember(hash, "document", markdownDocObject(doc.children, doc.diagnostics))
	}
	return out
}

func boundMarkdownStream(pos ast.Position, env *object.Environment, name string) (*markdownStreamState, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing stream receiver", name)
	}
	state, ok := goObj.Value.(*markdownStreamState)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid stream receiver", name)
	}
	return state, nil
}

func markdownWidthOption(pos ast.Position, name string, args []object.Object, index int) (int, *object.Error) {
	width := 80
	if len(args) > index && args[index] != object.UNDEFINED && args[index] != object.NULL {
		opts, ok := args[index].(*object.Hash)
		if !ok {
			return width, object.NewError(pos, "%s: options must be an object", name)
		}
		if value, ok := hashValue(opts, "width"); ok && value != object.UNDEFINED && value != object.NULL {
			n, ok := value.(*object.Number)
			if !ok {
				return width, object.NewError(pos, "%s: width must be a number", name)
			}
			width = int(n.Value)
		}
	}
	if width < 1 {
		width = 1
	}
	return width, nil
}

func markdownRenderTerminalObject(source string, width int) object.Object {
	doc := parseMarkdownDocument(source)
	lines, headings, links := renderMarkdownTerminalLines(doc.children, width)
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "lines", strSliceToArray(lines))
	setHashMember(out, "width", &object.Number{Value: float64(width)})
	setHashMember(out, "headings", markdownHeadingArray(headings))
	setHashMember(out, "links", markdownLinkArray(links))
	setHashMember(out, "diagnostics", markdownDiagnosticsObject(doc.diagnostics))
	return out
}

type markdownDocument struct {
	children    []*object.Hash
	diagnostics []markdownDiagnostic
}

func parseMarkdownDocument(source string) markdownDocument {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")
	lines := strings.Split(source, "\n")
	children, diagnostics := parseMarkdownBlocks(lines, 0)
	return markdownDocument{children: children, diagnostics: diagnostics}
}

func parseMarkdownBlocks(lines []string, offset int) ([]*object.Hash, []markdownDiagnostic) {
	var nodes []*object.Hash
	var diagnostics []markdownDiagnostic
	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		startLine := offset + i + 1
		if trimmed == "" {
			i++
			continue
		}
		if marker, lang, ok := markdownFenceStart(trimmed); ok {
			codeLines := []string{}
			j := i + 1
			closed := false
			for ; j < len(lines); j++ {
				if markdownFenceClose(strings.TrimSpace(lines[j]), marker) {
					closed = true
					break
				}
				codeLines = append(codeLines, lines[j])
			}
			endLine := offset + j + 1
			if !closed {
				endLine = offset + len(lines)
				diagnostics = append(diagnostics, markdownDiagnostic{message: "unclosed code fence", line: startLine})
				i = len(lines)
			} else {
				i = j + 1
			}
			nodes = append(nodes, markdownCodeNode(strings.Join(codeLines, "\n"), lang, startLine, endLine))
			continue
		}
		if level, text, ok := markdownHeading(trimmed); ok {
			nodes = append(nodes, markdownInlineBlockNode("heading", parseMarkdownInline(text), startLine, startLine, map[string]object.Object{
				"level": &object.Number{Value: float64(level)},
			}))
			i++
			continue
		}
		if markdownIsHR(trimmed) {
			nodes = append(nodes, markdownSimpleNode("hr", startLine, startLine))
			i++
			continue
		}
		if markdownIsBlockquote(trimmed) {
			inner := []string{}
			j := i
			for ; j < len(lines); j++ {
				t := strings.TrimSpace(lines[j])
				if t == "" {
					inner = append(inner, "")
					continue
				}
				if !markdownIsBlockquote(t) {
					break
				}
				inner = append(inner, markdownBlockquoteText(t))
			}
			children, childDiagnostics := parseMarkdownBlocks(inner, offset+i)
			diagnostics = append(diagnostics, childDiagnostics...)
			nodes = append(nodes, markdownChildrenNode("blockquote", children, startLine, offset+j))
			i = j
			continue
		}
		if item, ok := markdownListItem(trimmed); ok {
			j := i
			list := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
			setHashMember(list, "type", &object.String{Value: "list"})
			setHashMember(list, "ordered", object.NativeBool(item.ordered))
			setHashMember(list, "startLine", &object.Number{Value: float64(startLine)})
			children := []*object.Hash{}
			for ; j < len(lines); j++ {
				nextTrimmed := strings.TrimSpace(lines[j])
				next, ok := markdownListItem(nextTrimmed)
				if !ok || next.ordered != item.ordered {
					break
				}
				children = append(children, markdownInlineBlockNode("list_item", parseMarkdownInline(next.text), offset+j+1, offset+j+1, nil))
			}
			setHashMember(list, "endLine", &object.Number{Value: float64(offset + j)})
			setHashMember(list, "children", markdownHashArray(children))
			nodes = append(nodes, list)
			i = j
			continue
		}

		j := i
		paragraphLines := []string{}
		for ; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			if t == "" || markdownStartsBlock(t) {
				break
			}
			paragraphLines = append(paragraphLines, strings.TrimRight(lines[j], " \t"))
		}
		text := strings.Join(paragraphLines, "\n")
		nodes = append(nodes, markdownInlineBlockNode("paragraph", parseMarkdownInline(text), startLine, offset+j, nil))
		i = j
	}
	return nodes, diagnostics
}

func markdownDocObject(children []*object.Hash, diagnostics []markdownDiagnostic) *object.Hash {
	doc := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(doc, "type", &object.String{Value: "document"})
	setHashMember(doc, "children", markdownHashArray(children))
	setHashMember(doc, "diagnostics", markdownDiagnosticsObject(diagnostics))
	return doc
}

func markdownSimpleNode(typ string, startLine, endLine int) *object.Hash {
	node := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(node, "type", &object.String{Value: typ})
	setHashMember(node, "startLine", &object.Number{Value: float64(startLine)})
	setHashMember(node, "endLine", &object.Number{Value: float64(endLine)})
	return node
}

func markdownCodeNode(code, lang string, startLine, endLine int) *object.Hash {
	node := markdownSimpleNode("code", startLine, endLine)
	setHashMember(node, "text", &object.String{Value: code})
	setHashMember(node, "lang", &object.String{Value: lang})
	return node
}

func markdownChildrenNode(typ string, children []*object.Hash, startLine, endLine int) *object.Hash {
	node := markdownSimpleNode(typ, startLine, endLine)
	setHashMember(node, "children", markdownHashArray(children))
	return node
}

func markdownInlineBlockNode(typ string, children []object.Object, startLine, endLine int, extra map[string]object.Object) *object.Hash {
	node := markdownSimpleNode(typ, startLine, endLine)
	setHashMember(node, "children", &object.Array{Elements: children})
	for key, value := range extra {
		setHashMember(node, key, value)
	}
	return node
}

func markdownHashArray(nodes []*object.Hash) *object.Array {
	elements := make([]object.Object, len(nodes))
	for i, node := range nodes {
		elements[i] = node
	}
	return &object.Array{Elements: elements}
}

func markdownDiagnosticsObject(diagnostics []markdownDiagnostic) *object.Array {
	elements := make([]object.Object, len(diagnostics))
	for i, diagnostic := range diagnostics {
		out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(out, "message", &object.String{Value: diagnostic.message})
		setHashMember(out, "line", &object.Number{Value: float64(diagnostic.line)})
		elements[i] = out
	}
	return &object.Array{Elements: elements}
}

func parseMarkdownInline(text string) []object.Object {
	var nodes []object.Object
	for len(text) > 0 {
		if strings.HasPrefix(text, "  \n") {
			nodes = append(nodes, markdownInlineNode("hardbreak", "", nil))
			text = text[3:]
			continue
		}
		if strings.HasPrefix(text, "\n") {
			nodes = append(nodes, markdownInlineNode("softbreak", "", nil))
			text = text[1:]
			continue
		}
		if strings.HasPrefix(text, "**") {
			if end := strings.Index(text[2:], "**"); end >= 0 {
				content := text[2 : 2+end]
				nodes = append(nodes, markdownInlineContainer("strong", parseMarkdownInline(content), nil))
				text = text[2+end+2:]
				continue
			}
		}
		if strings.HasPrefix(text, "*") {
			if end := strings.Index(text[1:], "*"); end >= 0 {
				content := text[1 : 1+end]
				nodes = append(nodes, markdownInlineContainer("em", parseMarkdownInline(content), nil))
				text = text[1+end+1:]
				continue
			}
		}
		if strings.HasPrefix(text, "`") {
			if end := strings.Index(text[1:], "`"); end >= 0 {
				content := text[1 : 1+end]
				nodes = append(nodes, markdownInlineNode("code", content, nil))
				text = text[1+end+1:]
				continue
			}
		}
		if strings.HasPrefix(text, "[") {
			if node, rest, ok := markdownParseLink(text); ok {
				nodes = append(nodes, node)
				text = rest
				continue
			}
		}
		next := markdownNextInlineSpecial(text)
		chunk := text
		if next > 0 {
			chunk = text[:next]
		}
		nodes = append(nodes, markdownInlineNode("text", chunk, nil))
		text = text[len(chunk):]
	}
	return nodes
}

func markdownInlineNode(typ, text string, extra map[string]object.Object) *object.Hash {
	node := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(node, "type", &object.String{Value: typ})
	if text != "" || typ == "text" || typ == "code" {
		setHashMember(node, "text", &object.String{Value: text})
	}
	for key, value := range extra {
		setHashMember(node, key, value)
	}
	return node
}

func markdownInlineContainer(typ string, children []object.Object, extra map[string]object.Object) *object.Hash {
	node := markdownInlineNode(typ, "", extra)
	setHashMember(node, "children", &object.Array{Elements: children})
	return node
}

func markdownParseLink(text string) (object.Object, string, bool) {
	closeBracket := strings.Index(text, "](")
	if closeBracket <= 0 {
		return nil, text, false
	}
	closeParen := strings.Index(text[closeBracket+2:], ")")
	if closeParen < 0 {
		return nil, text, false
	}
	label := text[1:closeBracket]
	url := text[closeBracket+2 : closeBracket+2+closeParen]
	node := markdownInlineContainer("link", parseMarkdownInline(label), map[string]object.Object{
		"url": &object.String{Value: url},
	})
	return node, text[closeBracket+2+closeParen+1:], true
}

func markdownNextInlineSpecial(text string) int {
	next := -1
	for _, needle := range []string{"**", "*", "`", "[", "\n"} {
		if idx := strings.Index(text, needle); idx >= 0 && (next < 0 || idx < next) {
			next = idx
		}
	}
	if next <= 0 {
		return len(text)
	}
	return next
}

func markdownFenceStart(trimmed string) (marker, lang string, ok bool) {
	if strings.HasPrefix(trimmed, "```") {
		return "```", strings.TrimSpace(trimmed[3:]), true
	}
	if strings.HasPrefix(trimmed, "~~~") {
		return "~~~", strings.TrimSpace(trimmed[3:]), true
	}
	return "", "", false
}

func markdownFenceClose(trimmed, marker string) bool {
	return strings.HasPrefix(trimmed, marker)
}

func markdownHeading(trimmed string) (int, string, bool) {
	level := 0
	for level < len(trimmed) && level < 6 && trimmed[level] == '#' {
		level++
	}
	if level == 0 {
		return 0, "", false
	}
	if len(trimmed) > level && trimmed[level] != ' ' && trimmed[level] != '\t' {
		return 0, "", false
	}
	return level, strings.TrimSpace(trimmed[level:]), true
}

func markdownIsHR(trimmed string) bool {
	if len(trimmed) < 3 {
		return false
	}
	ch := rune(trimmed[0])
	if ch != '-' && ch != '*' && ch != '_' {
		return false
	}
	count := 0
	for _, r := range trimmed {
		if unicode.IsSpace(r) {
			continue
		}
		if r != ch {
			return false
		}
		count++
	}
	return count >= 3
}

func markdownIsBlockquote(trimmed string) bool {
	return strings.HasPrefix(trimmed, ">")
}

func markdownBlockquoteText(trimmed string) string {
	text := strings.TrimPrefix(trimmed, ">")
	return strings.TrimPrefix(text, " ")
}

type markdownListItemInfo struct {
	ordered bool
	text    string
}

var markdownOrderedListRE = regexp.MustCompile(`^\d+[.)]\s+(.*)$`)

func markdownListItem(trimmed string) (markdownListItemInfo, bool) {
	if len(trimmed) >= 2 && (trimmed[0] == '-' || trimmed[0] == '+' || trimmed[0] == '*') && unicode.IsSpace(rune(trimmed[1])) {
		return markdownListItemInfo{text: strings.TrimSpace(trimmed[2:])}, true
	}
	if matches := markdownOrderedListRE.FindStringSubmatch(trimmed); len(matches) == 2 {
		return markdownListItemInfo{ordered: true, text: matches[1]}, true
	}
	return markdownListItemInfo{}, false
}

func markdownStartsBlock(trimmed string) bool {
	if trimmed == "" {
		return true
	}
	if _, _, ok := markdownFenceStart(trimmed); ok {
		return true
	}
	if _, _, ok := markdownHeading(trimmed); ok {
		return true
	}
	if markdownIsHR(trimmed) || markdownIsBlockquote(trimmed) {
		return true
	}
	_, ok := markdownListItem(trimmed)
	return ok
}

type markdownHeadingRender struct {
	level int
	text  string
	line  int
}

type markdownLinkRender struct {
	text string
	url  string
}

func renderMarkdownTerminalLines(nodes []*object.Hash, width int) ([]string, []markdownHeadingRender, []markdownLinkRender) {
	var lines []string
	var headings []markdownHeadingRender
	var links []markdownLinkRender
	for _, node := range nodes {
		typ := markdownNodeString(node, "type")
		switch typ {
		case "heading":
			text, nodeLinks := markdownInlinePlainText(markdownNodeArray(node, "children"))
			links = append(links, nodeLinks...)
			level := int(markdownNodeNumber(node, "level"))
			prefix := strings.Repeat("#", level) + " "
			for _, line := range textWrapToWidth(prefix+text, width) {
				lines = append(lines, line)
			}
			headings = append(headings, markdownHeadingRender{level: level, text: text, line: int(markdownNodeNumber(node, "startLine"))})
		case "paragraph":
			text, nodeLinks := markdownInlinePlainText(markdownNodeArray(node, "children"))
			links = append(links, nodeLinks...)
			lines = append(lines, textWrapToWidth(text, width)...)
		case "list":
			for _, item := range markdownNodeArray(node, "children") {
				hash, ok := item.(*object.Hash)
				if !ok {
					continue
				}
				text, nodeLinks := markdownInlinePlainText(markdownNodeArray(hash, "children"))
				links = append(links, nodeLinks...)
				wrapped := textWrapToWidth(text, width-2)
				for i, line := range wrapped {
					if i == 0 {
						lines = append(lines, "- "+line)
					} else {
						lines = append(lines, "  "+line)
					}
				}
			}
		case "blockquote":
			children := markdownNodeHashChildren(node)
			childLines, childHeadings, childLinks := renderMarkdownTerminalLines(children, width-2)
			headings = append(headings, childHeadings...)
			links = append(links, childLinks...)
			for _, line := range childLines {
				lines = append(lines, "> "+line)
			}
		case "code":
			code := markdownNodeString(node, "text")
			for _, raw := range strings.Split(code, "\n") {
				lines = append(lines, textWrapToWidth("  "+raw, width)...)
			}
		case "hr":
			lines = append(lines, strings.Repeat("-", width))
		}
	}
	return lines, headings, links
}

func markdownInlinePlainText(nodes []object.Object) (string, []markdownLinkRender) {
	var out strings.Builder
	var links []markdownLinkRender
	for _, node := range nodes {
		hash, ok := node.(*object.Hash)
		if !ok {
			continue
		}
		typ := markdownNodeString(hash, "type")
		switch typ {
		case "text", "code":
			out.WriteString(markdownNodeString(hash, "text"))
		case "softbreak", "hardbreak":
			out.WriteString("\n")
		case "strong", "em":
			text, childLinks := markdownInlinePlainText(markdownNodeArray(hash, "children"))
			out.WriteString(text)
			links = append(links, childLinks...)
		case "link":
			text, childLinks := markdownInlinePlainText(markdownNodeArray(hash, "children"))
			url := markdownNodeString(hash, "url")
			out.WriteString(text)
			if url != "" {
				out.WriteString(" <")
				out.WriteString(url)
				out.WriteString(">")
				links = append(links, markdownLinkRender{text: text, url: url})
			}
			links = append(links, childLinks...)
		default:
			out.WriteString(hash.Inspect())
		}
	}
	return out.String(), links
}

func markdownNodeString(node *object.Hash, key string) string {
	if value, ok := hashValue(node, key); ok {
		if s, ok := value.(*object.String); ok {
			return s.Value
		}
	}
	return ""
}

func markdownNodeNumber(node *object.Hash, key string) float64 {
	if value, ok := hashValue(node, key); ok {
		if n, ok := value.(*object.Number); ok {
			return n.Value
		}
	}
	return 0
}

func markdownNodeArray(node *object.Hash, key string) []object.Object {
	if value, ok := hashValue(node, key); ok {
		if arr, ok := value.(*object.Array); ok {
			return arr.Elements
		}
	}
	return nil
}

func markdownNodeHashChildren(node *object.Hash) []*object.Hash {
	raw := markdownNodeArray(node, "children")
	children := make([]*object.Hash, 0, len(raw))
	for _, child := range raw {
		if hash, ok := child.(*object.Hash); ok {
			children = append(children, hash)
		}
	}
	return children
}

func markdownHeadingArray(headings []markdownHeadingRender) *object.Array {
	elements := make([]object.Object, len(headings))
	for i, heading := range headings {
		item := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(item, "level", &object.Number{Value: float64(heading.level)})
		setHashMember(item, "text", &object.String{Value: heading.text})
		setHashMember(item, "line", &object.Number{Value: float64(heading.line)})
		elements[i] = item
	}
	return &object.Array{Elements: elements}
}

func markdownLinkArray(links []markdownLinkRender) *object.Array {
	elements := make([]object.Object, len(links))
	for i, link := range links {
		item := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(item, "text", &object.String{Value: link.text})
		setHashMember(item, "url", &object.String{Value: link.url})
		elements[i] = item
	}
	return &object.Array{Elements: elements}
}

func markdownDebugNode(node *object.Hash) string {
	return fmt.Sprintf("%s:%g-%g", markdownNodeString(node, "type"), markdownNodeNumber(node, "startLine"), markdownNodeNumber(node, "endLine"))
}
