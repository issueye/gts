package evaluator

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

var dateMethods = map[string]object.BuiltinFunc{
	"getTime":            builtinDateGetTime,
	"valueOf":            builtinDateGetTime,
	"toISOString":        builtinDateToISOString,
	"toString":           builtinDateToString,
	"toLocaleString":     builtinDateToLocaleString,
	"toLocaleDateString": builtinDateToLocaleDateString,
	"toLocaleTimeString": builtinDateToLocaleTimeString,
	"getFullYear":        builtinDateGetFullYear,
	"getMonth":           builtinDateGetMonth,
	"getDate":            builtinDateGetDate,
	"getDay":             builtinDateGetDay,
	"getHours":           builtinDateGetHours,
	"getMinutes":         builtinDateGetMinutes,
	"getSeconds":         builtinDateGetSeconds,
	"getMilliseconds":    builtinDateGetMilliseconds,
	"getUTCFullYear":     builtinDateGetUTCFullYear,
	"getUTCMonth":        builtinDateGetUTCMonth,
	"getUTCDate":         builtinDateGetUTCDate,
	"getUTCDay":          builtinDateGetUTCDay,
	"getUTCHours":        builtinDateGetUTCHours,
	"getUTCMinutes":      builtinDateGetUTCMinutes,
	"getUTCSeconds":      builtinDateGetUTCSeconds,
	"getUTCMilliseconds": builtinDateGetUTCMilliseconds,
	"setTime":            builtinDateSetTime,
	"setFullYear":        builtinDateSetFullYear,
	"setMonth":           builtinDateSetMonth,
	"setDate":            builtinDateSetDate,
	"setHours":           builtinDateSetHours,
	"setMinutes":         builtinDateSetMinutes,
	"setSeconds":         builtinDateSetSeconds,
	"setMilliseconds":    builtinDateSetMilliseconds,
	"setUTCFullYear":     builtinDateSetUTCFullYear,
	"setUTCMonth":        builtinDateSetUTCMonth,
	"setUTCDate":         builtinDateSetUTCDate,
	"setUTCHours":        builtinDateSetUTCHours,
	"setUTCMinutes":      builtinDateSetUTCMinutes,
	"setUTCSeconds":      builtinDateSetUTCSeconds,
	"setUTCMilliseconds": builtinDateSetUTCMilliseconds,
}

var regexpMethods = map[string]object.BuiltinFunc{
	"test":     builtinRegExpTest,
	"exec":     builtinRegExpExec,
	"toString": builtinRegExpToString,
}

func builtinDate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	d, err := makeDate(pos, args...)
	if err != nil {
		return err
	}
	env.ObjectManager().Register(d)
	return d
}

func builtinDateNow(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.Number{Value: dateMillis(time.Now())}
}

func builtinDateParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return &object.Number{Value: math.NaN()}
	}
	t, err := parseDateString(args[0].Inspect())
	if err != nil {
		return &object.Number{Value: math.NaN()}
	}
	return &object.Number{Value: dateMillis(t)}
}

func builtinDateUTC(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	parts := []int{0, 0, 1, 0, 0, 0, 0}
	for i := 0; i < len(args) && i < len(parts); i++ {
		n, ok := args[i].(*object.Number)
		if !ok {
			return object.NewError(pos, "TypeError: Date.UTC component must be a number")
		}
		parts[i] = int(n.Value)
	}
	if parts[0] >= 0 && parts[0] <= 99 {
		parts[0] += 1900
	}
	t := time.Date(parts[0], time.Month(parts[1]+1), parts[2], parts[3], parts[4], parts[5], parts[6]*int(time.Millisecond), time.UTC)
	return &object.Number{Value: dateMillis(t)}
}

