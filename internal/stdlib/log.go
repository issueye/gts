package stdlib

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type stdLogger struct {
	mu        sync.Mutex
	file      *os.File
	level     int
	timestamp bool
	json      bool
	closed    bool
}

func init() {
	module.RegisterNative("@std/log", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initLogModule(exports)
		return exports, nil
	})
}

func initLogModule(exports *object.Hash) {
	setHashMember(exports, "createFileLogger", &object.Builtin{Name: "log.createFileLogger", Fn: logCreateFileLogger})
}

func logCreateFileLogger(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "log.createFileLogger", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	opts, errObj := logOptions(pos, "log.createFileLogger", args, 1)
	if errObj != nil {
		return errObj
	}
	flag := os.O_CREATE | os.O_WRONLY
	if opts.append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return object.NewError(pos, "log.createFileLogger: %v", err)
	}
	logger := &stdLogger{
		file:      file,
		level:     opts.level,
		timestamp: opts.timestamp,
		json:      opts.json,
	}
	return loggerObject(logger)
}

type logModuleOptions struct {
	append    bool
	level     int
	timestamp bool
	json      bool
}

func logOptions(pos ast.Position, name string, args []object.Object, index int) (logModuleOptions, *object.Error) {
	opts := logModuleOptions{append: true, level: logLevelInfo, timestamp: true}
	if len(args) <= index || args[index] == object.UNDEFINED || args[index] == object.NULL {
		return opts, nil
	}
	hash, ok := args[index].(*object.Hash)
	if !ok {
		return opts, object.NewError(pos, "%s: options must be an object", name)
	}
	if value, ok := hashValue(hash, "append"); ok {
		b, ok := value.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "%s: options.append must be a boolean", name)
		}
		opts.append = b.Value
	}
	if value, ok := hashValue(hash, "timestamp"); ok {
		b, ok := value.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "%s: options.timestamp must be a boolean", name)
		}
		opts.timestamp = b.Value
	}
	if value, ok := hashValue(hash, "json"); ok {
		b, ok := value.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "%s: options.json must be a boolean", name)
		}
		opts.json = b.Value
	}
	if value, ok := hashValue(hash, "level"); ok {
		text, ok := value.(*object.String)
		if !ok {
			return opts, object.NewError(pos, "%s: options.level must be a string", name)
		}
		level, ok := parseLogLevel(text.Value)
		if !ok {
			return opts, object.NewError(pos, "%s: unsupported level %q", name, text.Value)
		}
		opts.level = level
	}
	return opts, nil
}

func loggerObject(logger *stdLogger) *object.Hash {
	holder := &object.GoObject{Value: logger}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "debug", &object.Builtin{Name: "log.debug", Fn: logDebug, Extra: holder})
	setHashMember(obj, "info", &object.Builtin{Name: "log.info", Fn: logInfo, Extra: holder})
	setHashMember(obj, "warn", &object.Builtin{Name: "log.warn", Fn: logWarn, Extra: holder})
	setHashMember(obj, "error", &object.Builtin{Name: "log.error", Fn: logError, Extra: holder})
	setHashMember(obj, "log", &object.Builtin{Name: "log.log", Fn: logInfo, Extra: holder})
	setHashMember(obj, "close", &object.Builtin{Name: "log.close", Fn: logClose, Extra: holder})
	return obj
}

func logDebug(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return writeLog(env, pos, logLevelDebug, "debug", args)
}

func logInfo(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return writeLog(env, pos, logLevelInfo, "info", args)
}

func logWarn(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return writeLog(env, pos, logLevelWarn, "warn", args)
}

func logError(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return writeLog(env, pos, logLevelError, "error", args)
}

func writeLog(env *object.Environment, pos ast.Position, level int, label string, args []object.Object) object.Object {
	logger, errObj := boundLogger(pos, env, "log."+label)
	if errObj != nil {
		return errObj
	}
	if level < logger.level {
		return object.UNDEFINED
	}
	line, err := logger.format(label, logJoin(args))
	if err != nil {
		return object.NewError(pos, "log.%s: %v", label, err)
	}
	if err := logger.write(line); err != nil {
		return object.NewError(pos, "log.%s: %v", label, err)
	}
	return object.UNDEFINED
}

func logClose(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	logger, errObj := boundLogger(pos, env, "log.close")
	if errObj != nil {
		return errObj
	}
	logger.mu.Lock()
	defer logger.mu.Unlock()
	if logger.closed {
		return object.UNDEFINED
	}
	logger.closed = true
	if err := logger.file.Close(); err != nil {
		return object.NewError(pos, "log.close: %v", err)
	}
	return object.UNDEFINED
}

func boundLogger(pos ast.Position, env *object.Environment, name string) (*stdLogger, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing logger receiver", name)
	}
	logger, ok := goObj.Value.(*stdLogger)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid logger receiver", name)
	}
	return logger, nil
}

func (l *stdLogger) format(level, message string) (string, error) {
	if l.json {
		entry := map[string]string{
			"level":   level,
			"message": message,
		}
		if l.timestamp {
			entry["time"] = time.Now().Format(time.RFC3339)
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}
	if l.timestamp {
		return fmt.Sprintf("%s [%s] %s\n", time.Now().Format(time.RFC3339), strings.ToUpper(level), message), nil
	}
	return fmt.Sprintf("[%s] %s\n", strings.ToUpper(level), message), nil
}

func (l *stdLogger) write(line string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return os.ErrClosed
	}
	_, err := l.file.WriteString(line)
	return err
}

func logJoin(args []object.Object) string {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = arg.Inspect()
	}
	return strings.Join(parts, " ")
}

const (
	logLevelDebug = iota
	logLevelInfo
	logLevelWarn
	logLevelError
)

func parseLogLevel(level string) (int, bool) {
	switch strings.ToLower(level) {
	case "debug":
		return logLevelDebug, true
	case "info":
		return logLevelInfo, true
	case "warn", "warning":
		return logLevelWarn, true
	case "error":
		return logLevelError, true
	default:
		return 0, false
	}
}
