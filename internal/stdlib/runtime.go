package stdlib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/async"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/packagefile"
	"github.com/issueye/goscript/internal/parser"
)

func init() {
	module.RegisterNative("@std/runtime", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initRuntimeModule(exports)
		return exports, nil
	})
}

func initRuntimeModule(exports *object.Hash) {
	setHashMember(exports, "runScript", &object.Builtin{Name: "runtime.runScript", Fn: runtimeRunScript})
	setHashMember(exports, "runTool", &object.Builtin{Name: "runtime.runTool", Fn: runtimeRunTool})
}

type runtimeScriptOptions struct {
	cwd      string
	argv     []string
	autoMain bool
}

type runtimeExecution struct {
	env     *object.Environment
	exports object.Object
	cwd     string
}

func runtimeRunScript(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "runtime.runScript", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	opts, errObj := runtimeOptions(pos, "runtime.runScript", args, 1)
	if errObj != nil {
		return errObj
	}
	exec, errObj := runtimeExecuteScript(env, pos, path, opts)
	if errObj != nil {
		return errObj
	}
	return exec.exports
}

func runtimeRunTool(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "runtime.runTool", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	var input object.Object = object.UNDEFINED
	if len(args) >= 2 {
		input = args[1]
	}
	opts, errObj := runtimeOptions(pos, "runtime.runTool", args, 2)
	if errObj != nil {
		return errObj
	}
	exec, errObj := runtimeExecuteScript(env, pos, path, opts)
	if errObj != nil {
		return errObj
	}
	exports, ok := exec.exports.(*object.Hash)
	if !ok {
		return object.NewError(pos, "runtime.runTool: %s exports must be an object", path)
	}
	runFn, ok := hashValue(exports, "run")
	if !ok || runFn == object.UNDEFINED || runFn == object.NULL {
		return object.NewError(pos, "runtime.runTool: %s must export run(input)", path)
	}
	runCwd := opts.cwd
	if runCwd == "" {
		runCwd = exec.cwd
	}
	restore, err := runtimeEnterWorkingDir(runCwd)
	if err != nil {
		return object.NewError(pos, "runtime.runTool: %v", err)
	}
	defer restore()
	result := callRuntimeFunction(runFn, exec.env, []object.Object{input}, pos)
	if promise, ok := result.(*object.Promise); ok {
		result = promise.Wait()
	}
	exec.env.VM().WaitAsync()
	if object.IsRuntimeError(result) {
		return result
	}
	return result
}

func runtimeOptions(pos ast.Position, name string, args []object.Object, index int) (runtimeScriptOptions, *object.Error) {
	opts := runtimeScriptOptions{}
	if len(args) <= index || args[index] == object.UNDEFINED || args[index] == object.NULL {
		return opts, nil
	}
	hash, ok := args[index].(*object.Hash)
	if !ok {
		return opts, object.NewError(pos, "%s: options must be an object", name)
	}
	if cwdObj, ok := hashValue(hash, "cwd"); ok && cwdObj != object.UNDEFINED && cwdObj != object.NULL {
		cwd, ok := cwdObj.(*object.String)
		if !ok {
			return opts, object.NewError(pos, "%s: cwd must be a string", name)
		}
		opts.cwd = cwd.Value
	}
	if argvObj, ok := hashValue(hash, "argv"); ok && argvObj != object.UNDEFINED && argvObj != object.NULL {
		argv, ok := argvObj.(*object.Array)
		if !ok {
			return opts, object.NewError(pos, "%s: argv must be an array", name)
		}
		opts.argv = toStringSlice(argv.Elements)
	}
	if autoMainObj, ok := hashValue(hash, "autoMain"); ok && autoMainObj != object.UNDEFINED && autoMainObj != object.NULL {
		autoMain, ok := autoMainObj.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "%s: autoMain must be a boolean", name)
		}
		opts.autoMain = autoMain.Value
	}
	return opts, nil
}

func runtimeExecuteScript(caller *object.Environment, pos ast.Position, path string, opts runtimeScriptOptions) (runtimeExecution, *object.Error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return runtimeExecution{}, object.NewError(pos, "runtime: %v", err)
	}
	src, err := os.ReadFile(absPath)
	if err != nil {
		return runtimeExecution{}, object.NewError(pos, "runtime: %v", err)
	}
	cwd := opts.cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return runtimeExecution{}, object.NewError(pos, "runtime: %v", err)
		}
	}
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return runtimeExecution{}, object.NewError(pos, "runtime: %v", err)
	}

	vm := object.NewVirtualMachine()
	vm.SetTypeCheck(caller.VM().TypeCheck())
	pool := async.NewPool(1)
	vm.SetSpawner(pool.Go)
	defer pool.Wait()

	argv := opts.argv
	if len(argv) == 0 {
		argv = []string{runtimeExecutableArgv0(), absPath}
	}
	vm.SetArgv(argv)

	cache := module.NewCacheWithVM(vm)
	rootDir := module.FindProjectRoot(filepath.Dir(absPath))
	resolver := module.NewResolver(rootDir)
	env := vm.NewEnvironment()
	module.SetupExports(env)
	configureRuntimeModuleLoaders(env, cache, resolver, rootDir, filepath.Dir(absPath))

	restore, err := runtimeEnterWorkingDir(cwd)
	if err != nil {
		return runtimeExecution{}, object.NewError(pos, "runtime: %v", err)
	}
	defer restore()

	if _, err := evalRuntimeSource(string(src), absPath, env); err != nil {
		return runtimeExecution{}, object.NewError(pos, "%v", err)
	}
	if opts.autoMain {
		result := callRuntimeMain(env, absPath)
		if promise, ok := result.(*object.Promise); ok {
			result = promise.Wait()
		}
		if object.IsRuntimeError(result) {
			return runtimeExecution{}, object.NewError(pos, "%s", result.Inspect())
		}
	}
	vm.WaitAsync()
	return runtimeExecution{env: env, exports: module.GetExports(env), cwd: cwd}, nil
}

