package evaluator

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

var consoleState = struct {
	sync.Mutex
	timers map[string]time.Time
	counts map[string]int
	indent int
}{
	timers: make(map[string]time.Time),
	counts: make(map[string]int),
}

func registerConsole(env *object.Environment) {
	env.VM().SetGlobalConst("console", consoleObject())
}

func consoleObject() object.Object {
	members := map[string]object.Object{
		"log":        &object.Builtin{Name: "console.log", Fn: builtinConsoleLog},
		"info":       &object.Builtin{Name: "console.info", Fn: builtinConsoleInfo},
		"warn":       &object.Builtin{Name: "console.warn", Fn: builtinConsoleWarn},
		"error":      &object.Builtin{Name: "console.error", Fn: builtinConsoleError},
		"debug":      &object.Builtin{Name: "console.debug", Fn: builtinConsoleDebug},
		"assert":     &object.Builtin{Name: "console.assert", Fn: builtinConsoleAssert},
		"time":       &object.Builtin{Name: "console.time", Fn: builtinConsoleTime},
		"timeEnd":    &object.Builtin{Name: "console.timeEnd", Fn: builtinConsoleTimeEnd},
		"trace":      &object.Builtin{Name: "console.trace", Fn: builtinConsoleTrace},
		"count":      &object.Builtin{Name: "console.count", Fn: builtinConsoleCount},
		"countReset": &object.Builtin{Name: "console.countReset", Fn: builtinConsoleCountReset},
		"group":      &object.Builtin{Name: "console.group", Fn: builtinConsoleGroup},
		"groupEnd":   &object.Builtin{Name: "console.groupEnd", Fn: builtinConsoleGroupEnd},
		"table":      &object.Builtin{Name: "console.table", Fn: builtinConsoleTable},
	}
	pairs := make(map[object.HashKey]object.HashPair, len(members))
	for name, value := range members {
		key := &object.String{Value: name}
		pairs[hashKey(key)] = object.HashPair{Key: key, Value: value}
	}
	return &object.Hash{Pairs: pairs}
}

func builtinConsoleLog(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	consoleWrite(os.Stdout, "", args...)
	return object.UNDEFINED
}

func builtinConsoleInfo(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	consoleWrite(os.Stdout, "[INFO] ", args...)
	return object.UNDEFINED
}

func builtinConsoleWarn(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	consoleWrite(os.Stderr, "[WARN] ", args...)
	return object.UNDEFINED
}

func builtinConsoleError(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	consoleWrite(os.Stderr, "[ERROR] ", args...)
	return object.UNDEFINED
}

func builtinConsoleDebug(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if os.Getenv("GTS_CONSOLE_DEBUG") != "" {
		consoleWrite(os.Stdout, "[DEBUG] ", args...)
	}
	return object.UNDEFINED
}

func builtinConsoleAssert(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 || object.IsTruthy(args[0]) {
		return object.UNDEFINED
	}
	message := []object.Object{&object.String{Value: "Assertion failed"}}
	if len(args) > 1 {
		message = append(message, args[1:]...)
	}
	consoleWrite(os.Stderr, "[ASSERT] ", message...)
	return object.UNDEFINED
}

func builtinConsoleTime(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	label := consoleLabel(args, "default")
	consoleState.Lock()
	consoleState.timers[label] = time.Now()
	consoleState.Unlock()
	return object.UNDEFINED
}

func builtinConsoleTimeEnd(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	label := consoleLabel(args, "default")
	consoleState.Lock()
	start, ok := consoleState.timers[label]
	if ok {
		delete(consoleState.timers, label)
	}
	consoleState.Unlock()
	if !ok {
		consoleWrite(os.Stderr, "[WARN] ", &object.String{Value: "No such label: " + label})
		return object.UNDEFINED
	}
	consoleWrite(os.Stdout, "", &object.String{Value: fmt.Sprintf("%s: %.3fms", label, float64(time.Since(start).Microseconds())/1000)})
	return object.UNDEFINED
}

func builtinConsoleTrace(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	values := append([]object.Object{}, args...)
	if len(values) == 0 {
		values = append(values, &object.String{Value: "Trace"})
	}
	consoleWrite(os.Stderr, "", values...)
	if pos.IsZero() {
		fmt.Fprintln(os.Stderr, consoleIndent()+"    at <source>")
	} else {
		fmt.Fprintln(os.Stderr, consoleIndent()+"    at "+pos.String())
	}
	return object.UNDEFINED
}

