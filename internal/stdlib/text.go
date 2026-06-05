package stdlib

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/text", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initTextModule(exports)
		return exports, nil
	})
}

func initTextModule(exports *object.Hash) {
	setHashMember(exports, "chars", &object.Builtin{Name: "text.chars", Fn: textChars})
	setHashMember(exports, "runes", &object.Builtin{Name: "text.runes", Fn: textChars})
	setHashMember(exports, "width", &object.Builtin{Name: "text.width", Fn: textWidth})
	setHashMember(exports, "truncateWidth", &object.Builtin{Name: "text.truncateWidth", Fn: textTruncateWidth})
	setHashMember(exports, "padRightWidth", &object.Builtin{Name: "text.padRightWidth", Fn: textPadRightWidth})
	setHashMember(exports, "wrapWidth", &object.Builtin{Name: "text.wrapWidth", Fn: textWrapWidth})
	setHashMember(exports, "stripAnsi", &object.Builtin{Name: "text.stripAnsi", Fn: textStripAnsi})
}

func textChars(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "text.chars", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	chars := textVisibleChars(value)
	elements := make([]object.Object, len(chars))
	for i, ch := range chars {
		elements[i] = &object.String{Value: ch}
	}
	return &object.Array{Elements: elements}
}

func textWidth(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "text.width", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	return &object.Number{Value: float64(textVisibleWidth(value))}
}

func textTruncateWidth(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "text.truncateWidth", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	width, errObj := requiredNumber(pos, "text.truncateWidth", args, 1, "width")
	if errObj != nil {
		return errObj
	}
	if width < 0 {
		width = 0
	}
	return &object.String{Value: textTruncateToWidth(value, int(width))}
}

func textPadRightWidth(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "text.padRightWidth", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	width, errObj := requiredNumber(pos, "text.padRightWidth", args, 1, "width")
	if errObj != nil {
		return errObj
	}
	out := value
	for textVisibleWidth(out) < int(width) {
		out += " "
	}
	return &object.String{Value: out}
}

func textWrapWidth(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "text.wrapWidth", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	width, errObj := requiredNumber(pos, "text.wrapWidth", args, 1, "width")
	if errObj != nil {
		return errObj
	}
	lines := textWrapToWidth(value, int(width))
	elements := make([]object.Object, len(lines))
	for i, line := range lines {
		elements[i] = &object.String{Value: line}
	}
	return &object.Array{Elements: elements}
}

func textStripAnsi(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "text.stripAnsi", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: stripANSI(value)}
}

func textVisibleChars(value string) []string {
	value = stripANSI(value)
	chars := make([]string, 0, utf8.RuneCountInString(value))
	var pending strings.Builder
	for _, r := range value {
		if isCombiningRune(r) {
			pending.WriteRune(r)
			continue
		}
		if pending.Len() > 0 {
			chars = append(chars, pending.String())
			pending.Reset()
		}
		pending.WriteRune(r)
	}
	if pending.Len() > 0 {
		chars = append(chars, pending.String())
	}
	return chars
}

func textVisibleWidth(value string) int {
	width := 0
	for _, ch := range textVisibleChars(value) {
		width += textCharWidth(ch)
	}
	return width
}

func textTruncateToWidth(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	var out strings.Builder
	width := 0
	for _, ch := range textVisibleChars(value) {
		chWidth := textCharWidth(ch)
		if width+chWidth > limit {
			break
		}
		out.WriteString(ch)
		width += chWidth
	}
	return out.String()
}

func textWrapToWidth(value string, limit int) []string {
	if limit <= 0 {
		return []string{""}
	}
	rawLines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, raw := range rawLines {
		chars := textVisibleChars(raw)
		if len(chars) == 0 {
			lines = append(lines, "")
			continue
		}
		var out strings.Builder
		width := 0
		for _, ch := range chars {
			chWidth := textCharWidth(ch)
			if width > 0 && width+chWidth > limit {
				lines = append(lines, out.String())
				out.Reset()
				width = 0
			}
			out.WriteString(ch)
			width += chWidth
		}
		lines = append(lines, out.String())
	}
	return lines
}

func textCharWidth(ch string) int {
	r, _ := utf8.DecodeRuneInString(ch)
	if r == utf8.RuneError || r == 0 || r == '\n' || r == '\r' || r == '\t' || isCombiningRune(r) {
		return 0
	}
	if isWideRune(r) {
		return 2
	}
	return 1
}

func isCombiningRune(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r)
}

func isWideRune(r rune) bool {
	return (r >= 0x1100 && r <= 0x115f) ||
		(r >= 0x2329 && r <= 0x232a) ||
		(r >= 0x2e80 && r <= 0xa4cf) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1faff) ||
		(r >= 0x20000 && r <= 0x3fffd)
}

func stripANSI(value string) string {
	var out strings.Builder
	for i := 0; i < len(value); {
		if value[i] == 0x1b && i+1 < len(value) {
			next := value[i+1]
			if next == '[' {
				i += 2
				for i < len(value) {
					b := value[i]
					i++
					if b >= 0x40 && b <= 0x7e {
						break
					}
				}
				continue
			}
			if next == ']' {
				i += 2
				for i < len(value) {
					if value[i] == 0x07 {
						i++
						break
					}
					if value[i] == 0x1b && i+1 < len(value) && value[i+1] == '\\' {
						i += 2
						break
					}
					i++
				}
				continue
			}
		}
		r, size := utf8.DecodeRuneInString(value[i:])
		out.WriteRune(r)
		i += size
	}
	return out.String()
}
