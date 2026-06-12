package stdlib

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/test", func(env *object.Environment) (object.Object, error) {
		runner := NewTestRunner()
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initTestModule(exports, runner)
		return exports, nil
	})
}

// TestRunner 管理所有测试
type TestRunner struct {
	mu          sync.Mutex
	suites      []*TestSuite
	currentSuite *TestSuite
	config      TestConfig
	stats       TestStats
}

type TestConfig struct {
	Timeout  time.Duration
	Verbose  bool
	Bail     bool
	Parallel bool
}

type TestStats struct {
	Total   int
	Passed  int
	Failed  int
	Skipped int
	Errors  []TestError
}

type TestSuite struct {
	Name     string
	Tests    []*TestCase
	Hooks    TestHooks
	Parent   *TestSuite
	Children []*TestSuite
	Skip     bool
	Only     bool
}

type TestCase struct {
	Name    string
	Fn      object.Object
	Async   bool
	Skip    bool
	Only    bool
	Timeout time.Duration
}

type TestHooks struct {
	BeforeAll  []object.Object
	AfterAll   []object.Object
	BeforeEach []object.Object
	AfterEach  []object.Object
}

type TestError struct {
	Suite   string
	Test    string
	Message string
}

func NewTestRunner() *TestRunner {
	return &TestRunner{
		config: TestConfig{
			Timeout: 5 * time.Second,
			Verbose: false,
			Bail:    false,
		},
	}
}

func initTestModule(exports *object.Hash, runner *TestRunner) {
	// 基础测试定义
	setHashMember(exports, "test", createTestFn(runner, false, false))
	setHashMember(exports, "it", createTestFn(runner, false, false))
	setHashMember(exports, "skip", createTestFn(runner, true, false))
	setHashMember(exports, "only", createTestFn(runner, false, true))

	// 测试套件
	setHashMember(exports, "describe", createDescribeFn(runner, false, false))

	// 钩子函数
	setHashMember(exports, "beforeAll", createHookFn(runner, "beforeAll"))
	setHashMember(exports, "afterAll", createHookFn(runner, "afterAll"))
	setHashMember(exports, "beforeEach", createHookFn(runner, "beforeEach"))
	setHashMember(exports, "afterEach", createHookFn(runner, "afterEach"))

	// 断言
	setHashMember(exports, "expect", &object.Builtin{
		Name: "test.expect",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "expect requires a value")
			}
			return createExpectation(args[0], false)
		},
	})

	// 配置
	setHashMember(exports, "configure", &object.Builtin{
		Name: "test.configure",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.UNDEFINED
			}
			hash, ok := args[0].(*object.Hash)
			if !ok {
				return object.NewError(pos, "configure expects an object")
			}
			applyConfig(runner, hash)
			return object.UNDEFINED
		},
	})

	// 运行测试
	setHashMember(exports, "run", &object.Builtin{
		Name: "test.run",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			return runTests(runner, env, pos)
		},
	})
}

func createTestFn(runner *TestRunner, skip, only bool) *object.Builtin {
	return &object.Builtin{
		Name: "test",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 2 {
				return object.NewError(pos, "test requires name and function")
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "test name must be string")
			}

			runner.mu.Lock()
			defer runner.mu.Unlock()

			tc := &TestCase{
				Name:    name.Value,
				Fn:      args[1],
				Skip:    skip,
				Only:    only,
				Timeout: runner.config.Timeout,
			}

			if runner.currentSuite != nil {
				runner.currentSuite.Tests = append(runner.currentSuite.Tests, tc)
			} else {
				// 顶层测试，创建匿名套件
				if len(runner.suites) == 0 || runner.suites[len(runner.suites)-1].Name != "" {
					runner.suites = append(runner.suites, &TestSuite{})
				}
				runner.suites[len(runner.suites)-1].Tests = append(runner.suites[len(runner.suites)-1].Tests, tc)
			}

			return object.UNDEFINED
		},
	}
}

