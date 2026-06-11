package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/async"
	"github.com/issueye/goscript/internal/bundle"
	"github.com/issueye/goscript/internal/config"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/gtp/pluginhost"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/packagefile"
	"github.com/issueye/goscript/internal/parser"
	"github.com/issueye/goscript/internal/proj"
	"github.com/issueye/goscript/internal/stdlib"
)

const version = "0.1.0-dev"
const defaultTimeout = 10 * time.Second

var sharedVMPool = object.NewVirtualMachinePool(runtime.NumCPU())
var cliInput io.Reader = os.Stdin
var hasAppendedPackage = currentExecutableHasAppendedPackage

type options struct {
	checkTypes bool
	workers    int
	timeout    time.Duration
}

type runner struct {
	opts     options
	pool     *async.Pool
	cache    *module.Cache
	vm       *object.VirtualMachine
	resolver *module.Resolver
	plugins  *pluginhost.Host
	rootDir  string
	loading  map[string]bool // 正在加载的模块路径，防止循环依赖
}

type replConfig struct {
	in        io.Reader
	out       io.Writer
	errOut    io.Writer
	showIntro bool
}

type runOptions struct {
	autoMain   bool
	workingDir string
	argv       []string
}

func main() {
	defer func() {
		if async.RecoverPanic("gs main") {
			os.Exit(1)
		}
	}()
	code := run(os.Args[1:])
	os.Exit(code)
}