func builtinConsoleCount(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	label := consoleLabel(args, "default")
	consoleState.Lock()
	consoleState.counts[label]++
	count := consoleState.counts[label]
	consoleState.Unlock()
	consoleWrite(os.Stdout, "", &object.String{Value: fmt.Sprintf("%s: %d", label, count)})
	return object.UNDEFINED
}

func builtinConsoleCountReset(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	label := consoleLabel(args, "default")
	consoleState.Lock()
	delete(consoleState.counts, label)
	consoleState.Unlock()
	return object.UNDEFINED
}

func builtinConsoleGroup(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) > 0 {
		consoleWrite(os.Stdout, "", args...)
	}
	consoleState.Lock()
	consoleState.indent++
	consoleState.Unlock()
	return object.UNDEFINED
}

func builtinConsoleGroupEnd(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	consoleState.Lock()
	if consoleState.indent > 0 {
		consoleState.indent--
	}
	consoleState.Unlock()
	return object.UNDEFINED
}

func builtinConsoleTable(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		consoleWrite(os.Stdout, "", &object.String{Value: "undefined"})
		return object.UNDEFINED
	}
	for _, line := range consoleTableLines(args[0]) {
		fmt.Fprintln(os.Stdout, consoleIndent()+line)
	}
	return object.UNDEFINED
}

func consoleWrite(w io.Writer, prefix string, args ...object.Object) {
	fmt.Fprintln(w, consoleIndent()+prefix+consoleJoin(args))
}

func consoleJoin(args []object.Object) string {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = arg.Inspect()
	}
	return strings.Join(parts, " ")
}

func consoleLabel(args []object.Object, fallback string) string {
	if len(args) == 0 {
		return fallback
	}
	return args[0].Inspect()
}

func consoleIndent() string {
	consoleState.Lock()
	indent := consoleState.indent
	consoleState.Unlock()
	return strings.Repeat("  ", indent)
}

func consoleTableLines(obj object.Object) []string {
	switch v := obj.(type) {
	case *object.Array:
		rows := make([]map[string]string, len(v.Elements))
		keys := []string{"(index)"}
		seen := map[string]bool{"(index)": true}
		for i, elem := range v.Elements {
			row := consoleRow(elem)
			row["(index)"] = fmt.Sprintf("%d", i)
			rows[i] = row
			for key := range row {
				if !seen[key] {
					seen[key] = true
					keys = append(keys, key)
				}
			}
		}
		sort.Strings(keys[1:])
		return renderConsoleRows(keys, rows)
	case *object.Hash:
		keys := []string{"(key)", "value"}
		rows := make([]map[string]string, 0, len(v.Pairs))
		hashKeys := sortedHashPairs(v)
		for _, pair := range hashKeys {
			rows = append(rows, map[string]string{
				"(key)": pair.Key.Inspect(),
				"value": pair.Value.Inspect(),
			})
		}
		return renderConsoleRows(keys, rows)
	default:
		return []string{obj.Inspect()}
	}
}

func consoleRow(obj object.Object) map[string]string {
	switch v := obj.(type) {
	case *object.Hash:
		row := make(map[string]string, len(v.Pairs))
		for _, pair := range v.OrderedPairs() {
			row[pair.Key.Inspect()] = pair.Value.Inspect()
		}
		return row
	default:
		return map[string]string{"value": obj.Inspect()}
	}
}

func renderConsoleRows(keys []string, rows []map[string]string) []string {
	widths := make(map[string]int, len(keys))
	for _, key := range keys {
		widths[key] = len(key)
	}
	for _, row := range rows {
		for _, key := range keys {
			if len(row[key]) > widths[key] {
				widths[key] = len(row[key])
			}
		}
	}
	lines := []string{renderConsoleLine(keys, widths, nil)}
	for _, row := range rows {
		lines = append(lines, renderConsoleLine(keys, widths, row))
	}
	return lines
}

func renderConsoleLine(keys []string, widths map[string]int, row map[string]string) string {
	cells := make([]string, len(keys))
	for i, key := range keys {
		value := key
		if row != nil {
			value = row[key]
		}
		cells[i] = value + strings.Repeat(" ", widths[key]-len(value))
	}
	return strings.Join(cells, " | ")
}

func sortedHashPairs(hash *object.Hash) []object.HashPair {
	pairs := make([]object.HashPair, 0, len(hash.Pairs))
	for _, pair := range hash.OrderedPairs() {
		pairs = append(pairs, pair)
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Key.Inspect() < pairs[j].Key.Inspect()
	})
	return pairs
}
