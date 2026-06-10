package stdlib

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type cliCommand struct {
	use     string
	name    string
	short   string
	long    string
	example string
	version string
	aliases []string

	run     object.Object
	preRun  object.Object
	postRun object.Object
	args    object.Object

	parent          *cliCommand
	children        []*cliCommand
	flags           *cliFlagSet
	persistentFlags *cliFlagSet
	helpFlag        *cliFlag
	versionFlag     *cliFlag
	self            *object.Hash
}

type cliFlagSet struct {
	command    *cliCommand
	persistent bool
	flags      []*cliFlag
	byName     map[string]*cliFlag
	byShort    map[string]*cliFlag
	self       *object.Hash
}

type cliFlag struct {
	name      string
	shorthand string
	usage     string
	kind      string
	def       object.Object
	value     object.Object
	changed   bool
}

type cliArgValidator struct {
	kind string
	min  int
	max  int
}

func init() {
	module.RegisterNative("@std/cli", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initCLIModule(exports)
		return exports, nil
	})
}

func initCLIModule(exports *object.Hash) {
	setHashMember(exports, "command", &object.Builtin{Name: "cli.command", Fn: cliCommandNew})
	setHashMember(exports, "root", &object.Builtin{Name: "cli.root", Fn: cliCommandNew})
	setHashMember(exports, "noArgs", &object.Builtin{Name: "cli.noArgs", Fn: cliNoArgs})
	setHashMember(exports, "arbitraryArgs", &object.Builtin{Name: "cli.arbitraryArgs", Fn: cliArbitraryArgs})
	setHashMember(exports, "exactArgs", &object.Builtin{Name: "cli.exactArgs", Fn: cliExactArgs})
	setHashMember(exports, "minArgs", &object.Builtin{Name: "cli.minArgs", Fn: cliMinArgs})
	setHashMember(exports, "maxArgs", &object.Builtin{Name: "cli.maxArgs", Fn: cliMaxArgs})
	setHashMember(exports, "rangeArgs", &object.Builtin{Name: "cli.rangeArgs", Fn: cliRangeArgs})
}

func cliCommandNew(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cmd := newCLICommand()
	if len(args) > 0 && args[0] != object.UNDEFINED && args[0] != object.NULL {
		opts, ok := args[0].(*object.Hash)
		if !ok {
			return object.NewError(pos, "cli.command: options must be an object")
		}
		if errObj := cmd.applyOptions(pos, opts); errObj != nil {
			return errObj
		}
	}
	return cmd.self
}

func newCLICommand() *cliCommand {
	cmd := &cliCommand{}
	cmd.flags = newCLIFlagSet(cmd, false)
	cmd.persistentFlags = newCLIFlagSet(cmd, true)
	cmd.helpFlag = &cliFlag{name: "help", shorthand: "h", usage: "show help", kind: "bool", def: object.FALSE, value: object.FALSE}
	cmd.versionFlag = &cliFlag{name: "version", shorthand: "v", usage: "show version", kind: "bool", def: object.FALSE, value: object.FALSE}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	cmd.self = obj
	extra := &object.GoObject{Value: cmd}
	setHashMember(obj, "__cliCommand", extra)
	setHashMember(obj, "addCommand", &object.Builtin{Name: "cli.Command.addCommand", Fn: cliCommandAddCommand, Extra: extra})
	setHashMember(obj, "command", &object.Builtin{Name: "cli.Command.command", Fn: cliCommandCommand, Extra: extra})
	setHashMember(obj, "flags", &object.Builtin{Name: "cli.Command.flags", Fn: cliCommandFlags, Extra: extra})
	setHashMember(obj, "persistentFlags", &object.Builtin{Name: "cli.Command.persistentFlags", Fn: cliCommandPersistentFlags, Extra: extra})
	setHashMember(obj, "execute", &object.Builtin{Name: "cli.Command.execute", Fn: cliCommandExecute, Extra: extra})
	setHashMember(obj, "usage", &object.Builtin{Name: "cli.Command.usage", Fn: cliCommandUsage, Extra: extra})
	setHashMember(obj, "help", &object.Builtin{Name: "cli.Command.help", Fn: cliCommandHelp, Extra: extra})
	setHashMember(obj, "commandPath", &object.Builtin{Name: "cli.Command.commandPath", Fn: cliCommandPathFn, Extra: extra})
	setHashMember(obj, "flag", &object.Builtin{Name: "cli.Command.flag", Fn: cliCommandFlag, Extra: extra})
	return cmd
}