func makeDate(pos ast.Position, args ...object.Object) (*object.Date, *object.Error) {
	if len(args) == 0 {
		return &object.Date{Time: time.Now()}, nil
	}
	if len(args) == 1 {
		switch a := args[0].(type) {
		case *object.Date:
			return &object.Date{Time: a.Time}, nil
		case *object.Number:
			return &object.Date{Time: time.UnixMilli(int64(a.Value)).UTC()}, nil
		case *object.String:
			t, err := parseDateString(a.Value)
			if err != nil {
				return nil, object.NewError(pos, "RangeError: invalid date")
			}
			return &object.Date{Time: t}, nil
		default:
			return nil, object.NewError(pos, "TypeError: Date requires number or string argument")
		}
	}
	parts := []int{0, 0, 1, 0, 0, 0, 0}
	for i := 0; i < len(args) && i < len(parts); i++ {
		n, ok := args[i].(*object.Number)
		if !ok {
			return nil, object.NewError(pos, "TypeError: Date component must be a number")
		}
		parts[i] = int(n.Value)
	}
	if parts[0] >= 0 && parts[0] <= 99 {
		parts[0] += 1900
	}
	return &object.Date{Time: time.Date(parts[0], time.Month(parts[1]+1), parts[2], parts[3], parts[4], parts[5], parts[6]*int(time.Millisecond), time.Local)}, nil
}

func parseDateString(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
		"2006/01/02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date")
}

func dateReceiver(env *object.Environment, pos ast.Position, name string) (*object.Date, *object.Error) {
	d, ok := env.Extra.(*object.Date)
	if !ok {
		return nil, object.NewError(pos, "TypeError: %s requires Date receiver", name)
	}
	return d, nil
}

func dateMillis(t time.Time) float64 {
	return float64(t.UnixMilli())
}

func dateNumber(args []object.Object, idx int, fallback int) int {
	if idx >= len(args) {
		return fallback
	}
	if n, ok := args[idx].(*object.Number); ok {
		return int(n.Value)
	}
	return fallback
}

func builtinDateGetTime(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	d, err := dateReceiver(env, pos, "Date.getTime")
	if err != nil {
		return err
	}
	return &object.Number{Value: dateMillis(d.Time)}
}

func builtinDateToISOString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	d, err := dateReceiver(env, pos, "Date.toISOString")
	if err != nil {
		return err
	}
	return &object.String{Value: d.Time.UTC().Format("2006-01-02T15:04:05.000Z")}
}

func builtinDateToString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	d, err := dateReceiver(env, pos, "Date.toString")
	if err != nil {
		return err
	}
	return &object.String{Value: d.Time.Local().Format(time.RFC1123)}
}

func builtinDateToLocaleString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	d, err := dateReceiver(env, pos, "Date.toLocaleString")
	if err != nil {
		return err
	}
	return &object.String{Value: d.Time.Local().Format("2006-01-02 15:04:05")}
}

func builtinDateToLocaleDateString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	d, err := dateReceiver(env, pos, "Date.toLocaleDateString")
	if err != nil {
		return err
	}
	return &object.String{Value: d.Time.Local().Format("2006-01-02")}
}

func builtinDateToLocaleTimeString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	d, err := dateReceiver(env, pos, "Date.toLocaleTimeString")
	if err != nil {
		return err
	}
	return &object.String{Value: d.Time.Local().Format("15:04:05")}
}

func datePart(env *object.Environment, pos ast.Position, utc bool, pick func(time.Time) float64) object.Object {
	d, err := dateReceiver(env, pos, "Date getter")
	if err != nil {
		return err
	}
	t := d.Time
	if utc {
		t = t.UTC()
	} else {
		t = t.Local()
	}
	return &object.Number{Value: pick(t)}
}

func builtinDateGetFullYear(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, false, func(t time.Time) float64 { return float64(t.Year()) })
}
func builtinDateGetMonth(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, false, func(t time.Time) float64 { return float64(t.Month() - 1) })
}
func builtinDateGetDate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, false, func(t time.Time) float64 { return float64(t.Day()) })
}
func builtinDateGetDay(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, false, func(t time.Time) float64 { return float64(t.Weekday()) })
}
func builtinDateGetHours(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, false, func(t time.Time) float64 { return float64(t.Hour()) })
}
func builtinDateGetMinutes(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, false, func(t time.Time) float64 { return float64(t.Minute()) })
}
func builtinDateGetSeconds(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, false, func(t time.Time) float64 { return float64(t.Second()) })
}
func builtinDateGetMilliseconds(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, false, func(t time.Time) float64 { return float64(t.Nanosecond() / int(time.Millisecond)) })
}
func builtinDateGetUTCFullYear(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, true, func(t time.Time) float64 { return float64(t.Year()) })
}
func builtinDateGetUTCMonth(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, true, func(t time.Time) float64 { return float64(t.Month() - 1) })
}
func builtinDateGetUTCDate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, true, func(t time.Time) float64 { return float64(t.Day()) })
}
func builtinDateGetUTCDay(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, true, func(t time.Time) float64 { return float64(t.Weekday()) })
}
func builtinDateGetUTCHours(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, true, func(t time.Time) float64 { return float64(t.Hour()) })
}
func builtinDateGetUTCMinutes(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, true, func(t time.Time) float64 { return float64(t.Minute()) })
}
func builtinDateGetUTCSeconds(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, true, func(t time.Time) float64 { return float64(t.Second()) })
}
func builtinDateGetUTCMilliseconds(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return datePart(env, pos, true, func(t time.Time) float64 { return float64(t.Nanosecond() / int(time.Millisecond)) })
}

