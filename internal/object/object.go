package object

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/ast"
)

type ObjectType string

const (
	NUMBER_OBJ         ObjectType = "NUMBER"
	STRING_OBJ         ObjectType = "STRING"
	BOOLEAN_OBJ        ObjectType = "BOOLEAN"
	NULL_OBJ           ObjectType = "NULL"
	UNDEFINED_OBJ      ObjectType = "UNDEFINED"
	ARRAY_OBJ          ObjectType = "ARRAY"
	OBJECT_OBJ         ObjectType = "OBJECT"
	FUNCTION_OBJ       ObjectType = "FUNCTION"
	BUILTIN_OBJ        ObjectType = "BUILTIN"
	ERROR_OBJ          ObjectType = "ERROR"
	RETURN_OBJ         ObjectType = "RETURN"
	CLASS_OBJ          ObjectType = "CLASS"
	INSTANCE_OBJ       ObjectType = "INSTANCE"
	GOOBJECT_OBJ       ObjectType = "GOOBJECT"
	TIMER_OBJ          ObjectType = "TIMER"
	DATE_OBJ           ObjectType = "DATE"
	REGEXP_OBJ         ObjectType = "REGEXP"
	MAP_OBJ            ObjectType = "MAP"
	SET_OBJ            ObjectType = "SET"
	BOOLEAN_OBJECT_OBJ ObjectType = "BOOLEAN_OBJECT"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

// --- Primitives ---

type Number struct{ Value float64 }

func (n *Number) Type() ObjectType { return NUMBER_OBJ }
func (n *Number) Inspect() string {
	if float64(int64(n.Value)) == n.Value {
		return fmt.Sprintf("%.0f", n.Value)
	}
	return fmt.Sprintf("%v", n.Value)
}

type String struct{ Value string }

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type Boolean struct{ Value bool }

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string {
	if b.Value {
		return "true"
	}
	return "false"
}

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

type Undefined struct{}

func (u *Undefined) Type() ObjectType { return UNDEFINED_OBJ }
func (u *Undefined) Inspect() string  { return "undefined" }

// --- Array ---

type Array struct {
	Elements []Object
	Pos      ast.Position
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	var out bytes.Buffer
	elems := make([]string, len(a.Elements))
	for i, e := range a.Elements {
		elems[i] = e.Inspect()
	}
	out.WriteString("[")
	out.WriteString(strings.Join(elems, ", "))
	out.WriteString("]")
	return out.String()
}

// --- Object (Hash) ---

type HashPair struct {
	Key   Object
	Value Object
}

type Hash struct {
	Pairs  map[HashKey]HashPair
	Proto  *Hash
	Frozen bool
	Sealed bool
	Pos    ast.Position
}

func (h *Hash) Type() ObjectType { return OBJECT_OBJ }
func (h *Hash) Inspect() string {
	var out bytes.Buffer
	pairs := make([]string, 0, len(h.Pairs))
	for _, p := range h.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s", p.Key.Inspect(), p.Value.Inspect()))
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

// HashKey is used for map lookups.
type HashKey struct {
	Type  ObjectType
	Value string
}

// --- Map / Set ---

type Map struct {
	Entries map[HashKey]HashPair
	Pos     ast.Position
}

func (m *Map) Type() ObjectType { return MAP_OBJ }
func (m *Map) Inspect() string {
	var out bytes.Buffer
	pairs := make([]string, 0, len(m.Entries))
	for _, p := range m.Entries {
		pairs = append(pairs, fmt.Sprintf("%s => %s", p.Key.Inspect(), p.Value.Inspect()))
	}
	out.WriteString("Map{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

type Set struct {
	Values map[HashKey]Object
	Pos    ast.Position
}

func (s *Set) Type() ObjectType { return SET_OBJ }
func (s *Set) Inspect() string {
	var out bytes.Buffer
	values := make([]string, 0, len(s.Values))
	for _, v := range s.Values {
		values = append(values, v.Inspect())
	}
	out.WriteString("Set{")
	out.WriteString(strings.Join(values, ", "))
	out.WriteString("}")
	return out.String()
}

// --- Function ---

type Function struct {
	Name       string
	Parameters []*ast.Param
	Body       *ast.BlockStmt
	Env        *Environment
	IsAsync    bool
	ReturnT    *ast.TypeAnnotation
	Pos        ast.Position
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	params := make([]string, len(f.Parameters))
	for i, p := range f.Parameters {
		params[i] = p.Name
	}
	if f.Name != "" {
		return fmt.Sprintf("fn %s(%s)", f.Name, strings.Join(params, ", "))
	}
	return fmt.Sprintf("fn(%s)", strings.Join(params, ", "))
}

// --- Builtin ---

// BuiltinFunc is the signature for built-in function implementations.
type BuiltinFunc func(env *Environment, pos ast.Position, args ...Object) Object

// --- Builtin ---

type Builtin struct {
	Name  string
	Fn    BuiltinFunc
	Extra Object // context for array/string method binding
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "<builtin " + b.Name + ">" }

// --- Control Flow ---

type ReturnValue struct{ Value Object }

func (r *ReturnValue) Type() ObjectType { return RETURN_OBJ }
func (r *ReturnValue) Inspect() string  { return r.Value.Inspect() }

type Error struct {
	Message string
	Name    string
	Stack   string
	Runtime bool
	Pos     ast.Position
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string {
	name := e.Name
	if name == "" {
		name = "Error"
	}
	if e.Pos.IsZero() {
		return fmt.Sprintf("%s: %s", name, e.Message)
	}
	return fmt.Sprintf("%s: %s: %s", e.Pos, name, e.Message)
}

// --- Class / Instance ---

type Class struct {
	Name    string
	Super   *Class
	Methods map[string]*Function
	Fields  map[string]Object
	Statics map[string]Object
	Pos     ast.Position
}

func (c *Class) Type() ObjectType { return CLASS_OBJ }
func (c *Class) Inspect() string  { return "<class " + c.Name + ">" }

type Instance struct {
	Class *Class
	Props map[string]Object
	Pos   ast.Position
}

func (i *Instance) Type() ObjectType { return INSTANCE_OBJ }
func (i *Instance) Inspect() string {
	return fmt.Sprintf("<%s instance>", i.Class.Name)
}

// GoObject wraps an arbitrary Go value for use within the scripting runtime.
type GoObject struct {
	Value interface{}
}

func (g *GoObject) Type() ObjectType { return GOOBJECT_OBJ }
func (g *GoObject) Inspect() string  { return fmt.Sprintf("<go %T>", g.Value) }

// TimerId is an opaque handle returned by setTimeout/setInterval.
type TimerId struct {
	ID     int64
	Cancel func()
}

func (t *TimerId) Type() ObjectType { return TIMER_OBJ }
func (t *TimerId) Inspect() string  { return fmt.Sprintf("<timer %d>", t.ID) }

// Date wraps a point in time for the runtime Date builtin.
type Date struct {
	Time time.Time
}

func (d *Date) Type() ObjectType { return DATE_OBJ }
func (d *Date) Inspect() string  { return d.Time.Local().Format(time.RFC1123) }

// RegExp wraps a compiled Go regexp plus the user-facing source and flags.
type RegExp struct {
	Source string
	Flags  string
	Re     *regexp.Regexp
}

func (r *RegExp) Type() ObjectType { return REGEXP_OBJ }
func (r *RegExp) Inspect() string  { return "/" + r.Source + "/" + r.Flags }

// BooleanObject is the boxed value returned by new Boolean(...).
type BooleanObject struct {
	Value bool
}

func (b *BooleanObject) Type() ObjectType { return BOOLEAN_OBJECT_OBJ }
func (b *BooleanObject) Inspect() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// --- Helpers ---

func NewError(pos ast.Position, format string, args ...interface{}) *Error {
	name := "Error"
	message := fmt.Sprintf(format, args...)
	for _, prefix := range []string{"TypeError", "RangeError", "ReferenceError", "SyntaxError", "ImportError", "ExportError", "MatchError"} {
		if strings.HasPrefix(message, prefix+": ") {
			name = prefix
			message = strings.TrimPrefix(message, prefix+": ")
			break
		}
	}
	err := NewNamedError(pos, name, message)
	err.Runtime = true
	return err
}

func NewNamedError(pos ast.Position, name, message string) *Error {
	if name == "" {
		name = "Error"
	}
	err := &Error{Name: name, Message: message, Pos: pos}
	err.Stack = err.FormatStack()
	return err
}

func (e *Error) FormatStack() string {
	name := e.Name
	if name == "" {
		name = "Error"
	}
	first := fmt.Sprintf("%s: %s", name, e.Message)
	if e.Pos.IsZero() {
		return first
	}
	return fmt.Sprintf("%s\n    at %s", first, e.Pos)
}

var (
	NULL      = &Null{}
	UNDEFINED = &Undefined{}
	TRUE      = &Boolean{Value: true}
	FALSE     = &Boolean{Value: false}
)

func NativeBool(v bool) *Boolean {
	if v {
		return TRUE
	}
	return FALSE
}

func IsNumber(o Object) bool { return o != nil && o.Type() == NUMBER_OBJ }
func IsString(o Object) bool { return o != nil && o.Type() == STRING_OBJ }
func IsError(o Object) bool  { return o != nil && o.Type() == ERROR_OBJ }
func IsRuntimeError(o Object) bool {
	err, ok := o.(*Error)
	return ok && err.Runtime
}
func IsTruthy(o Object) bool {
	switch o := o.(type) {
	case *Null, *Undefined:
		return false
	case *Boolean:
		return o.Value
	case *Number:
		return o.Value != 0
	case *String:
		return len(o.Value) > 0
	default:
		return true
	}
}