func newCLIFlagSet(cmd *cliCommand, persistent bool) *cliFlagSet {
	set := &cliFlagSet{
		command:    cmd,
		persistent: persistent,
		byName:     make(map[string]*cliFlag),
		byShort:    make(map[string]*cliFlag),
	}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	set.self = obj
	extra := &object.GoObject{Value: set}
	setHashMember(obj, "__cliFlagSet", extra)
	setHashMember(obj, "string", &object.Builtin{Name: "cli.FlagSet.string", Fn: cliFlagSetString, Extra: extra})
	setHashMember(obj, "bool", &object.Builtin{Name: "cli.FlagSet.bool", Fn: cliFlagSetBool, Extra: extra})
	setHashMember(obj, "int", &object.Builtin{Name: "cli.FlagSet.int", Fn: cliFlagSetInt, Extra: extra})
	setHashMember(obj, "number", &object.Builtin{Name: "cli.FlagSet.number", Fn: cliFlagSetNumber, Extra: extra})
	setHashMember(obj, "get", &object.Builtin{Name: "cli.FlagSet.get", Fn: cliFlagSetGet, Extra: extra})
	setHashMember(obj, "changed", &object.Builtin{Name: "cli.FlagSet.changed", Fn: cliFlagSetChanged, Extra: extra})
	return set
}

func (cmd *cliCommand) applyOptions(pos ast.Position, opts *object.Hash) *object.Error {
	if value, ok := hashValue(opts, "use"); ok {
		s, ok := value.(*object.String)
		if !ok {
			return object.NewError(pos, "cli.command: use must be a string")
		}
		cmd.use = s.Value
	}
	if value, ok := hashValue(opts, "Use"); ok && cmd.use == "" {
		s, ok := value.(*object.String)
		if !ok {
			return object.NewError(pos, "cli.command: Use must be a string")
		}
		cmd.use = s.Value
	}
	cmd.name = cliCommandName(cmd.use)
	for _, field := range []struct {
		key string
		dst *string
	}{
		{"short", &cmd.short},
		{"Short", &cmd.short},
		{"long", &cmd.long},
		{"Long", &cmd.long},
		{"example", &cmd.example},
		{"Example", &cmd.example},
		{"version", &cmd.version},
		{"Version", &cmd.version},
	} {
		if value, ok := hashValue(opts, field.key); ok {
			s, ok := value.(*object.String)
			if !ok {
				return object.NewError(pos, "cli.command: %s must be a string", field.key)
			}
			*field.dst = s.Value
		}
	}
	for _, key := range []string{"aliases", "Aliases"} {
		if value, ok := hashValue(opts, key); ok {
			aliases, errObj := cliStringArray(pos, "cli.command", value, key)
			if errObj != nil {
				return errObj
			}
			cmd.aliases = aliases
		}
	}
	for _, field := range []struct {
		key string
		dst *object.Object
	}{
		{"run", &cmd.run},
		{"Run", &cmd.run},
		{"preRun", &cmd.preRun},
		{"PreRun", &cmd.preRun},
		{"postRun", &cmd.postRun},
		{"PostRun", &cmd.postRun},
		{"args", &cmd.args},
		{"Args", &cmd.args},
	} {
		if value, ok := hashValue(opts, field.key); ok {
			*field.dst = value
		}
	}
	return nil
}

func cliCommandAddCommand(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cmd, errObj := currentCLICommand(env, pos, "cli.Command.addCommand")
	if errObj != nil {
		return errObj
	}
	if len(args) == 0 {
		return object.NewError(pos, "cli.Command.addCommand requires command")
	}
	for _, arg := range args {
		child, errObj := cliCommandFromObject(pos, "cli.Command.addCommand", arg)
		if errObj != nil {
			return errObj
		}
		child.parent = cmd
		cmd.children = append(cmd.children, child)
	}
	return cmd.self
}