func run(args []string) int {
	embedded := hasAppendedPackage()
	if embedded && len(args) > 0 && args[0] == "run-script" {
		return runEmbeddedScriptCommand(args[1:])
	}
	embeddedArgs := splitEmbeddedAppArgs(args)
	if embeddedArgs != nil {
		return runWithEmbeddedArgs(args[:embeddedArgs.separator], embeddedArgs.app)
	}
	if embedded {
		embeddedArgs = splitDirectEmbeddedAppArgs(args)
		if embeddedArgs != nil {
			return runWithEmbeddedArgs(args[:embeddedArgs.separator], embeddedArgs.app)
		}
	}

	fs := flag.NewFlagSet("gs", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	opts := options{}
	fs.BoolVar(&opts.checkTypes, "check-types", false, "enable optional type checking")
	fs.IntVar(&opts.workers, "workers", runtime.NumCPU(), "maximum async worker count")
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "maximum script runtime; use 0 to disable")
	showVersion := fs.Bool("version", false, "print version")
	apiDoc := fs.String("api_doc", "", "print native module API docs, e.g. @std/web; use all to list modules")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	timeoutExplicit := flagWasSet(fs, "timeout")
	if *showVersion {
		fmt.Fprintln(os.Stdout, "GoScript", version)
		return 0
	}
	if *apiDoc != "" {
		if err := printAPIDoc(*apiDoc); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	r := newRunner(opts)
	rest := fs.Args()
	if len(rest) == 0 {
		originalTimeout := r.opts.timeout
		if !timeoutExplicit {
			r.opts.timeout = 0
		}
		if err := r.runEmbeddedExecutable(); err == nil {
			return 0
		} else if !errors.Is(err, packagefile.ErrNoAppendedPackage) {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		r.opts.timeout = originalTimeout
		if err := r.runREPL(replConfig{in: cliInput, out: os.Stdout, errOut: os.Stderr, showIntro: true}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	var err error
	switch rest[0] {
	case "init":
		err = initCommand(rest[1:])
	case "run":
		err = r.runProject(".", scriptArgs(rest[1:])...)
	case "run-script":
		err = r.runScriptCommand(rest[1:])
	case "pack":
		err = packCommand(rest[1:])
	case "dist":
		err = distCommand(rest[1:])
	case "bundle":
		err = bundleCommand(rest[1:])
	case "help", "-h", "--help":
		printUsage(fs)
		return 0
	default:
		err = r.runArg(rest)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

type embeddedAppArgs struct {
	separator int
	app       []string
}

func splitEmbeddedAppArgs(args []string) *embeddedAppArgs {
	sep := -1
	for i, arg := range args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep < 0 || !hasAppendedPackage() {
		return nil
	}
	return &embeddedAppArgs{
		separator: sep,
		app:       append([]string{}, args[sep+1:]...),
	}
}

func splitDirectEmbeddedAppArgs(args []string) *embeddedAppArgs {
	for i, arg := range args {
		if arg == "--" {
			return nil
		}
		if isKnownCLIFlagValue(args, i) {
			continue
		}
		if isEmbeddedAppDirectArg(arg) {
			return &embeddedAppArgs{
				separator: i,
				app:       append([]string{}, args[i:]...),
			}
		}
	}
	return nil
}

func isEmbeddedAppDirectArg(arg string) bool {
	if arg == "" {
		return false
	}
	if arg == "--help" || arg == "-h" {
		return true
	}
	if strings.HasPrefix(arg, "-") {
		return !isKnownCLIFlag(arg)
	}
	return true
}

func isKnownCLIFlag(arg string) bool {
	name := strings.TrimPrefix(arg, "-")
	name = strings.TrimPrefix(name, "-")
	if eq := strings.Index(name, "="); eq >= 0 {
		name = name[:eq]
	}
	return name == "check-types" || name == "workers" || name == "timeout" || name == "version" || name == "api_doc" || name == "h" || name == "help"
}

func isKnownCLIFlagValue(args []string, index int) bool {
	if index == 0 {
		return false
	}
	prev := args[index-1]
	if strings.Contains(prev, "=") {
		return false
	}
	return prev == "--workers" || prev == "-workers" || prev == "--timeout" || prev == "-timeout" || prev == "--api_doc" || prev == "-api_doc"
}

func currentExecutableHasAppendedPackage() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	_, err = packagefile.ReadAppendedPackage(exe)
	return err == nil
}

func runWithEmbeddedArgs(cliArgs, appArgs []string) int {
	fs := flag.NewFlagSet("gs", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	opts := options{}
	fs.BoolVar(&opts.checkTypes, "check-types", false, "enable optional type checking")
	fs.IntVar(&opts.workers, "workers", runtime.NumCPU(), "maximum async worker count")
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "maximum script runtime; use 0 to disable")
	showVersion := fs.Bool("version", false, "print version")
	apiDoc := fs.String("api_doc", "", "print native module API docs, e.g. @std/web; use all to list modules")

	if err := fs.Parse(cliArgs); err != nil {
		return 2
	}
	timeoutExplicit := flagWasSet(fs, "timeout")
	if *showVersion {
		fmt.Fprintln(os.Stdout, "GoScript", version)
		return 0
	}
	if *apiDoc != "" {
		if err := printAPIDoc(*apiDoc); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintln(os.Stderr, "embedded executable accepts app arguments after --")
		return 2
	}
	if !timeoutExplicit {
		opts.timeout = 0
	}

	r := newRunner(opts)
	if err := r.runEmbeddedExecutable(appArgs...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runEmbeddedScriptCommand(args []string) int {
	fs := flag.NewFlagSet("gs", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	opts := options{}
	fs.BoolVar(&opts.checkTypes, "check-types", false, "enable optional type checking")
	fs.IntVar(&opts.workers, "workers", runtime.NumCPU(), "maximum async worker count")
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "maximum script runtime; use 0 to disable")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := newRunner(opts)
	if err := r.runScriptCommand(fs.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	wasSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			wasSet = true
		}
	})
	return wasSet
}

func newRunner(opts options) *runner {
	if opts.workers < 1 {
		opts.workers = 1
	}
	return &runner{
		opts:    opts,
		loading: make(map[string]bool),
	}
}

func printUsage(fs *flag.FlagSet) {
	fmt.Fprintf(fs.Output(), "Usage:\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] <script.gs>\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] <code>\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] init [dir]\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] run\n\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] run-script <script.gs> [args...]\n\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] pack [dir] [out.gspkg]\n\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] dist [dir] [out]\n\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] bundle <entry.gs> [out.gs]\n\n")
	fmt.Fprintf(fs.Output(), "Flags:\n")
	fs.PrintDefaults()
}

func printAPIDoc(path string) error {
	if path == "all" || path == "list" {
		fmt.Fprintln(os.Stdout, "Native modules:")
		for _, p := range module.ListNative() {
			fmt.Fprintf(os.Stdout, "  %s\n", p)
		}
		return nil
	}
	env := object.NewEnvironment()
	evaluator.RegisterBuiltins(env)
	obj, ok := module.GetNative(path, env)
	if !ok {
		return fmt.Errorf("native module %s is not registered", path)
	}
	fmt.Fprintf(os.Stdout, "%s\n", path)
	entries, ok := module.GetNativeAPIDoc(path)
	if !ok {
		entries = apiDocEntries(obj, "")
	}
	for _, entry := range entries {
		fmt.Fprintf(os.Stdout, "  %s\n", entry)
	}
	return nil
}

func apiDocEntries(obj object.Object, prefix string) []string {
	hash, ok := obj.(*object.Hash)
	if !ok {
		return nil
	}
	entries := make([]string, 0, len(hash.Pairs))
	for _, pair := range hash.Pairs {
		name := pair.Key.Inspect()
		if name == "" || strings.HasPrefix(name, "__") {
			continue
		}
		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}
		switch value := pair.Value.(type) {
		case *object.Builtin, *object.Function:
			entries = append(entries, fullName+"()")
		case *object.Hash:
			entries = append(entries, apiDocEntries(value, fullName)...)
		default:
			entries = append(entries, fullName)
		}
	}
	sort.Strings(entries)
	return entries
}

func initCommand(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	if len(args) > 1 {
		return fmt.Errorf("init expects at most 1 argument: [dir]")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return err
	}
	name := filepath.Base(filepath.Clean(absDir))
	if name == "." || name == string(filepath.Separator) {
		name = "goscript-app"
	}
	files := map[string]string{
		"project.toml": initProjectTemplate(name),
		"main.gs":      initMainTemplate,
	}
	for rel, contents := range files {
		path := filepath.Join(absDir, rel)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists", path)
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
			return err
		}
	}
	fmt.Fprintln(os.Stdout, absDir)
	return nil
}

func initProjectTemplate(name string) string {
	return fmt.Sprintf(`[project]
name = %q
version = "0.1.0"
entry = "main.gs"
`, name)
}

const initMainTemplate = `function main() {
  println("Hello, GoScript!");
}
`

func bundleCommand(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("bundle expects: <entry.gs> [out.gs]")
	}
	entry, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	out, err := bundle.Bundle(entry)
	if err != nil {
		return err
	}
	if len(args) == 1 {
		fmt.Fprint(os.Stdout, out)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(args[1]), 0755); err != nil {
		return err
	}
	return os.WriteFile(args[1], []byte(out), 0644)
}

func packCommand(args []string) error {
	dir := "."
	out := ""
	if len(args) > 0 {
		dir = args[0]
	}
	if len(args) > 1 {
		out = args[1]
	}
	if len(args) > 2 {
		return fmt.Errorf("pack expects at most 2 arguments: [dir] [out.gspkg]")
	}
	if err := packagefile.PackDirectory(dir, out); err != nil {
		return err
	}
	if out == "" {
		out = filepath.Base(filepath.Clean(dir)) + packagefile.Extension
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, absOut)
	return nil
}

func distCommand(args []string) error {
	dir := "."
	out := ""
	if len(args) > 0 {
		dir = args[0]
	}
	if len(args) > 1 {
		out = args[1]
	}
	if len(args) > 2 {
		return fmt.Errorf("dist expects at most 2 arguments: [dir] [out]")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if out == "" {
		name := filepath.Base(filepath.Clean(absDir))
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		out = filepath.Join(absDir, "dist", name)
	}
	tmpDir, err := os.MkdirTemp("", "goscript-dist-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	pkgPath := filepath.Join(tmpDir, "app"+packagefile.Extension)
	if err := packagefile.PackDirectory(absDir, pkgPath); err != nil {
		return err
	}
	stub, err := os.Executable()
	if err != nil {
		return err
	}
	if err := packagefile.AppendPackageToExecutable(stub, pkgPath, out); err != nil {
		return err
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, absOut)
	return nil
}

func (r *runner) runProject(dir string, args ...string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	cfg, err := proj.LoadStrict(filepath.Join(absDir, "project.toml"))
	if err != nil {
		return err
	}
	runCfg, err := config.LoadStrict(filepath.Join(absDir, "config.toml"))
	if err != nil {
		return err
	}
	r.plugins = pluginhost.New(absDir)
	if err := r.plugins.StartConfigured(runCfg.Plugins); err != nil {
		return err
	}
	defer func() {
		r.plugins.Close()
		r.plugins = nil
	}()
	entry := filepath.Join(absDir, cfg.Entry)
	return r.runFileWithOptions(entry, runOptions{autoMain: true, workingDir: absDir, argv: scriptArgv(entry, args)})
}

func (r *runner) runScriptCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("run-script expects: <script.gs> [args...]")
	}
	script := args[0]
	return r.runFileWithOptions(script, runOptions{
		autoMain: true,
		argv:     scriptArgv(script, scriptArgs(args[1:])),
	})
}

func (r *runner) runFile(path string, args []string) error {
	autoMain := strings.EqualFold(filepath.Base(path), "main.gs")
	return r.runFileWithOptions(path, runOptions{autoMain: autoMain, argv: scriptArgv(path, args)})
}

func (r *runner) runArg(args []string) error {
	if len(args) == 0 {
		return nil
	}
	if _, err := os.Stat(args[0]); err == nil {
		return r.runFile(args[0], scriptArgs(args[1:]))
	} else if !os.IsNotExist(err) && !isInvalidPathError(err) {
		return err
	}
	return r.runInline(strings.Join(args, " "))
}

func scriptArgs(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}
	return args
}

func isInvalidPathError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "filename, directory name, or volume label syntax is incorrect")
}