func createDescribeFn(runner *TestRunner, skip, only bool) *object.Builtin {
	return &object.Builtin{
		Name: "test.describe",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 2 {
				return object.NewError(pos, "describe requires name and function")
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "describe name must be string")
			}
			fn, ok := args[1].(*object.Function)
			if !ok {
				return object.NewError(pos, "describe expects function")
			}

			runner.mu.Lock()
			suite := &TestSuite{
				Name: name.Value,
				Skip: skip,
				Only: only,
			}
			if runner.currentSuite != nil {
				suite.Parent = runner.currentSuite
				runner.currentSuite.Children = append(runner.currentSuite.Children, suite)
			} else {
				runner.suites = append(runner.suites, suite)
			}
			prevSuite := runner.currentSuite
			runner.currentSuite = suite
			runner.mu.Unlock()

			// 执行套件函数（收集测试）
			_ = callFunction(env, fn, pos)

			runner.mu.Lock()
			runner.currentSuite = prevSuite
			runner.mu.Unlock()

			return object.UNDEFINED
		},
	}
}

func createHookFn(runner *TestRunner, hookType string) *object.Builtin {
	return &object.Builtin{
		Name: "test." + hookType,
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "%s", hookType+" requires function")
			}

			runner.mu.Lock()
			defer runner.mu.Unlock()

			if runner.currentSuite == nil {
				return object.NewError(pos, "%s", hookType+" must be inside describe")
			}

			switch hookType {
			case "beforeAll":
				runner.currentSuite.Hooks.BeforeAll = append(runner.currentSuite.Hooks.BeforeAll, args[0])
			case "afterAll":
				runner.currentSuite.Hooks.AfterAll = append(runner.currentSuite.Hooks.AfterAll, args[0])
			case "beforeEach":
				runner.currentSuite.Hooks.BeforeEach = append(runner.currentSuite.Hooks.BeforeEach, args[0])
			case "afterEach":
				runner.currentSuite.Hooks.AfterEach = append(runner.currentSuite.Hooks.AfterEach, args[0])
			}

			return object.UNDEFINED
		},
	}
}