func cliCommandCommand(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cmd, errObj := currentCLICommand(env, pos, "cli.Command.command")
	if errObj != nil {
		return errObj
	}
	childObj := cliCommandNew(env, pos, args...)
	if object.IsRuntimeError(childObj) {
		return childObj
	}
	child, errObj := cliCommandFromObject(pos, "cli.Command.command", childObj)
	if errObj != nil {
		return errObj
	}
	child.parent = cmd
	cmd.children = append(cmd.children, child)
	return child.self
}

func cliCommandFlags(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cmd, errObj := currentCLICommand(env, pos, "cli.Command.flags")
	if errObj != nil {
		return errObj
	}
	return cmd.flags.self
}

func cliCommandPersistentFlags(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cmd, errObj := currentCLICommand(env, pos, "cli.Command.persistentFlags")
	if errObj != nil {
		return errObj
	}
	return cmd.persistentFlags.self
}

func cliCommandExecute(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cmd, errObj := currentCLICommand(env, pos, "cli.Command.execute")
	if errObj != nil {
		return errObj
	}
	argv := runtimeArgv(env)
	runArgs := []string{}
	if len(argv) > 2 {
		runArgs = append(runArgs, argv[2:]...)
	}
	if len(args) >= 1 && args[0] != object.UNDEFINED && args[0] != object.NULL {
		values, errObj := cliStringArray(pos, "cli.Command.execute", args[0], "args")
		if errObj != nil {
			return errObj
		}
		runArgs = values
	}
	result := cmd.execute(env, pos, runArgs)
	if object.IsRuntimeError(result) {
		return result
	}
	return result
}

func cliCommandUsage(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cmd, errObj := currentCLICommand(env, pos, "cli.Command.usage")
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: cmd.usage()}
}

func cliCommandHelp(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cmd, errObj := currentCLICommand(env, pos, "cli.Command.help")
	if errObj != nil {
		return errObj
	}
	text := cmd.usage()
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	fmt.Fprint(os.Stdout, text)
	return object.UNDEFINED
}

func cliCommandPathFn(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cmd, errObj := currentCLICommand(env, pos, "cli.Command.commandPath")
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: cmd.commandPath()}
}

func cliCommandFlag(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cmd, errObj := currentCLICommand(env, pos, "cli.Command.flag")
	if errObj != nil {
		return errObj
	}
	name, errObj := requiredString(pos, "cli.Command.flag", args, 0, "name")
	if errObj != nil {
		return errObj
	}
	if flag := cmd.lookupFlag(name); flag != nil {
		return flag.value
	}
	return object.UNDEFINED
}

func cliFlagSetString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return cliFlagSetAdd(env, pos, "cli.FlagSet.string", "string", args...)
}

func cliFlagSetBool(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return cliFlagSetAdd(env, pos, "cli.FlagSet.bool", "bool", args...)
}

func cliFlagSetInt(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return cliFlagSetAdd(env, pos, "cli.FlagSet.int", "int", args...)
}

func cliFlagSetNumber(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return cliFlagSetAdd(env, pos, "cli.FlagSet.number", "number", args...)
}

func cliFlagSetGet(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	set, errObj := currentCLIFlagSet(env, pos, "cli.FlagSet.get")
	if errObj != nil {
		return errObj
	}
	name, errObj := requiredString(pos, "cli.FlagSet.get", args, 0, "name")
	if errObj != nil {
		return errObj
	}
	if flag := set.byName[name]; flag != nil {
		return flag.value
	}
	return object.UNDEFINED
}

func cliFlagSetChanged(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	set, errObj := currentCLIFlagSet(env, pos, "cli.FlagSet.changed")
	if errObj != nil {
		return errObj
	}
	name, errObj := requiredString(pos, "cli.FlagSet.changed", args, 0, "name")
	if errObj != nil {
		return errObj
	}
	flag := set.byName[name]
	return object.NativeBool(flag != nil && flag.changed)
}