func (r *runner) runInline(src string) error {
	return r.withTimeout("script execution", func() error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		reuseVM := false
		defer func() {
			if reuseVM {
				r.releaseVM()
			} else {
				r.discardVM()
			}
		}()
		r.checkoutVM()
		r.pool = async.NewPool(r.opts.workers)
		r.vm.SetSpawner(r.pool.Go)
		r.cache = module.NewCacheWithVM(r.vm)
		r.rootDir = module.FindProjectRoot(cwd)
		r.resolver = module.NewResolver(r.rootDir)
		env := r.vm.NewEnvironment()
		module.SetupExports(env)
		r.configureModuleLoaders(env, cwd)
		if _, err := r.evalSource(src, "<inline>", env); err != nil {
			return err
		}
		if err := r.drainRuntime(); err != nil {
			return err
		}
		reuseVM = true
		return nil
	})
}

func (r *runner) runEmbeddedExecutable(args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	data, err := packagefile.ReadAppendedPackage(exe)
	if err != nil {
		return err
	}
	pkg, err := packagefile.OpenBytes(exe, data)
	if err != nil {
		return err
	}
	defer pkg.Close()

	entry := pkg.Manifest.Entry
	if pkg.Manifest.Package.Main != "" {
		entry = pkg.Manifest.Package.Main
	}
	if entry == "" {
		entry = "main.gs"
	}
	if args == nil {
		args = os.Args[1:]
	}
	return r.runPackageEntryFromExecutable(pkg, exe, entry, args...)
}