func builtinDateSetTime(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	d, err := dateReceiver(env, pos, "Date.setTime")
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return &object.Number{Value: math.NaN()}
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "TypeError: Date.setTime requires a number")
	}
	d.Time = time.UnixMilli(int64(n.Value)).UTC()
	return &object.Number{Value: dateMillis(d.Time)}
}

func setDateInZone(env *object.Environment, pos ast.Position, args []object.Object, utc bool, update func(time.Time, []object.Object, *time.Location) time.Time, name string) object.Object {
	d, err := dateReceiver(env, pos, name)
	if err != nil {
		return err
	}
	loc := time.Local
	t := d.Time.Local()
	if utc {
		loc = time.UTC
		t = d.Time.UTC()
	}
	d.Time = update(t, args, loc)
	return &object.Number{Value: dateMillis(d.Time)}
}

func builtinDateSetFullYear(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, false, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(dateNumber(args, 0, t.Year()), time.Month(dateNumber(args, 1, int(t.Month())-1)+1), dateNumber(args, 2, t.Day()), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	}, "Date.setFullYear")
}
func builtinDateSetMonth(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, false, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), time.Month(dateNumber(args, 0, int(t.Month())-1)+1), dateNumber(args, 1, t.Day()), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	}, "Date.setMonth")
}
func builtinDateSetDate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, false, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), t.Month(), dateNumber(args, 0, t.Day()), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	}, "Date.setDate")
}
func builtinDateSetHours(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, false, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), dateNumber(args, 0, t.Hour()), dateNumber(args, 1, t.Minute()), dateNumber(args, 2, t.Second()), dateNumber(args, 3, t.Nanosecond()/int(time.Millisecond))*int(time.Millisecond), loc)
	}, "Date.setHours")
}
func builtinDateSetMinutes(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, false, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), dateNumber(args, 0, t.Minute()), dateNumber(args, 1, t.Second()), dateNumber(args, 2, t.Nanosecond()/int(time.Millisecond))*int(time.Millisecond), loc)
	}, "Date.setMinutes")
}
func builtinDateSetSeconds(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, false, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), dateNumber(args, 0, t.Second()), dateNumber(args, 1, t.Nanosecond()/int(time.Millisecond))*int(time.Millisecond), loc)
	}, "Date.setSeconds")
}
func builtinDateSetMilliseconds(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, false, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), dateNumber(args, 0, t.Nanosecond()/int(time.Millisecond))*int(time.Millisecond), loc)
	}, "Date.setMilliseconds")
}

func builtinDateSetUTCFullYear(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, true, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(dateNumber(args, 0, t.Year()), time.Month(dateNumber(args, 1, int(t.Month())-1)+1), dateNumber(args, 2, t.Day()), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	}, "Date.setUTCFullYear")
}
func builtinDateSetUTCMonth(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, true, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), time.Month(dateNumber(args, 0, int(t.Month())-1)+1), dateNumber(args, 1, t.Day()), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	}, "Date.setUTCMonth")
}
func builtinDateSetUTCDate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, true, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), t.Month(), dateNumber(args, 0, t.Day()), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	}, "Date.setUTCDate")
}
func builtinDateSetUTCHours(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, true, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), dateNumber(args, 0, t.Hour()), dateNumber(args, 1, t.Minute()), dateNumber(args, 2, t.Second()), dateNumber(args, 3, t.Nanosecond()/int(time.Millisecond))*int(time.Millisecond), loc)
	}, "Date.setUTCHours")
}
func builtinDateSetUTCMinutes(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, true, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), dateNumber(args, 0, t.Minute()), dateNumber(args, 1, t.Second()), dateNumber(args, 2, t.Nanosecond()/int(time.Millisecond))*int(time.Millisecond), loc)
	}, "Date.setUTCMinutes")
}
func builtinDateSetUTCSeconds(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, true, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), dateNumber(args, 0, t.Second()), dateNumber(args, 1, t.Nanosecond()/int(time.Millisecond))*int(time.Millisecond), loc)
	}, "Date.setUTCSeconds")
}
func builtinDateSetUTCMilliseconds(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return setDateInZone(env, pos, args, true, func(t time.Time, args []object.Object, loc *time.Location) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), dateNumber(args, 0, t.Nanosecond()/int(time.Millisecond))*int(time.Millisecond), loc)
	}, "Date.setUTCMilliseconds")
}