func cliFlagSetAdd(env *object.Environment, pos ast.Position, name, kind string, args ...object.Object) object.Object {
	set, errObj := currentCLIFlagSet(env, pos, name)
	if errObj != nil {
		return errObj
	}
	flagName, shorthand, def, usage, errObj := cliFlagDefinition(pos, name, kind, args)
	if errObj != nil {
		return errObj
	}
	if flagName == "" {
		return object.NewError(pos, "%s: flag name cannot be empty", name)
	}
	if _, exists := set.byName[flagName]; exists {
		return object.NewError(pos, "%s: flag %s is already defined", name, flagName)
	}
	if shorthand != "" {
		if _, exists := set.byShort[shorthand]; exists {
			return object.NewError(pos, "%s: shorthand %s is already defined", name, shorthand)
		}
	}
	coerced, errObj := cliCoerceFlagValue(pos, name, kind, def)
	if errObj != nil {
		return errObj
	}
	flag := &cliFlag{name: flagName, shorthand: shorthand, usage: usage, kind: kind, def: coerced, value: coerced}
	set.flags = append(set.flags, flag)
	set.byName[flagName] = flag
	if shorthand != "" {
		set.byShort[shorthand] = flag
	}
	return set.self
}

func cliFlagDefinition(pos ast.Position, fnName, kind string, args []object.Object) (string, string, object.Object, string, *object.Error) {
	if len(args) == 1 {
		opts, ok := args[0].(*object.Hash)
		if !ok {
			return "", "", nil, "", object.NewError(pos, "%s: options must be an object", fnName)
		}
		flagName, _ := cliHashString(opts, "name")
		shorthand, _ := cliHashString(opts, "shorthand")
		if shorthand == "" {
			shorthand, _ = cliHashString(opts, "short")
		}
		usage, _ := cliHashString(opts, "usage")
		def := cliDefaultForKind(kind)
		if value, ok := hashValue(opts, "default"); ok {
			def = value
		}
		return flagName, shorthand, def, usage, nil
	}
	flagName, errObj := requiredString(pos, fnName, args, 0, "name")
	if errObj != nil {
		return "", "", nil, "", errObj
	}
	shorthand := ""
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		s, ok := args[1].(*object.String)
		if !ok {
			return "", "", nil, "", object.NewError(pos, "%s: shorthand must be a string", fnName)
		}
		shorthand = s.Value
	}
	def := cliDefaultForKind(kind)
	if len(args) >= 3 {
		def = args[2]
	}
	usage := ""
	if len(args) >= 4 && args[3] != object.UNDEFINED && args[3] != object.NULL {
		s, ok := args[3].(*object.String)
		if !ok {
			return "", "", nil, "", object.NewError(pos, "%s: usage must be a string", fnName)
		}
		usage = s.Value
	}
	return flagName, shorthand, def, usage, nil
}

func (cmd *cliCommand) execute(env *object.Environment, pos ast.Position, argv []string) object.Object {
	target, tokens := cmd.resolve(argv)
	target.resetFlags()
	if errObj := target.parseFlags(pos, tokens); errObj != nil {
		return errObj
	}
	if target.hasFlagChanged("help") || target.hasFlagChanged("h") {
		fmt.Fprint(os.Stdout, target.usage())
		return &object.Number{Value: 0}
	}
	if target == cmd && cmd.version != "" && (target.hasFlagChanged("version") || target.hasFlagChanged("v")) {
		fmt.Fprintln(os.Stdout, cmd.version)
		return &object.Number{Value: 0}
	}
	positionals := target.positionals(tokens)
	if len(positionals) > 0 {
		if child := target.findChild(positionals[0]); child != nil {
			return object.NewError(pos, "cli: command %s was not resolved", child.name)
		}
	}
	if target.args != nil {
		result := cliCallArgValidator(env, pos, target.args, target.self, positionals)
		if object.IsRuntimeError(result) {
			return result
		}
	}
	if target.run == nil {
		fmt.Fprint(os.Stdout, target.usage())
		return &object.Number{Value: 0}
	}
	for _, hook := range []object.Object{target.preRun, target.run, target.postRun} {
		if hook == nil {
			continue
		}
		result := cliCallFunction(env, pos, hook, target.self, []object.Object{target.self, strSliceToArray(positionals)})
		if object.IsRuntimeError(result) {
			return result
		}
	}
	return &object.Number{Value: 0}
}

func (cmd *cliCommand) resolve(argv []string) (*cliCommand, []string) {
	current := cmd
	tokens := append([]string{}, argv...)
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token == "--" {
			break
		}
		if strings.HasPrefix(token, "-") && token != "-" {
			if flag := current.lookupFlag(strings.TrimLeft(strings.SplitN(token, "=", 2)[0], "-")); flag != nil && flag.kind != "bool" && !strings.Contains(token, "=") && i+1 < len(tokens) {
				i++
			}
			continue
		}
		child := current.findChild(token)
		if child == nil {
			break
		}
		current = child
		tokens = append(tokens[:i], tokens[i+1:]...)
		i--
	}
	return current, tokens
}