func (r *runner) runFileWithOptions(path string, opts runOptions) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if len(opts.argv) == 0 {
		opts.argv = scriptArgv(absPath, nil)
	}
	return r.withTimeout("script execution", func() error {
		restore, err := enterWorkingDir(opts.workingDir)
		if err != nil {
			return err
		}
		defer restore()

		reuseVM := false
		defer func() {
			if reuseVM {
				r.releaseVM()
			} else {
				r.discardVM()
			}
		}()
		if _, err := r.evalFile(absPath, opts); err != nil {
			return err
		}
		if err := r.drainRuntime(); err != nil {
			return err
		}
		reuseVM = true
		return nil
	})
}

func scriptArgv(path string, args []string) []string {
	argv := []string{executableArgv0(), path}
	argv = append(argv, args...)
	return argv
}

func executableArgv0() string {
	exe, err := os.Executable()
	if err == nil && exe != "" {
		return exe
	}
	if len(os.Args) > 0 {
		return os.Args[0]
	}
	return ""
}

func enterWorkingDir(dir string) (func(), error) {
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

func (r *runner) evalFile(absPath string, opts runOptions) (object.Object, error) {
	src, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	r.checkoutVM()
	r.vm.SetArgv(opts.argv)
	r.pool = async.NewPool(r.opts.workers)
	r.vm.SetSpawner(r.pool.Go)
	r.cache = module.NewCacheWithVM(r.vm)
	r.rootDir = module.FindProjectRoot(filepath.Dir(absPath))
	r.resolver = module.NewResolver(r.rootDir)
	env := r.vm.NewEnvironment()
	module.SetupExports(env)
	r.configureModuleLoaders(env, filepath.Dir(absPath))
	result, err := r.evalSource(string(src), absPath, env)
	if err != nil {
		return nil, err
	}
	if opts.autoMain {
		return r.callMain(env, absPath)
	}
	return result, nil
}

func (r *runner) runPackageEntryFromExecutable(pkg *packagefile.Package, executablePath, entry string, args ...string) error {
	entry = filepath.ToSlash(entry)
	src, err := pkg.ReadText(entry)
	if err != nil {
		return err
	}
	return r.withTimeout("script execution", func() error {
		absExe, err := filepath.Abs(executablePath)
		if err != nil {
			return err
		}
		archivePath := filepath.ToSlash(absExe) + "!" + entry
		r.checkoutVM()
		r.vm.SetArgv(append([]string{absExe, entry}, args...))
		r.pool = async.NewPool(r.opts.workers)
		r.vm.SetSpawner(r.pool.Go)
		r.cache = module.NewCacheWithVM(r.vm)
		r.rootDir = absExe
		r.resolver = module.NewResolver(r.rootDir)
		env := r.vm.NewEnvironment()
		module.SetupExports(env)
		r.configureModuleLoaders(env, filepath.ToSlash(absExe)+"!"+filepath.ToSlash(filepath.Dir(entry)))
		reuseVM := false
		defer func() {
			if reuseVM {
				r.releaseVM()
			} else {
				r.discardVM()
			}
		}()
		if _, err := r.evalSource(src, archivePath, env); err != nil {
			return err
		}
		if _, err := r.callMain(env, archivePath); err != nil {
			return err
		}
		if err := r.drainRuntime(); err != nil {
			return err
		}
		reuseVM = true
		return nil
	})
}

func (r *runner) evalSource(src, file string, env *object.Environment) (object.Object, error) {
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
		var err error
		result, err = r.waitPromise(promise, "top-level promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

func (r *runner) callMain(env *object.Environment, file string) (object.Object, error) {
	mainFn, ok := env.Get("main")
	if !ok {
		return object.UNDEFINED, nil
	}
	if _, ok := mainFn.(*object.Function); !ok {
		return nil, fmt.Errorf("%s: top-level main is not a function", file)
	}

	pos := ast.Position{File: file}
	call := &ast.CallExpr{
		Pos_:     pos,
		TokenLit: "main",
		Callee:   &ast.Ident{Pos_: pos, TokenLit: "main"},
	}
	result := evaluator.Eval(call, env)
	if promise, ok := result.(*object.Promise); ok {
		var err error
		result, err = r.waitPromise(promise, "main promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

func (r *runner) requireFunc(baseDir string) evaluator.RequireFn {
	return func(path string) (object.Object, error) {
		r.ensureRuntime()
		resolved, err := r.resolver.Resolve(path, module.ResolveOptions{ProjectRoot: r.rootDir, BaseDir: baseDir})
		if err != nil {
			return nil, err
		}
		if resolved.Path == "" {
			return nil, fmt.Errorf("module %s resolved without a source path", path)
		}
		cacheKey := resolved.ID
		if cacheKey == "" {
			cacheKey = resolved.Path
		}
		if cached := r.cache.Get(cacheKey); cached != nil {
			return module.GetExports(cached), nil
		}

		// 检测循环依赖
		if r.loading[cacheKey] {
			return nil, fmt.Errorf("circular dependency detected: %s", path)
		}
		r.loading[cacheKey] = true
		defer delete(r.loading, cacheKey)

		env := r.cache.GetOrCreate(cacheKey)
		module.SetupExports(env)
		r.configureModuleLoaders(env, resolvedModuleDir(resolved))

		src, err := r.readResolvedSource(resolved)
		if err != nil {
			return nil, err
		}
		if _, err := r.evalSource(string(src), resolved.Path, env); err != nil {
			return nil, err
		}
		return module.GetExports(env), nil
	}
}

func (r *runner) ensureRuntime() {
	if r.vm == nil {
		r.checkoutVM()
	}
	r.vm.SetTypeCheck(r.opts.checkTypes)
	if r.pool == nil {
		r.pool = async.NewPool(r.opts.workers)
		r.vm.SetSpawner(r.pool.Go)
	}
	if r.cache == nil {
		r.cache = module.NewCacheWithVM(r.vm)
	}
	if r.resolver == nil {
		r.resolver = module.NewResolver("")
	}
	if r.rootDir == "" {
		r.rootDir = module.FindProjectRoot("")
	}
}

func (r *runner) checkoutVM() {
	r.vm = sharedVMPool.Get()
	r.vm.SetTypeCheck(r.opts.checkTypes)
}

func (r *runner) releaseVM() {
	if r.vm == nil {
		return
	}
	stdlib.StopTerminalSessionsForVM(r.vm)
	sharedVMPool.Put(r.vm)
	r.discardVM()
}

func (r *runner) discardVM() {
	if r.vm != nil {
		stdlib.StopTerminalSessionsForVM(r.vm)
	}
	r.vm = nil
	r.cache = nil
	r.resolver = nil
	r.rootDir = ""
	r.pool = nil
}

func (r *runner) drainRuntime() error {
	if r.vm != nil {
		if err := r.waitGroup("async tasks", r.vm.WaitAsync); err != nil {
			r.vm = nil
			return err
		}
	}
	if r.pool != nil {
		if err := r.waitGroup("worker pool", r.pool.Wait); err != nil {
			r.vm = nil
			r.pool = nil
			return err
		}
	}
	return nil
}

func (r *runner) configureModuleLoaders(env *object.Environment, baseDir string) {
	r.ensureRuntime()
	env.ModuleDir = baseDir
	require := func(path string) (object.Object, error) {
		return r.requireFrom(env, path)
	}
	evaluator.RegisterBuiltinsWithCache(env, require)
	env.VM().SetImportFunc(func(env *object.Environment, path string) (object.Object, error) {
		return r.requireFrom(env, path)
	})
}

func (r *runner) requireFrom(env *object.Environment, path string) (object.Object, error) {
	baseDir := env.ModuleDir
	if baseDir == "" {
		baseDir = r.rootDir
	}
	resolved, err := r.resolver.Resolve(path, module.ResolveOptions{ProjectRoot: r.rootDir, BaseDir: baseDir})
	if err != nil {
		return nil, err
	}
	if resolved.Kind == module.ModuleKindNative {
		if r.plugins != nil {
			if native, ok := r.plugins.NativeModule(path, env); ok {
				return native, nil
			}
		}
		native, ok := module.GetNative(path, env)
		if !ok {
			return nil, fmt.Errorf("native module %s is not registered", path)
		}
		return native, nil
	}

	// 检测循环依赖
	cacheKey := resolved.ID
	if cacheKey == "" {
		cacheKey = resolved.Path
	}
	if r.loading[cacheKey] {
		return nil, fmt.Errorf("circular dependency detected: %s", path)
	}

	return r.requireFunc(baseDir)(path)
}

func (r *runner) readResolvedSource(resolved module.ResolvedModule) (string, error) {
	if resolved.PackageFile != "" {
		return packagefile.ReadNestedText(resolved.PackageFile, resolved.ArchivePath)
	}
	data, err := os.ReadFile(resolved.Path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func resolvedModuleDir(resolved module.ResolvedModule) string {
	if resolved.PackageFile != "" {
		return filepath.ToSlash(resolved.PackageFile) + "!" + filepath.ToSlash(filepath.Dir(resolved.ArchivePath))
	}
	return filepath.Dir(resolved.Path)
}

func (r *runner) waitPromise(p *object.Promise, label string) (object.Object, error) {
	if r.opts.timeout <= 0 {
		return p.Wait(), nil
	}

	done := make(chan object.Object, 1)
	go func() {
		done <- p.Wait()
	}()
	select {
	case result := <-done:
		return result, nil
	case <-time.After(r.opts.timeout):
		return nil, fmt.Errorf("%s timed out after %s", label, r.opts.timeout)
	}
}

func (r *runner) waitGroup(label string, wait func()) error {
	if r.opts.timeout <= 0 {
		wait()
		return nil
	}

	done := make(chan struct{})
	go func() {
		wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(r.opts.timeout):
		return fmt.Errorf("%s timed out after %s", label, r.opts.timeout)
	}
}

func (r *runner) withTimeout(label string, fn func() error) error {
	if r.opts.timeout <= 0 {
		return fn()
	}

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()
	timer := time.NewTimer(r.opts.timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		return err
	case <-timer.C:
		return fmt.Errorf("%s timed out after %s", label, r.opts.timeout)
	}
}
