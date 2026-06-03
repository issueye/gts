package stdlib

import (
	"strings"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/time", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initTimeModule(exports)
		return exports, nil
	})
}

func initTimeModule(exports *object.Hash) {
	setHashMember(exports, "now", &object.Builtin{Name: "time.now", Fn: timeNow})
	setHashMember(exports, "nowMs", &object.Builtin{Name: "time.nowMs", Fn: timeNowMs})
	setHashMember(exports, "unix", &object.Builtin{Name: "time.unix", Fn: timeUnix})
	setHashMember(exports, "unixMs", &object.Builtin{Name: "time.unixMs", Fn: timeUnixMs})
	setHashMember(exports, "parse", &object.Builtin{Name: "time.parse", Fn: timeParse})
	setHashMember(exports, "format", &object.Builtin{Name: "time.format", Fn: timeFormat})
	setHashMember(exports, "add", &object.Builtin{Name: "time.add", Fn: timeAdd})
	setHashMember(exports, "since", &object.Builtin{Name: "time.since", Fn: timeSince})
	setHashMember(exports, "until", &object.Builtin{Name: "time.until", Fn: timeUntil})
	setHashMember(exports, "parseDuration", &object.Builtin{Name: "time.parseDuration", Fn: timeParseDuration})
	setHashMember(exports, "duration", &object.Builtin{Name: "time.duration", Fn: timeDuration})
	setHashMember(exports, "sleep", timerAlias("sleep"))

	setHashMember(exports, "RFC3339", &object.String{Value: time.RFC3339})
	setHashMember(exports, "RFC3339Nano", &object.String{Value: time.RFC3339Nano})
	setHashMember(exports, "RFC1123", &object.String{Value: time.RFC1123})
	setHashMember(exports, "RFC1123Z", &object.String{Value: time.RFC1123Z})
	setHashMember(exports, "UnixDate", &object.String{Value: time.UnixDate})
	setHashMember(exports, "DateTime", &object.String{Value: time.DateTime})
	setHashMember(exports, "DateOnly", &object.String{Value: time.DateOnly})
	setHashMember(exports, "TimeOnly", &object.String{Value: time.TimeOnly})
	setHashMember(exports, "Kitchen", &object.String{Value: time.Kitchen})
}

func timeNow(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.Date{Time: time.Now()}
}

func timeNowMs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.Number{Value: float64(time.Now().UnixMilli())}
}

func timeUnix(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	seconds, errObj := requiredNumber(pos, "time.unix", args, 0, "seconds")
	if errObj != nil {
		return errObj
	}
	nanos := float64(0)
	if len(args) >= 2 {
		n, ok := args[1].(*object.Number)
		if !ok {
			return object.NewError(pos, "time.unix: nanoseconds must be a number")
		}
		nanos = n.Value
	}
	return &object.Date{Time: time.Unix(int64(seconds), int64(nanos)).UTC()}
}

func timeUnixMs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	ms, errObj := requiredNumber(pos, "time.unixMs", args, 0, "milliseconds")
	if errObj != nil {
		return errObj
	}
	return &object.Date{Time: time.UnixMilli(int64(ms)).UTC()}
}

func timeParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "time.parse", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	layout := ""
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		s, ok := args[1].(*object.String)
		if !ok {
			return object.NewError(pos, "time.parse: layout must be a string")
		}
		layout = s.Value
	}
	t, err := parseStdTime(value, layout)
	if err != nil {
		return object.NewError(pos, "time.parse: %v", err)
	}
	return &object.Date{Time: t}
}

func timeFormat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	t, errObj := timeFromObject(pos, "time.format", args, 0)
	if errObj != nil {
		return errObj
	}
	layout := time.RFC3339
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		s, ok := args[1].(*object.String)
		if !ok {
			return object.NewError(pos, "time.format: layout must be a string")
		}
		layout = s.Value
	}
	loc, errObj := timeLocationFromArgs(pos, "time.format", args, 2)
	if errObj != nil {
		return errObj
	}
	if loc != nil {
		t = t.In(loc)
	}
	return &object.String{Value: t.Format(layout)}
}

func timeAdd(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	t, errObj := timeFromObject(pos, "time.add", args, 0)
	if errObj != nil {
		return errObj
	}
	d, errObj := durationFromObject(pos, "time.add", args, 1)
	if errObj != nil {
		return errObj
	}
	return &object.Date{Time: t.Add(d)}
}