func (cmd *cliCommand) parseFlags(pos ast.Position, tokens []string) *object.Error {
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token == "--" {
			return nil
		}
		if token == "-" || !strings.HasPrefix(token, "-") {
			continue
		}
		if strings.HasPrefix(token, "--") {
			nameValue := strings.TrimPrefix(token, "--")
			if nameValue == "" {
				return nil
			}
			name, raw, hasValue := strings.Cut(nameValue, "=")
			negated := false
			if strings.HasPrefix(name, "no-") {
				if flag := cmd.lookupFlag(strings.TrimPrefix(name, "no-")); flag != nil && flag.kind == "bool" {
					name = strings.TrimPrefix(name, "no-")
					raw = "false"
					hasValue = true
					negated = true
				}
			}
			flag := cmd.lookupFlag(name)
			if flag == nil {
				return object.NewError(pos, "cli: unknown flag --%s", name)
			}
			if flag.kind == "bool" && !hasValue {
				raw = "true"
				hasValue = true
			}
			if !hasValue {
				if i+1 >= len(tokens) {
					return object.NewError(pos, "cli: flag --%s requires value", name)
				}
				i++
				raw = tokens[i]
			}
			if negated && raw != "false" {
				return object.NewError(pos, "cli: invalid negated flag --no-%s", name)
			}
			if errObj := flag.set(pos, raw); errObj != nil {
				return errObj
			}
			continue
		}
		shorts := strings.TrimPrefix(token, "-")
		if shorts == "" {
			continue
		}
		for j := 0; j < len(shorts); j++ {
			key := string(shorts[j])
			flag := cmd.lookupShortFlag(key)
			if flag == nil {
				return object.NewError(pos, "cli: unknown shorthand -%s", key)
			}
			raw := "true"
			if flag.kind != "bool" {
				if j+1 < len(shorts) {
					raw = shorts[j+1:]
					j = len(shorts)
				} else {
					if i+1 >= len(tokens) {
						return object.NewError(pos, "cli: flag -%s requires value", key)
					}
					i++
					raw = tokens[i]
				}
			}
			if errObj := flag.set(pos, raw); errObj != nil {
				return errObj
			}
		}
	}
	return nil
}

func (cmd *cliCommand) positionals(tokens []string) []string {
	out := []string{}
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token == "--" {
			out = append(out, tokens[i+1:]...)
			break
		}
		if token == "-" || !strings.HasPrefix(token, "-") {
			out = append(out, token)
			continue
		}
		if strings.HasPrefix(token, "--") {
			nameValue := strings.TrimPrefix(token, "--")
			name, _, hasValue := strings.Cut(nameValue, "=")
			if strings.HasPrefix(name, "no-") {
				if flag := cmd.lookupFlag(strings.TrimPrefix(name, "no-")); flag != nil && flag.kind == "bool" {
					continue
				}
			}
			flag := cmd.lookupFlag(name)
			if flag != nil && flag.kind != "bool" && !hasValue && i+1 < len(tokens) {
				i++
			}
			continue
		}
		shorts := strings.TrimPrefix(token, "-")
		for j := 0; j < len(shorts); j++ {
			flag := cmd.lookupShortFlag(string(shorts[j]))
			if flag != nil && flag.kind != "bool" {
				if j+1 >= len(shorts) && i+1 < len(tokens) {
					i++
				}
				break
			}
		}
	}
	return out
}