func createExpectation(value object.Object, not bool) *object.Instance {
	inst := &object.Instance{
		Props: make(map[string]object.Object),
	}

	// toBe
	inst.Props["toBe"] = &object.Builtin{
		Name: "expect.toBe",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "toBe requires expected value")
			}
			equal := testObjectsEqual(value, args[0], true)
			if not {
				equal = !equal
			}
			if !equal {
				notStr := ""
				if not {
					notStr = "not "
				}
				msg := fmt.Sprintf("Expected %s%s to be %s", notStr, value.Inspect(), args[0].Inspect())
				return object.NewError(pos, "%s", msg)
			}
			return object.UNDEFINED
		},
	}

	// toEqual
	inst.Props["toEqual"] = &object.Builtin{
		Name: "expect.toEqual",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "toEqual requires expected value")
			}
			equal := testObjectsEqual(value, args[0], false)
			if not {
				equal = !equal
			}
			if !equal {
				notStr := ""
				if not {
					notStr = "not "
				}
				msg := fmt.Sprintf("Expected %s%s to equal %s", notStr, value.Inspect(), args[0].Inspect())
				return object.NewError(pos, "%s", msg)
			}
			return object.UNDEFINED
		},
	}

	// toBeTruthy
	inst.Props["toBeTruthy"] = &object.Builtin{
		Name: "expect.toBeTruthy",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			truthy := isTruthy(value)
			if not {
				truthy = !truthy
			}
			if !truthy {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected %s%s to be truthy", notStr, value.Inspect())
			}
			return object.UNDEFINED
		},
	}

	// toBeFalsy
	inst.Props["toBeFalsy"] = &object.Builtin{
		Name: "expect.toBeFalsy",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			falsy := !isTruthy(value)
			if not {
				falsy = !falsy
			}
			if !falsy {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected %s%s to be falsy", notStr, value.Inspect())
			}
			return object.UNDEFINED
		},
	}

	// toBeDefined
	inst.Props["toBeDefined"] = &object.Builtin{
		Name: "expect.toBeDefined",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			defined := value.Type() != object.UNDEFINED_OBJ
			if not {
				defined = !defined
			}
			if !defined {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected value %sto be defined", notStr)
			}
			return object.UNDEFINED
		},
	}

	// toBeUndefined
	inst.Props["toBeUndefined"] = &object.Builtin{
		Name: "expect.toBeUndefined",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			undefined := value.Type() == object.UNDEFINED_OBJ
			if not {
				undefined = !undefined
			}
			if !undefined {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected %s %sto be undefined", value.Inspect(), notStr)
			}
			return object.UNDEFINED
		},
	}

	// toBeNull
	inst.Props["toBeNull"] = &object.Builtin{
		Name: "expect.toBeNull",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			isNull := value.Type() == object.NULL_OBJ
			if not {
				isNull = !isNull
			}
			if !isNull {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected %s %sto be null", value.Inspect(), notStr)
			}
			return object.UNDEFINED
		},
	}

	// toBeGreaterThan
	inst.Props["toBeGreaterThan"] = createNumberCompare(value, not, ">")
	inst.Props["toBeGreaterThanOrEqual"] = createNumberCompare(value, not, ">=")
	inst.Props["toBeLessThan"] = createNumberCompare(value, not, "<")
	inst.Props["toBeLessThanOrEqual"] = createNumberCompare(value, not, "<=")

	// toMatch
	inst.Props["toMatch"] = &object.Builtin{
		Name: "expect.toMatch",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "toMatch requires pattern")
			}
			str, ok := value.(*object.String)
			if !ok {
				return object.NewError(pos, "toMatch expects string value")
			}
			pattern := ""
			if re, ok := args[0].(*object.RegExp); ok {
				pattern = re.Source
			} else if s, ok := args[0].(*object.String); ok {
				pattern = s.Value
			} else {
				return object.NewError(pos, "toMatch expects regex or string")
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return object.NewError(pos, "Invalid regex: %v", err)
			}
			matches := re.MatchString(str.Value)
			if not {
				matches = !matches
			}
			if !matches {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected %s %sto match %s", str.Value, notStr, pattern)
			}
			return object.UNDEFINED
		},
	}

	// toContain
	inst.Props["toContain"] = &object.Builtin{
		Name: "expect.toContain",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "toContain requires item")
			}
			contains := false
			switch v := value.(type) {
			case *object.String:
				if s, ok := args[0].(*object.String); ok {
					contains = strings.Contains(v.Value, s.Value)
				}
			case *object.Array:
				for _, elem := range v.Elements {
					if testObjectsEqual(elem, args[0], false) {
						contains = true
						break
					}
				}
			default:
				return object.NewError(pos, "toContain expects string or array")
			}
			if not {
				contains = !contains
			}
			if !contains {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected %s%s to contain %s", notStr, value.Inspect(), args[0].Inspect())
			}
			return object.UNDEFINED
		},
	}

	// toHaveLength
	inst.Props["toHaveLength"] = &object.Builtin{
		Name: "expect.toHaveLength",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "toHaveLength requires length")
			}
			expected, ok := args[0].(*object.Number)
			if !ok {
				return object.NewError(pos, "toHaveLength expects number")
			}
			length := int64(-1)
			switch v := value.(type) {
			case *object.String:
				length = int64(len([]rune(v.Value)))
			case *object.Array:
				length = int64(len(v.Elements))
			default:
				return object.NewError(pos, "toHaveLength expects string or array")
			}
			match := length == int64(expected.Value)
			if not {
				match = !match
			}
			if !match {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected length %s%.0f but got %d", notStr, expected.Value, length)
			}
			return object.UNDEFINED
		},
	}

	// toHaveProperty
	inst.Props["toHaveProperty"] = &object.Builtin{
		Name: "expect.toHaveProperty",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "toHaveProperty requires key")
			}
			key, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "toHaveProperty expects string key")
			}
			has := false
			var propValue object.Object
			if hash, ok := value.(*object.Hash); ok {
				hashKey := object.HashKeyFor(&object.String{Value: key.Value})
				if pair, exists := hash.Pairs[hashKey]; exists {
					has = true
					propValue = pair.Value
				}
			} else if inst, ok := value.(*object.Instance); ok {
				if val, exists := inst.Props[key.Value]; exists {
					has = true
					propValue = val
				}
			} else {
				return object.NewError(pos, "toHaveProperty expects object")
			}
			if not {
				has = !has
			}
			if !has {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected %sobject to have property %s", notStr, key.Value)
			}
			// 如果提供了值，检查值是否匹配
			if len(args) > 1 && !not {
				if !testObjectsEqual(propValue, args[1], false) {
					return object.NewError(pos, "Property %s has value %s but expected %s",
						key.Value, propValue.Inspect(), args[1].Inspect())
				}
			}
			return object.UNDEFINED
		},
	}

	// toThrow
	inst.Props["toThrow"] = &object.Builtin{
		Name: "expect.toThrow",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			fn, ok := value.(*object.Function)
			if !ok {
				return object.NewError(pos, "toThrow expects function")
			}
			result := callFunction(env, fn, pos)
			threw := result != nil && result.Type() == object.ERROR_OBJ
			if not {
				threw = !threw
			}
			if !threw {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected function %sto throw", notStr)
			}
			// 如果提供了错误消息或类型，检查匹配
			if len(args) > 0 && !not && threw {
				if s, ok := args[0].(*object.String); ok {
					if !strings.Contains(result.Inspect(), s.Value) {
						return object.NewError(pos, "Expected error to contain %s", s.Value)
					}
				}
			}
			return object.UNDEFINED
		},
	}

	// not
	inst.Props["not"] = createExpectation(value, !not)

	return inst
}