func timeSince(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	t, errObj := timeFromObject(pos, "time.since", args, 0)
	if errObj != nil {
		return errObj
	}
	return &object.Number{Value: float64(time.Since(t).Milliseconds())}
}

func timeUntil(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	t, errObj := timeFromObject(pos, "time.until", args, 0)
	if errObj != nil {
		return errObj
	}
	return &object.Number{Value: float64(time.Until(t).Milliseconds())}
}

func timeParseDuration(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "time.parseDuration", args, 0, "duration")
	if errObj != nil {
		return errObj
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return object.NewError(pos, "time.parseDuration: %v", err)
	}
	return durationObject(d)
}

func timeDuration(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	d, errObj := durationFromObject(pos, "time.duration", args, 0)
	if errObj != nil {
		return errObj
	}
	return durationObject(d)
}

func parseStdTime(value, layout string) (time.Time, error) {
	if layout != "" {
		return time.Parse(layout, value)
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		time.UnixDate,
		time.DateTime,
		time.DateOnly,
		time.TimeOnly,
	}
	var last error
	for _, candidate := range layouts {
		t, err := time.Parse(candidate, value)
		if err == nil {
			return t, nil
		}
		last = err
	}
	return time.Time{}, last
}

func timeFromObject(pos ast.Position, name string, args []object.Object, index int) (time.Time, *object.Error) {
	if len(args) <= index {
		return time.Time{}, object.NewError(pos, "%s requires time", name)
	}
	switch v := args[index].(type) {
	case *object.Date:
		return v.Time, nil
	case *object.Number:
		return time.UnixMilli(int64(v.Value)).UTC(), nil
	case *object.String:
		t, err := parseStdTime(v.Value, "")
		if err != nil {
			return time.Time{}, object.NewError(pos, "%s: %v", name, err)
		}
		return t, nil
	default:
		return time.Time{}, object.NewError(pos, "%s: time must be a Date, number milliseconds, or string", name)
	}
}

func durationFromObject(pos ast.Position, name string, args []object.Object, index int) (time.Duration, *object.Error) {
	if len(args) <= index {
		return 0, object.NewError(pos, "%s requires duration", name)
	}
	switch v := args[index].(type) {
	case *object.Number:
		return time.Duration(v.Value) * time.Millisecond, nil
	case *object.String:
		d, err := time.ParseDuration(v.Value)
		if err != nil {
			return 0, object.NewError(pos, "%s: %v", name, err)
		}
		return d, nil
	default:
		return 0, object.NewError(pos, "%s: duration must be a number of milliseconds or Go duration string", name)
	}
}

func durationObject(d time.Duration) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "nanoseconds", &object.Number{Value: float64(d.Nanoseconds())})
	setHashMember(out, "microseconds", &object.Number{Value: float64(d.Microseconds())})
	setHashMember(out, "milliseconds", &object.Number{Value: float64(d.Milliseconds())})
	setHashMember(out, "seconds", &object.Number{Value: d.Seconds()})
	setHashMember(out, "string", &object.String{Value: d.String()})
	return out
}

func timeLocationFromArgs(pos ast.Position, name string, args []object.Object, index int) (*time.Location, *object.Error) {
	if len(args) <= index || args[index] == object.UNDEFINED || args[index] == object.NULL {
		return nil, nil
	}
	s, ok := args[index].(*object.String)
	if !ok {
		return nil, object.NewError(pos, "%s: timezone must be a string", name)
	}
	switch strings.ToUpper(s.Value) {
	case "", "LOCAL":
		return time.Local, nil
	case "UTC", "Z":
		return time.UTC, nil
	default:
		loc, err := time.LoadLocation(s.Value)
		if err != nil {
			return nil, object.NewError(pos, "%s: %v", name, err)
		}
		return loc, nil
	}
}

func requiredNumber(pos ast.Position, name string, args []object.Object, index int, label string) (float64, *object.Error) {
	if len(args) <= index {
		return 0, object.NewError(pos, "%s requires %s", name, label)
	}
	n, ok := args[index].(*object.Number)
	if !ok {
		return 0, object.NewError(pos, "%s: %s must be a number", name, label)
	}
	return n.Value, nil
}