func (cmd *cliCommand) usage() string {
	var b strings.Builder
	title := cmd.commandPath()
	if title == "" {
		title = cmd.name
	}
	if cmd.short != "" {
		fmt.Fprintf(&b, "%s - %s\n\n", title, cmd.short)
	} else if title != "" {
		fmt.Fprintf(&b, "%s\n\n", title)
	}
	if cmd.long != "" {
		b.WriteString(cmd.long)
		b.WriteString("\n\n")
	}
	useLine := cmd.use
	if useLine == "" {
		useLine = cmd.name
	}
	if parent := cmd.parentPath(); parent != "" && !strings.HasPrefix(useLine, parent+" ") {
		useLine = parent + " " + useLine
	}
	if useLine != "" {
		b.WriteString("Usage:\n  ")
		b.WriteString(useLine)
		b.WriteString("\n\n")
	}
	if len(cmd.aliases) > 0 {
		b.WriteString("Aliases:\n  ")
		b.WriteString(strings.Join(cmd.aliases, ", "))
		b.WriteString("\n\n")
	}
	if len(cmd.children) > 0 {
		b.WriteString("Commands:\n")
		children := append([]*cliCommand(nil), cmd.children...)
		sort.Slice(children, func(i, j int) bool { return children[i].name < children[j].name })
		for _, child := range children {
			fmt.Fprintf(&b, "  %-14s %s\n", child.name, child.short)
		}
		b.WriteString("\n")
	}
	flags := cmd.visibleFlags()
	if len(flags) > 0 {
		b.WriteString("Flags:\n")
		for _, flag := range flags {
			short := "  "
			if flag.shorthand != "" {
				short = "-" + flag.shorthand + ", "
			}
			def := ""
			if flag.def != object.UNDEFINED {
				def = " (default " + flag.def.Inspect() + ")"
			}
			fmt.Fprintf(&b, "  %s--%-14s %s%s\n", short, flag.name, flag.usage, def)
		}
		b.WriteString("\n")
	}
	if cmd.example != "" {
		b.WriteString("Examples:\n")
		b.WriteString(cmd.example)
		if !strings.HasSuffix(cmd.example, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (cmd *cliCommand) commandPath() string {
	parts := []string{}
	for cur := cmd; cur != nil; cur = cur.parent {
		if cur.name != "" {
			parts = append([]string{cur.name}, parts...)
		}
	}
	return strings.Join(parts, " ")
}

func (cmd *cliCommand) parentPath() string {
	if cmd.parent == nil {
		return ""
	}
	return cmd.parent.commandPath()
}

func (cmd *cliCommand) visibleFlags() []*cliFlag {
	flags := []*cliFlag{cmd.helpFlag}
	if cmd.parent == nil && cmd.version != "" {
		flags = append(flags, cmd.versionFlag)
	}
	seen := map[string]bool{"help": true}
	for cur := cmd; cur != nil; cur = cur.parent {
		for _, flag := range cur.persistentFlags.flags {
			if !seen[flag.name] {
				flags = append(flags, flag)
				seen[flag.name] = true
			}
		}
	}
	for _, flag := range cmd.flags.flags {
		if !seen[flag.name] {
			flags = append(flags, flag)
			seen[flag.name] = true
		}
	}
	return flags
}

func (cmd *cliCommand) resetFlags() {
	cmd.helpFlag.value = cmd.helpFlag.def
	cmd.helpFlag.changed = false
	cmd.versionFlag.value = cmd.versionFlag.def
	cmd.versionFlag.changed = false
	for cur := cmd; cur != nil; cur = cur.parent {
		cur.flags.reset()
		cur.persistentFlags.reset()
	}
}

func (set *cliFlagSet) reset() {
	for _, flag := range set.flags {
		flag.value = flag.def
		flag.changed = false
	}
}

func (cmd *cliCommand) lookupFlag(name string) *cliFlag {
	if name == "help" || name == "h" {
		return cmd.helpFlag
	}
	if (name == "version" || name == "v") && cmd.root().version != "" {
		return cmd.root().versionFlag
	}
	if flag := cmd.flags.byName[name]; flag != nil {
		return flag
	}
	for cur := cmd; cur != nil; cur = cur.parent {
		if flag := cur.persistentFlags.byName[name]; flag != nil {
			return flag
		}
	}
	return nil
}

func (cmd *cliCommand) lookupShortFlag(shorthand string) *cliFlag {
	if shorthand == "h" {
		return cmd.helpFlag
	}
	if shorthand == "v" && cmd.root().version != "" {
		return cmd.root().versionFlag
	}
	if flag := cmd.flags.byShort[shorthand]; flag != nil {
		return flag
	}
	for cur := cmd; cur != nil; cur = cur.parent {
		if flag := cur.persistentFlags.byShort[shorthand]; flag != nil {
			return flag
		}
	}
	return nil
}

func (cmd *cliCommand) hasFlagChanged(name string) bool {
	for _, flag := range cmd.visibleFlags() {
		if flag.name == name || flag.shorthand == name {
			return flag.changed
		}
	}
	return false
}

func (cmd *cliCommand) findChild(name string) *cliCommand {
	for _, child := range cmd.children {
		if child.name == name {
			return child
		}
		for _, alias := range child.aliases {
			if alias == name {
				return child
			}
		}
	}
	return nil
}

func (cmd *cliCommand) root() *cliCommand {
	for cmd.parent != nil {
		cmd = cmd.parent
	}
	return cmd
}

func (flag *cliFlag) set(pos ast.Position, raw string) *object.Error {
	switch flag.kind {
	case "string":
		flag.value = &object.String{Value: raw}
	case "bool":
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return object.NewError(pos, "cli: flag --%s expects bool", flag.name)
		}
		flag.value = object.NativeBool(value)
	case "int":
		value, err := strconv.Atoi(raw)
		if err != nil {
			return object.NewError(pos, "cli: flag --%s expects int", flag.name)
		}
		flag.value = &object.Number{Value: float64(value)}
	case "number":
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return object.NewError(pos, "cli: flag --%s expects number", flag.name)
		}
		flag.value = &object.Number{Value: value}
	}
	flag.changed = true
	return nil
}

func cliCallFunction(env *object.Environment, pos ast.Position, fn object.Object, this object.Object, args []object.Object) object.Object {
	switch f := fn.(type) {
	case *object.Function:
		scope := f.Env.NewScope()
		scope.Set("this", this)
		for i, p := range f.Parameters {
			if i < len(args) {
				if p.Spread {
					rest := make([]object.Object, len(args)-i)
					copy(rest, args[i:])
					scope.Set(p.Name, f.Env.ObjectManager().NewArray(rest))
					break
				}
				scope.Set(p.Name, args[i])
			} else if p.Default != nil {
				scope.Set(p.Name, f.Env.VM().EvalNode(p.Default, f.Env))
			} else {
				scope.Set(p.Name, object.UNDEFINED)
			}
		}
		result := f.Env.VM().EvalNode(f.Body, scope)
		if rv, ok := result.(*object.ReturnValue); ok {
			return rv.Value
		}
		return result
	case *object.Builtin:
		env.Extra = f.Extra
		result := f.Fn(env, pos, args...)
		env.Extra = nil
		return result
	default:
		return object.NewError(pos, "cli: callback must be a function")
	}
}

func cliCallArgValidator(env *object.Environment, pos ast.Position, validator object.Object, cmd object.Object, args []string) object.Object {
	if goObj, ok := validator.(*object.GoObject); ok {
		if v, ok := goObj.Value.(cliArgValidator); ok {
			return v.validate(pos, args)
		}
	}
	return cliCallFunction(env, pos, validator, cmd, []object.Object{cmd, strSliceToArray(args)})
}

func (v cliArgValidator) validate(pos ast.Position, args []string) object.Object {
	count := len(args)
	switch v.kind {
	case "none":
		if count != 0 {
			return object.NewError(pos, "cli: accepts no arguments, got %d", count)
		}
	case "exact":
		if count != v.min {
			return object.NewError(pos, "cli: accepts %d argument(s), got %d", v.min, count)
		}
	case "min":
		if count < v.min {
			return object.NewError(pos, "cli: requires at least %d argument(s), got %d", v.min, count)
		}
	case "max":
		if count > v.max {
			return object.NewError(pos, "cli: accepts at most %d argument(s), got %d", v.max, count)
		}
	case "range":
		if count < v.min || count > v.max {
			return object.NewError(pos, "cli: accepts between %d and %d argument(s), got %d", v.min, v.max, count)
		}
	}
	return object.UNDEFINED
}

func cliNoArgs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.GoObject{Value: cliArgValidator{kind: "none"}}
}

