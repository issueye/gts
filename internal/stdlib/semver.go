package stdlib

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

var semverRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z\-.]+))?(?:\+([0-9A-Za-z\-.]+))?$`)

func init() {
	module.RegisterNative("@std/semver", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initSemverModule(exports)
		return exports, nil
	})
}

func initSemverModule(exports *object.Hash) {
	setHashMember(exports, "parse", &object.Builtin{Name: "semver.parse", Fn: semverParse})
	setHashMember(exports, "valid", &object.Builtin{Name: "semver.valid", Fn: semverValid})
	setHashMember(exports, "compare", &object.Builtin{Name: "semver.compare", Fn: semverCompare})
	setHashMember(exports, "gt", &object.Builtin{Name: "semver.gt", Fn: semverGt})
	setHashMember(exports, "gte", &object.Builtin{Name: "semver.gte", Fn: semverGte})
	setHashMember(exports, "lt", &object.Builtin{Name: "semver.lt", Fn: semverLt})
	setHashMember(exports, "lte", &object.Builtin{Name: "semver.lte", Fn: semverLte})
	setHashMember(exports, "eq", &object.Builtin{Name: "semver.eq", Fn: semverEq})
	setHashMember(exports, "neq", &object.Builtin{Name: "semver.neq", Fn: semverNeq})
	setHashMember(exports, "inc", &object.Builtin{Name: "semver.inc", Fn: semverInc})
	setHashMember(exports, "satisfies", &object.Builtin{Name: "semver.satisfies", Fn: semverSatisfies})
}

type semver struct {
	major, minor, patch int64
	prerelease          []string
	build               []string
}

func parseSemver(version string) (*semver, error) {
	matches := semverRegex.FindStringSubmatch(strings.TrimSpace(version))
	if matches == nil {
		return nil, fmt.Errorf("invalid version: %s", version)
	}
	major, _ := strconv.ParseInt(matches[1], 10, 64)
	minor, _ := strconv.ParseInt(matches[2], 10, 64)
	patch, _ := strconv.ParseInt(matches[3], 10, 64)
	var prerelease, build []string
	if matches[4] != "" {
		prerelease = strings.Split(matches[4], ".")
	}
	if matches[5] != "" {
		build = strings.Split(matches[5], ".")
	}
	return &semver{major, minor, patch, prerelease, build}, nil
}

func (v *semver) compare(other *semver) int {
	if v.major != other.major {
		if v.major < other.major {
			return -1
		}
		return 1
	}
	if v.minor != other.minor {
		if v.minor < other.minor {
			return -1
		}
		return 1
	}
	if v.patch != other.patch {
		if v.patch < other.patch {
			return -1
		}
		return 1
	}
	if len(v.prerelease) == 0 && len(other.prerelease) > 0 {
		return 1
	}
	if len(v.prerelease) > 0 && len(other.prerelease) == 0 {
		return -1
	}
	for i := 0; i < len(v.prerelease) && i < len(other.prerelease); i++ {
		a, erra := strconv.ParseInt(v.prerelease[i], 10, 64)
		b, errb := strconv.ParseInt(other.prerelease[i], 10, 64)
		if erra == nil && errb == nil {
			if a != b {
				if a < b {
					return -1
				}
				return 1
			}
		} else {
			cmp := strings.Compare(v.prerelease[i], other.prerelease[i])
			if cmp != 0 {
				return cmp
			}
		}
	}
	if len(v.prerelease) != len(other.prerelease) {
		if len(v.prerelease) < len(other.prerelease) {
			return -1
		}
		return 1
	}
	return 0
}

func semverParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	version, errObj := requiredString(pos, "semver.parse", args, 0, "version")
	if errObj != nil {
		return errObj
	}
	v, err := parseSemver(version)
	if err != nil {
		return object.NewError(pos, "semver.parse: %v", err)
	}
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(result, "major", &object.Number{Value: float64(v.major)})
	setHashMember(result, "minor", &object.Number{Value: float64(v.minor)})
	setHashMember(result, "patch", &object.Number{Value: float64(v.patch)})
	preArr := &object.Array{Elements: make([]object.Object, len(v.prerelease))}
	for i, p := range v.prerelease {
		if n, err := strconv.ParseInt(p, 10, 64); err == nil {
			preArr.Elements[i] = &object.Number{Value: float64(n)}
		} else {
			preArr.Elements[i] = &object.String{Value: p}
		}
	}
	setHashMember(result, "prerelease", preArr)
	buildArr := &object.Array{Elements: make([]object.Object, len(v.build))}
	for i, b := range v.build {
		buildArr.Elements[i] = &object.String{Value: b}
	}
	setHashMember(result, "build", buildArr)
	return result
}

func semverValid(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	version, errObj := requiredString(pos, "semver.valid", args, 0, "version")
	if errObj != nil {
		return errObj
	}
	_, err := parseSemver(version)
	return object.NativeBool(err == nil)
}

func semverCompare(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	v1, errObj := requiredString(pos, "semver.compare", args, 0, "v1")
	if errObj != nil {
		return errObj
	}
	v2, errObj := requiredString(pos, "semver.compare", args, 1, "v2")
	if errObj != nil {
		return errObj
	}
	ver1, err := parseSemver(v1)
	if err != nil {
		return object.NewError(pos, "semver.compare: %v", err)
	}
	ver2, err := parseSemver(v2)
	if err != nil {
		return object.NewError(pos, "semver.compare: %v", err)
	}
	return &object.Number{Value: float64(ver1.compare(ver2))}
}

func semverGt(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	result := semverCompare(env, pos, args...)
	if result.Type() == object.ERROR_OBJ {
		return result
	}
	return object.NativeBool(result.(*object.Number).Value > 0)
}

func semverGte(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	result := semverCompare(env, pos, args...)
	if result.Type() == object.ERROR_OBJ {
		return result
	}
	return object.NativeBool(result.(*object.Number).Value >= 0)
}

func semverLt(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	result := semverCompare(env, pos, args...)
	if result.Type() == object.ERROR_OBJ {
		return result
	}
	return object.NativeBool(result.(*object.Number).Value < 0)
}

func semverLte(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	result := semverCompare(env, pos, args...)
	if result.Type() == object.ERROR_OBJ {
		return result
	}
	return object.NativeBool(result.(*object.Number).Value <= 0)
}

func semverEq(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	result := semverCompare(env, pos, args...)
	if result.Type() == object.ERROR_OBJ {
		return result
	}
	return object.NativeBool(result.(*object.Number).Value == 0)
}

func semverNeq(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	result := semverCompare(env, pos, args...)
	if result.Type() == object.ERROR_OBJ {
		return result
	}
	return object.NativeBool(result.(*object.Number).Value != 0)
}

func semverInc(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	version, errObj := requiredString(pos, "semver.inc", args, 0, "version")
	if errObj != nil {
		return errObj
	}
	release, errObj := requiredString(pos, "semver.inc", args, 1, "release")
	if errObj != nil {
		return errObj
	}
	v, err := parseSemver(version)
	if err != nil {
		return object.NewError(pos, "semver.inc: %v", err)
	}
	switch release {
	case "major":
		return &object.String{Value: fmt.Sprintf("%d.0.0", v.major+1)}
	case "minor":
		return &object.String{Value: fmt.Sprintf("%d.%d.0", v.major, v.minor+1)}
	case "patch":
		return &object.String{Value: fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch+1)}
	case "prerelease":
		return &object.String{Value: fmt.Sprintf("%d.%d.%d-0", v.major, v.minor, v.patch+1)}
	default:
		return object.NewError(pos, "semver.inc: invalid release type: %s", release)
	}
}

func semverSatisfies(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	version, errObj := requiredString(pos, "semver.satisfies", args, 0, "version")
	if errObj != nil {
		return errObj
	}
	rangeStr, errObj := requiredString(pos, "semver.satisfies", args, 1, "range")
	if errObj != nil {
		return errObj
	}
	v, err := parseSemver(version)
	if err != nil {
		return object.NewError(pos, "semver.satisfies: %v", err)
	}
	if strings.HasPrefix(rangeStr, "^") {
		base, err := parseSemver(rangeStr[1:])
		if err != nil {
			return object.NewError(pos, "semver.satisfies: %v", err)
		}
		if v.major != base.major {
			return object.FALSE
		}
		if v.compare(base) < 0 {
			return object.FALSE
		}
		return object.TRUE
	}
	if strings.HasPrefix(rangeStr, "~") {
		base, err := parseSemver(rangeStr[1:])
		if err != nil {
			return object.NewError(pos, "semver.satisfies: %v", err)
		}
		if v.major != base.major || v.minor != base.minor {
			return object.FALSE
		}
		if v.compare(base) < 0 {
			return object.FALSE
		}
		return object.TRUE
	}
	parts := strings.Fields(rangeStr)
	if len(parts) >= 2 {
		for i := 0; i < len(parts); i++ {
			op := parts[i]
			if i+1 >= len(parts) {
				break
			}
			verStr := parts[i+1]
			cmpVer, err := parseSemver(verStr)
			if err != nil {
				continue
			}
			cmp := v.compare(cmpVer)
			switch op {
			case ">=":
				if cmp < 0 {
					return object.FALSE
				}
			case ">":
				if cmp <= 0 {
					return object.FALSE
				}
			case "<=":
				if cmp > 0 {
					return object.FALSE
				}
			case "<":
				if cmp >= 0 {
					return object.FALSE
				}
			case "=", "==":
				if cmp != 0 {
					return object.FALSE
				}
			}
			i++
		}
		return object.TRUE
	}
	base, err := parseSemver(rangeStr)
	if err != nil {
		return object.NewError(pos, "semver.satisfies: %v", err)
	}
	return object.NativeBool(v.compare(base) == 0)
}