func createNumberCompare(value object.Object, not bool, op string) *object.Builtin {
	return &object.Builtin{
		Name: "expect.compare",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "comparison requires value")
			}
			num, ok := value.(*object.Number)
			if !ok {
				return object.NewError(pos, "comparison expects number")
			}
			expected, ok := args[0].(*object.Number)
			if !ok {
				return object.NewError(pos, "comparison expects number")
			}
			match := false
			switch op {
			case ">":
				match = num.Value > expected.Value
			case ">=":
				match = num.Value >= expected.Value
			case "<":
				match = num.Value < expected.Value
			case "<=":
				match = num.Value <= expected.Value
			}
			if not {
				match = !match
			}
			if !match {
				notStr := ""
				if not {
					notStr = "not "
				}
				return object.NewError(pos, "Expected %.0f %s%s %.0f", num.Value, notStr, op, expected.Value)
			}
			return object.UNDEFINED
		},
	}
}

func runTests(runner *TestRunner, env *object.Environment, pos ast.Position) object.Object {
	runner.mu.Lock()
	suites := runner.suites
	runner.mu.Unlock()

	startTime := time.Now()

	// 运行所有套件
	for _, suite := range suites {
		runSuite(runner, suite, env, pos)
	}

	duration := time.Since(startTime)

	// 输出报告
	fmt.Printf("\n")
	if runner.stats.Failed == 0 {
		fmt.Printf("✓ %d tests passed (%dms)\n", runner.stats.Passed, duration.Milliseconds())
	} else {
		fmt.Printf("✗ %d/%d tests failed (%dms)\n",
			runner.stats.Failed, runner.stats.Total, duration.Milliseconds())
		for _, err := range runner.stats.Errors {
			fmt.Printf("\n  %s > %s\n    %s\n", err.Suite, err.Test, err.Message)
		}
	}

	// 返回结果对象
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(result, "total", &object.Number{Value: float64(runner.stats.Total)})
	setHashMember(result, "passed", &object.Number{Value: float64(runner.stats.Passed)})
	setHashMember(result, "failed", &object.Number{Value: float64(runner.stats.Failed)})
	setHashMember(result, "skipped", &object.Number{Value: float64(runner.stats.Skipped)})
	setHashMember(result, "duration", &object.Number{Value: float64(duration.Milliseconds())})
	return result
}

