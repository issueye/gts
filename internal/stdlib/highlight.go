package stdlib

import (
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/highlight", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initHighlightModule(exports)
		return exports, nil
	})
}

func initHighlightModule(exports *object.Hash) {
	setHashMember(exports, "terminal", &object.Builtin{Name: "highlight.terminal", Fn: highlightTerminal})
}

func highlightTerminal(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	code, errObj := requiredString(pos, "highlight.terminal", args, 0, "code")
	if errObj != nil {
		return errObj
	}
	opts := highlightTerminalOptions{lang: "", width: 80, color: true}
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		hash, ok := args[1].(*object.Hash)
		if !ok {
			return object.NewError(pos, "highlight.terminal: options must be an object")
		}
		if value, ok := hashValue(hash, "lang"); ok && value != object.UNDEFINED && value != object.NULL {
			lang, ok := value.(*object.String)
			if !ok {
				return object.NewError(pos, "highlight.terminal: lang must be a string")
			}
			opts.lang = strings.ToLower(lang.Value)
		}
		if value, ok := hashValue(hash, "width"); ok && value != object.UNDEFINED && value != object.NULL {
			width, ok := value.(*object.Number)
			if !ok {
				return object.NewError(pos, "highlight.terminal: width must be a number")
			}
			opts.width = int(width.Value)
		}
		if value, ok := hashValue(hash, "color"); ok && value != object.UNDEFINED && value != object.NULL {
			color, ok := value.(*object.Boolean)
			if !ok {
				return object.NewError(pos, "highlight.terminal: color must be a boolean")
			}
			opts.color = color.Value
		}
	}
	if opts.width < 1 {
		opts.width = 80
	}
	lines := highlightLines(code, opts)
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "lines", strSliceToArray(lines))
	setHashMember(out, "text", &object.String{Value: strings.Join(lines, "\n")})
	setHashMember(out, "lang", &object.String{Value: opts.lang})
	return out
}

type highlightTerminalOptions struct {
	lang  string
	width int
	color bool
}

func highlightLines(code string, opts highlightTerminalOptions) []string {
	rawLines := strings.Split(strings.ReplaceAll(code, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		for _, wrapped := range textWrapToWidth(line, opts.width) {
			out = append(out, highlightLine(wrapped, opts))
		}
	}
	return out
}

func highlightLine(line string, opts highlightTerminalOptions) string {
	if !opts.color {
		return line
	}
	switch opts.lang {
	case "diff":
		if strings.HasPrefix(line, "+") {
			return terminalStyleString(line, terminalStyleOptions{fg: "success", color: true})
		}
		if strings.HasPrefix(line, "-") {
			return terminalStyleString(line, terminalStyleOptions{fg: "error", color: true})
		}
		if strings.HasPrefix(line, "@@") {
			return terminalStyleString(line, terminalStyleOptions{fg: "accent", color: true, bold: true})
		}
	case "json":
		return highlightJSONLine(line)
	case "shell", "sh", "bash", "gs", "js", "toml":
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			return terminalStyleString(line, terminalStyleOptions{fg: "muted", color: true})
		}
	}
	return line
}

func highlightJSONLine(line string) string {
	var out strings.Builder
	inString := false
	escaped := false
	var buf strings.Builder
	for _, r := range line {
		if inString {
			buf.WriteRune(r)
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				text := buf.String()
				out.WriteString(terminalStyleString(text, terminalStyleOptions{fg: "success", color: true}))
				buf.Reset()
				inString = false
			}
			continue
		}
		if r == '"' {
			inString = true
			buf.WriteRune(r)
			continue
		}
		out.WriteRune(r)
	}
	if buf.Len() > 0 {
		out.WriteString(buf.String())
	}
	return out.String()
}
