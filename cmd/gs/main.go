package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/async"
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/bundle"
	"github.com/issueye/goscript/internal/config"
	"github.com/issueye/goscript/internal/gtp/pluginhost"
	"github.com/issueye/goscript/internal/lsp"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/packagefile"
	"github.com/issueye/goscript/internal/proj"
	"github.com/issueye/goscript/internal/runtime" // reused loader (Session)
	stdruntime "runtime"
)

const version = "0.1.0-dev"
const defaultTimeout = 10 * time.Second

var cliInput io.Reader = os.Stdin
var hasAppendedPackage = currentExecutableHasAppendedPackage

type options struct {
	checkTypes bool
	workers    int
	timeout    time.Duration
}

// runner executes one CLI invocation. It delegates the VM / pool / cache /
// resolver / loader to a runtime.Session (shared with the SDK and @std/web
// isolated mode), keeping only CLI-specific concerns here: flag-driven
// options, embedded package handling, plugin host wiring, and the per-run
// timeout envelope. A fresh Session is built per run and closed on exit.
type runner struct {
	opts    options
	sess    *runtime.Session
	plugins *pluginhost.Host
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
	fs.IntVar(&opts.workers, "workers", stdruntime.NumCPU(), "maximum async worker count")
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
	case "agent":
		err = r.runFile("agent-cli.gs", scriptArgs(rest[1:]))
	case "lsp":
		err = lsp.NewServer(os.Stdin, os.Stdout, os.Stderr).Run()
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
	fs.IntVar(&opts.workers, "workers", stdruntime.NumCPU(), "maximum async worker count")
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
	fs.IntVar(&opts.workers, "workers", stdruntime.NumCPU(), "maximum async worker count")
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
		opts: opts,
	}
}

func printUsage(fs *flag.FlagSet) {
	fmt.Fprintf(fs.Output(), "Usage:\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] <script.gs>\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] <code>\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] init [dir]\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] run\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] agent\n\n")
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
	// Build a throwaway session just to host the builtins env that native
	// module factories expect. No script runs here.
	sess := runtime.NewSession(runtime.Options{})
	defer sess.Close()
	env := sess.NewEnvironment()
	sess.Configure(env, "")
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
	if err := validateProjectForDist(absDir); err != nil {
		return err
	}

	// 读取配置以检查插件
	cfg, _ := config.LoadStrict(filepath.Join(absDir, "config.toml"))

	if out == "" {
		name := filepath.Base(filepath.Clean(absDir))
		if stdruntime.GOOS == "windows" {
			name += ".exe"
		}
		out = filepath.Join(absDir, "dist", name)
	}

	// 创建 dist 目录
	distDir := filepath.Dir(out)
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return err
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

	// 复制插件二进制到 dist/plugins/
	if cfg != nil && len(cfg.Plugins) > 0 {
		if err := copyPluginsToDist(absDir, distDir, cfg.Plugins); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to copy plugins: %v\n", err)
		}
	}

	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, absOut)
	return nil
}

func copyPluginsToDist(projectDir, distDir string, plugins map[string]config.PluginConfig) error {
	pluginsDir := filepath.Join(distDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return err
	}

	copied := 0
	for name, cfg := range plugins {
		if cfg.Command == "" {
			continue
		}

		// 解析插件命令路径
		cmdPath := cfg.Command
		if !filepath.IsAbs(cmdPath) {
			cmdPath = filepath.Join(projectDir, cmdPath)
		}

		// 检查文件是否存在
		if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
			continue
		}

		// 复制插件二进制
		targetPath := filepath.Join(pluginsDir, filepath.Base(cmdPath))
		if err := copyFile(cmdPath, targetPath); err != nil {
			return fmt.Errorf("copy plugin %s: %w", name, err)
		}

		// 设置执行权限
		if stdruntime.GOOS != "windows" {
			if err := os.Chmod(targetPath, 0755); err != nil {
				return err
			}
		}

		copied++
	}

	if copied > 0 {
		fmt.Fprintf(os.Stdout, "Copied %d plugin(s) to %s\n", copied, pluginsDir)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func validateProjectForDist(absDir string) error {
	cfg, err := proj.LoadStrict(filepath.Join(absDir, "project.toml"))
	if err != nil {
		return err
	}
	entry := filepath.Join(absDir, cfg.Entry)
	if _, err := bundle.Bundle(entry); err != nil {
		return fmt.Errorf("dist preflight failed: %w", err)
	}
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
		r.beginSession(cwd)
		defer r.endSession()
		env := r.sess.NewEnvironment()
		r.sess.Configure(env, cwd)
		if _, err := r.sess.EvalSource(src, "<inline>", env); err != nil {
			return err
		}
		return r.sess.Drain()
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

		r.beginSession("")
		defer r.endSession()
		r.sess.VM().SetArgv(opts.argv)
		src, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		r.sess.SetBootstrapSource(string(src))
		env := r.sess.NewEnvironment()
		r.sess.Configure(env, filepath.Dir(absPath))
		if _, err := r.sess.EvalSource(string(src), absPath, env); err != nil {
			return err
		}
		if opts.autoMain {
			if _, err := r.callMain(env, absPath); err != nil {
				return err
			}
		}
		return r.sess.Drain()
	})
}