func cliArbitraryArgs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.GoObject{Value: cliArgValidator{kind: "any"}}
}

func cliExactArgs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, errObj := cliRequiredInt(pos, "cli.exactArgs", args, 0, "n")
	if errObj != nil {
		return errObj
	}
	return &object.GoObject{Value: cliArgValidator{kind: "exact", min: n}}
}

func cliMinArgs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, errObj := cliRequiredInt(pos, "cli.minArgs", args, 0, "n")
	if errObj != nil {
		return errObj
	}
	return &object.GoObject{Value: cliArgValidator{kind: "min", min: n}}
}

func cliMaxArgs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, errObj := cliRequiredInt(pos, "cli.maxArgs", args, 0, "n")
	if errObj != nil {
		return errObj
	}
	return &object.GoObject{Value: cliArgValidator{kind: "max", max: n}}
}

func cliRangeArgs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	min, errObj := cliRequiredInt(pos, "cli.rangeArgs", args, 0, "min")
	if errObj != nil {
		return errObj
	}
	max, errObj := cliRequiredInt(pos, "cli.rangeArgs", args, 1, "max")
	if errObj != nil {
		return errObj
	}
	if max < min {
		return object.NewError(pos, "cli.rangeArgs: max must be >= min")
	}
	return &object.GoObject{Value: cliArgValidator{kind: "range", min: min, max: max}}
}