func builtinRegExp(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	re, err := makeRegExp(pos, args...)
	if err != nil {
		return err
	}
	env.ObjectManager().Register(re)
	return re
}

func makeRegExp(pos ast.Position, args ...object.Object) (*object.RegExp, *object.Error) {
	if len(args) > 0 {
		if existing, ok := args[0].(*object.RegExp); ok {
			if len(args) == 1 {
				return existing, nil
			}
			args[0] = &object.String{Value: existing.Source}
		}
	}
	source := ""
	flags := ""
	if len(args) > 0 {
		source = args[0].Inspect()
	}
	if len(args) > 1 {
		flags = args[1].Inspect()
	}
	return compileRegExp(pos, source, flags)
}

func compileRegExp(pos ast.Position, source, flags string) (*object.RegExp, *object.Error) {
	prefix := ""
	seen := map[rune]bool{}
	for _, f := range flags {
		if seen[f] {
			return nil, object.NewError(pos, "SyntaxError: duplicate regexp flag %c", f)
		}
		seen[f] = true
		switch f {
		case 'i':
			prefix += "(?i)"
		case 'g':
		default:
			return nil, object.NewError(pos, "SyntaxError: unsupported regexp flag %c", f)
		}
	}
	re, err := regexp.Compile(prefix + source)
	if err != nil {
		return nil, object.NewError(pos, "SyntaxError: invalid regexp: %v", err)
	}
	return &object.RegExp{Source: source, Flags: flags, Re: re}, nil
}

func regexpReceiver(env *object.Environment, pos ast.Position, name string) (*object.RegExp, *object.Error) {
	re, ok := env.Extra.(*object.RegExp)
	if !ok {
		return nil, object.NewError(pos, "TypeError: %s requires RegExp receiver", name)
	}
	return re, nil
}

func regexpFromArg(pos ast.Position, arg object.Object) (*object.RegExp, *object.Error) {
	if re, ok := arg.(*object.RegExp); ok {
		return re, nil
	}
	return compileRegExp(pos, arg.Inspect(), "")
}

func builtinRegExpTest(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	re, err := regexpReceiver(env, pos, "RegExp.test")
	if err != nil {
		return err
	}
	input := ""
	if len(args) > 0 {
		input = args[0].Inspect()
	}
	return object.NativeBool(re.Re.MatchString(input))
}

func builtinRegExpExec(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	re, err := regexpReceiver(env, pos, "RegExp.exec")
	if err != nil {
		return err
	}
	input := ""
	if len(args) > 0 {
		input = args[0].Inspect()
	}
	return regexpExecArray(re, input)
}

func regexpExecArray(re *object.RegExp, input string) object.Object {
	match := re.Re.FindStringSubmatch(input)
	if match == nil {
		return object.NULL
	}
	elements := make([]object.Object, len(match))
	for i, item := range match {
		elements[i] = &object.String{Value: item}
	}
	arr := &object.Array{Elements: elements}
	return arr
}

func builtinRegExpToString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	re, err := regexpReceiver(env, pos, "RegExp.toString")
	if err != nil {
		return err
	}
	return &object.String{Value: re.Inspect()}
}

func regexpReplace(input string, re *object.RegExp, replacement string) string {
	if strings.Contains(re.Flags, "g") {
		return re.Re.ReplaceAllString(input, replacement)
	}
	loc := re.Re.FindStringIndex(input)
	if loc == nil {
		return input
	}
	return input[:loc[0]] + re.Re.ReplaceAllString(input[loc[0]:loc[1]], replacement) + input[loc[1]:]
}