func configureRuntimeModuleLoaders(env *object.Environment, cache *module.Cache, resolver *module.Resolver, rootDir, baseDir string) {
	env.ModuleDir = baseDir
	requireFromEnv := func(loadEnv *object.Environment, specifier string) (object.Object, error) {
		currentBaseDir := loadEnv.ModuleDir
		if currentBaseDir == "" {
			currentBaseDir = baseDir
		}
		resolved, err := resolver.Resolve(specifier, module.ResolveOptions{ProjectRoot: rootDir, BaseDir: currentBaseDir})
		if err != nil {
			return nil, err
		}
		if resolved.Kind == module.ModuleKindNative {
			native, ok := module.GetNative(specifier, loadEnv)
			if !ok {
				return nil, fmt.Errorf("native module %s is not registered", specifier)
			}
			return native, nil
		}
		return requireRuntimeResolved(cache, resolver, rootDir, resolved)
	}
	require := func(specifier string) (object.Object, error) {
		return requireFromEnv(env, specifier)
	}
	evaluator.RegisterBuiltinsWithCache(env, require)
	env.VM().SetImportFunc(func(importEnv *object.Environment, specifier string) (object.Object, error) {
		return requireFromEnv(importEnv, specifier)
	})
}

func requireRuntimeResolved(cache *module.Cache, resolver *module.Resolver, rootDir string, resolved module.ResolvedModule) (object.Object, error) {
	if resolved.Path == "" {
		return nil, fmt.Errorf("module %s resolved without a source path", resolved.Specifier)
	}
	cacheKey := resolved.ID
	if cacheKey == "" {
		cacheKey = resolved.Path
	}
	if cached := cache.Get(cacheKey); cached != nil {
		return module.GetExports(cached), nil
	}
	env := cache.GetOrCreate(cacheKey)
	module.SetupExports(env)
	configureRuntimeModuleLoaders(env, cache, resolver, rootDir, runtimeResolvedModuleDir(resolved))
	src, err := readRuntimeResolvedSource(resolved)
	if err != nil {
		return nil, err
	}
	if _, err := evalRuntimeSource(src, runtimeResolvedFile(resolved), env); err != nil {
		return nil, err
	}
	return module.GetExports(env), nil
}

func evalRuntimeSource(src, file string, env *object.Environment) (object.Object, error) {
	l := lexer.New(src)
	p := parser.New(l, file)
	program := p.ParseProgram()
	var parseErrors []string
	parseErrors = append(parseErrors, l.Errors()...)
	parseErrors = append(parseErrors, program.Errors...)
	if len(parseErrors) > 0 {
		return nil, errors.New(strings.Join(parseErrors, "\n"))
	}
	result := evaluator.Eval(program, env)
	if promise, ok := result.(*object.Promise); ok {
		result = promise.Wait()
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

func callRuntimeFunction(fn object.Object, env *object.Environment, args []object.Object, pos ast.Position) object.Object {
	call := &ast.CallExpr{
		Pos_:     pos,
		TokenLit: "__runtime_call",
		Callee:   &ast.Ident{Pos_: pos, TokenLit: "__runtime_call"},
		Args:     runtimeCallArgs(pos, args),
	}
	scope := env.NewScope()
	scope.Set("__runtime_call", fn)
	for i, arg := range args {
		scope.Set(fmt.Sprintf("__runtime_arg_%d", i), arg)
	}
	return evaluator.Eval(call, scope)
}

func runtimeCallArgs(pos ast.Position, args []object.Object) []ast.Expression {
	exprs := make([]ast.Expression, len(args))
	for i := range args {
		exprs[i] = &ast.Ident{Pos_: pos, TokenLit: fmt.Sprintf("__runtime_arg_%d", i)}
	}
	return exprs
}

func callRuntimeMain(env *object.Environment, file string) object.Object {
	mainFn, ok := env.Get("main")
	if !ok {
		return object.UNDEFINED
	}
	pos := ast.Position{File: file}
	return callRuntimeFunction(mainFn, env, nil, pos)
}

func readRuntimeResolvedSource(resolved module.ResolvedModule) (string, error) {
	if resolved.PackageFile != "" {
		return packagefile.ReadNestedText(resolved.PackageFile, resolved.ArchivePath)
	}
	data, err := os.ReadFile(resolved.Path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func runtimeResolvedFile(resolved module.ResolvedModule) string {
	if resolved.PackageFile != "" {
		return filepath.ToSlash(resolved.PackageFile) + "!" + filepath.ToSlash(resolved.ArchivePath)
	}
	return resolved.Path
}

func runtimeResolvedModuleDir(resolved module.ResolvedModule) string {
	if resolved.PackageFile != "" {
		return filepath.ToSlash(resolved.PackageFile) + "!" + filepath.ToSlash(filepath.Dir(resolved.ArchivePath))
	}
	return filepath.Dir(resolved.Path)
}

func runtimeExecutableArgv0() string {
	exe, err := os.Executable()
	if err == nil && exe != "" {
		return exe
	}
	return ""
}

func runtimeEnterWorkingDir(dir string) (func(), error) {
	if dir == "" {
		return func() {}, nil
	}
	oldWd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(dir); err != nil {
		return nil, err
	}
	return func() {
		_ = os.Chdir(oldWd)
	}, nil
}