func currentCLICommand(env *object.Environment, pos ast.Position, name string) (*cliCommand, *object.Error) {
	extra, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid command receiver", name)
	}
	cmd, ok := extra.Value.(*cliCommand)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid command receiver", name)
	}
	return cmd, nil
}

func currentCLIFlagSet(env *object.Environment, pos ast.Position, name string) (*cliFlagSet, *object.Error) {
	extra, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid flag set receiver", name)
	}
	set, ok := extra.Value.(*cliFlagSet)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid flag set receiver", name)
	}
	return set, nil
}

func cliCommandFromObject(pos ast.Position, name string, value object.Object) (*cliCommand, *object.Error) {
	hash, ok := value.(*object.Hash)
	if !ok {
		return nil, object.NewError(pos, "%s: command must be a cli command object", name)
	}
	extra, ok := hashValue(hash, "__cliCommand")
	if !ok {
		return nil, object.NewError(pos, "%s: command must be a cli command object", name)
	}
	goObj, ok := extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid command object", name)
	}
	cmd, ok := goObj.Value.(*cliCommand)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid command object", name)
	}
	return cmd, nil
}

func cliStringArray(pos ast.Position, name string, value object.Object, label string) ([]string, *object.Error) {
	arr, ok := value.(*object.Array)
	if !ok {
		return nil, object.NewError(pos, "%s: %s must be an array of strings", name, label)
	}
	out := make([]string, len(arr.Elements))
	for i, elem := range arr.Elements {
		s, ok := elem.(*object.String)
		if !ok {
			return nil, object.NewError(pos, "%s: %s[%d] must be a string", name, label, i)
		}
		out[i] = s.Value
	}
	return out, nil
}

func cliHashString(hash *object.Hash, key string) (string, bool) {
	value, ok := hashValue(hash, key)
	if !ok {
		return "", false
	}
	s, ok := value.(*object.String)
	if !ok {
		return "", false
	}
	return s.Value, true
}

func cliCommandName(use string) string {
	fields := strings.Fields(use)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func cliDefaultForKind(kind string) object.Object {
	switch kind {
	case "string":
		return &object.String{Value: ""}
	case "bool":
		return object.FALSE
	case "int", "number":
		return &object.Number{Value: 0}
	default:
		return object.UNDEFINED
	}
}

func cliCoerceFlagValue(pos ast.Position, name, kind string, value object.Object) (object.Object, *object.Error) {
	switch kind {
	case "string":
		if s, ok := value.(*object.String); ok {
			return s, nil
		}
		return nil, object.NewError(pos, "%s: default must be a string", name)
	case "bool":
		if b, ok := value.(*object.Boolean); ok {
			return b, nil
		}
		return nil, object.NewError(pos, "%s: default must be a bool", name)
	case "int", "number":
		if n, ok := value.(*object.Number); ok {
			return n, nil
		}
		return nil, object.NewError(pos, "%s: default must be a number", name)
	default:
		return value, nil
	}
}

func cliRequiredInt(pos ast.Position, name string, args []object.Object, index int, label string) (int, *object.Error) {
	if len(args) <= index {
		return 0, object.NewError(pos, "%s requires %s", name, label)
	}
	n, ok := args[index].(*object.Number)
	if !ok {
		return 0, object.NewError(pos, "%s: %s must be a number", name, label)
	}
	return int(n.Value), nil
}