// beginSession builds a fresh runtime.Session for this run, rooted at rootDir
// (derived from cwd when empty). The plugin host, if any, is wired in via the
// Session's NativeResolver so plugin modules resolve ahead of the global
// native registry. Each CLI run gets its own Session; there is no cross-run VM
// reuse (the old sharedVMPool is gone — per-run isolation matches the new
// model and the cost is negligible for a CLI invocation).
func (r *runner) beginSession(rootDir string) {
	if rootDir == "" {
		rootDir = module.FindProjectRoot("")
	}
	r.sess = runtime.NewSession(runtime.Options{
		Workers:    r.opts.workers,
		Timeout:    0, // CLI governs the timeout envelope via withTimeout.
		CheckTypes: r.opts.checkTypes,
		RootDir:    rootDir,
		NativeResolver: func(env *object.Environment, specifier string) (object.Object, bool, error) {
			if r.plugins != nil {
				if native, ok := r.plugins.NativeModule(specifier, env); ok {
					return native, true, nil
				}
			}
			return nil, false, nil
		},
	})
}

// endSession drains async work and closes the session, ensuring terminal
// sessions and other VM-bound resources are torn down even on error.
func (r *runner) endSession() {
	if r.sess == nil {
		return
	}
	_ = r.sess.Drain()
	r.sess.Close()
	r.sess = nil
}

// callMain invokes a top-level main() if present. Kept on the runner (rather
// than the Session) because the CLI's main()-auto-call convention is a
// CLI-level concern; the Session exposes the evaluator primitives.
func (r *runner) callMain(env *object.Environment, file string) (object.Object, error) {
	mainFn, ok := env.Get("main")
	if !ok {
		return object.UNDEFINED, nil
	}
	if _, ok := mainFn.(*object.Function); !ok {
		return nil, fmt.Errorf("%s: top-level main is not a function", file)
	}
	pos := objectEvalPos(file)
	call := mainCallExpr(pos)
	result := r.sess.VM().EvalNode(call, env)
	if promise, ok := result.(*object.Promise); ok {
		var err error
		result, err = r.sess.WaitPromise(promise, "main promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
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
		r.beginSession(absExe)
		defer r.endSession()
		r.sess.VM().SetArgv(append([]string{absExe, entry}, args...))
		r.sess.SetBootstrapSource(src)
		env := r.sess.NewEnvironment()
		r.sess.Configure(env, filepath.ToSlash(absExe)+"!"+filepath.ToSlash(filepath.Dir(entry)))
		if _, err := r.sess.EvalSource(src, archivePath, env); err != nil {
			return err
		}
		if _, err := r.callMain(env, archivePath); err != nil {
			return err
		}
		return r.sess.Drain()
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

// objectEvalPos / mainCallExpr build the tiny AST for invoking top-level
// main(). They live here (not on the Session) because the auto-call-main
// convention is a CLI concern.
func objectEvalPos(file string) ast.Position { return ast.Position{File: file} }

func mainCallExpr(pos ast.Position) *ast.CallExpr {
	return &ast.CallExpr{
		Pos_:     pos,
		TokenLit: "main",
		Callee:   &ast.Ident{Pos_: pos, TokenLit: "main"},
	}
}

// evalFile loads and evaluates a single script file within a fresh session,
// returning the result. Exposed for benchmarks/tests that want to drive the
// runner directly without going through the full CLI command dispatch.
//
// It deliberately does NOT drain or close the session: the web framework tests
// start an HTTP server inside the script via app.listen(), which runs on the
// session's pool, and the test then probes the live server. Draining here would
// block forever waiting for that server to stop. Callers that need teardown
// (the normal CLI run paths) use runFileWithOptions, which wraps evalFile-style
// logic with begin/endSession. Leaked sessions in tests are reclaimed when the
// test process exits.
func (r *runner) evalFile(absPath string, opts runOptions) (object.Object, error) {
	src, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	r.beginSession("")
	if len(opts.argv) > 0 {
		r.sess.VM().SetArgv(opts.argv)
	}
	r.sess.SetBootstrapSource(string(src))
	env := r.sess.NewEnvironment()
	r.sess.Configure(env, filepath.Dir(absPath))
	result, err := r.sess.EvalSource(string(src), absPath, env)
	if err != nil {
		return nil, err
	}
	if opts.autoMain {
		return r.callMain(env, absPath)
	}
	return result, nil
}