func runSuite(runner *TestRunner, suite *TestSuite, env *object.Environment, pos ast.Position) {
	if suite.Skip {
		return
	}

	// beforeAll
	for _, hook := range suite.Hooks.BeforeAll {
		if fn, ok := hook.(*object.Function); ok {
			callFunction(env, fn, pos)
		}
	}

	// 运行测试
	for _, test := range suite.Tests {
		if test.Skip {
			runner.stats.Skipped++
			continue
		}

		runner.stats.Total++

		// beforeEach
		for _, hook := range suite.Hooks.BeforeEach {
			if fn, ok := hook.(*object.Function); ok {
				callFunction(env, fn, pos)
			}
		}

		// 运行测试
		var result object.Object
		if fn, ok := test.Fn.(*object.Function); ok {
			result = callFunction(env, fn, pos)
		}

		// 检查结果
		if result != nil && result.Type() == object.ERROR_OBJ {
			runner.stats.Failed++
			runner.stats.Errors = append(runner.stats.Errors, TestError{
				Suite:   suite.Name,
				Test:    test.Name,
				Message: result.Inspect(),
			})
			if runner.config.Verbose {
				fmt.Printf("  ✗ %s\n", test.Name)
			}
		} else {
			runner.stats.Passed++
			if runner.config.Verbose {
				fmt.Printf("  ✓ %s\n", test.Name)
			}
		}

		// afterEach
		for _, hook := range suite.Hooks.AfterEach {
			if fn, ok := hook.(*object.Function); ok {
				callFunction(env, fn, pos)
			}
		}
	}

	// 运行子套件
	for _, child := range suite.Children {
		runSuite(runner, child, env, pos)
	}

	// afterAll
	for _, hook := range suite.Hooks.AfterAll {
		if fn, ok := hook.(*object.Function); ok {
			callFunction(env, fn, pos)
		}
	}
}

func callFunction(env *object.Environment, fn *object.Function, pos ast.Position) object.Object {
	// 简化版函数调用 - 实际需要通过 evaluator 调用
	// 这里暂时返回 nil，实际实现需要集成到 evaluator
	// TODO: 集成到实际的 evaluator.Eval
	return nil
}

func applyConfig(runner *TestRunner, hash *object.Hash) {
	for _, pair := range hash.Pairs {
		key := pair.Key.Inspect()
		switch key {
		case "timeout":
			if num, ok := pair.Value.(*object.Number); ok {
				runner.config.Timeout = time.Duration(num.Value) * time.Millisecond
			}
		case "verbose":
			if b, ok := pair.Value.(*object.Boolean); ok {
				runner.config.Verbose = b.Value
			}
		case "bail":
			if b, ok := pair.Value.(*object.Boolean); ok {
				runner.config.Bail = b.Value
			}
		}
	}
}

func testObjectsEqual(a, b object.Object, strict bool) bool {
	if a.Type() != b.Type() && strict {
		return false
	}

	switch a := a.(type) {
	case *object.Number:
		if b, ok := b.(*object.Number); ok {
			return a.Value == b.Value
		}
	case *object.String:
		if b, ok := b.(*object.String); ok {
			return a.Value == b.Value
		}
	case *object.Boolean:
		if b, ok := b.(*object.Boolean); ok {
			return a.Value == b.Value
		}
	case *object.Array:
		if b, ok := b.(*object.Array); ok {
			if len(a.Elements) != len(b.Elements) {
				return false
			}
			for i := range a.Elements {
				if !testObjectsEqual(a.Elements[i], b.Elements[i], strict) {
					return false
				}
			}
			return true
		}
	case *object.Hash:
		if b, ok := b.(*object.Hash); ok {
			if len(a.Pairs) != len(b.Pairs) {
				return false
			}
			for key, pairA := range a.Pairs {
				pairB, exists := b.Pairs[key]
				if !exists || !testObjectsEqual(pairA.Value, pairB.Value, strict) {
					return false
				}
			}
			return true
		}
	}

	return reflect.DeepEqual(a, b)
}

func isTruthy(obj object.Object) bool {
	switch obj.Type() {
	case object.NULL_OBJ, object.UNDEFINED_OBJ:
		return false
	case object.BOOLEAN_OBJ:
		return obj.(*object.Boolean).Value
	case object.NUMBER_OBJ:
		return obj.(*object.Number).Value != 0
	case object.STRING_OBJ:
		return obj.(*object.String).Value != ""
	default:
		return true
	}
}
