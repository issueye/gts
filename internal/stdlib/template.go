package stdlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"os"
	"strings"
	texttemplate "text/template"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/template", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initTemplateModule(exports)
		return exports, nil
	})
}

func initTemplateModule(exports *object.Hash) {
	setHashMember(exports, "render", &object.Builtin{Name: "template.render", Fn: templateRender})
	setHashMember(exports, "renderHTML", &object.Builtin{Name: "template.renderHTML", Fn: templateRenderHTML})
	setHashMember(exports, "renderFileSync", &object.Builtin{Name: "template.renderFileSync", Fn: templateRenderFileSync})
	setHashMember(exports, "escapeHTML", &object.Builtin{Name: "template.escapeHTML", Fn: templateEscapeHTML})
}

func templateRender(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	source, errObj := requiredString(pos, "template.render", args, 0, "source")
	if errObj != nil {
		return errObj
	}
	opts := templateOptionsFromArgs(args, 2)
	return executeTemplate(pos, "template.render", source, templateDataArg(args, 1), opts)
}

func templateRenderHTML(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	source, errObj := requiredString(pos, "template.renderHTML", args, 0, "source")
	if errObj != nil {
		return errObj
	}
	opts := templateOptionsFromArgs(args, 2)
	opts.html = true
	return executeTemplate(pos, "template.renderHTML", source, templateDataArg(args, 1), opts)
}

func templateRenderFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "template.renderFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return object.NewError(pos, "template.renderFileSync: %v", err)
	}
	opts := templateOptionsFromArgs(args, 2)
	return executeTemplate(pos, "template.renderFileSync", string(source), templateDataArg(args, 1), opts)
}

func templateEscapeHTML(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "template.escapeHTML", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: htmltemplate.HTMLEscapeString(value)}
}

type templateOptions struct {
	html       bool
	name       string
	missingKey string
}

func templateOptionsFromArgs(args []object.Object, index int) templateOptions {
	opts := templateOptions{name: "template", missingKey: "invalid"}
	if len(args) <= index {
		return opts
	}
	hash, ok := args[index].(*object.Hash)
	if !ok {
		return opts
	}
	if name, ok := hashString(hash, "name"); ok && name != "" {
		opts.name = name
	}
	if value, ok := hashValue(hash, "html"); ok {
		if b, ok := value.(*object.Boolean); ok {
			opts.html = b.Value
		}
	}
	if missingKey, ok := hashString(hash, "missingKey"); ok && missingKey != "" {
		opts.missingKey = missingKey
	}
	return opts
}

func templateDataArg(args []object.Object, index int) interface{} {
	if len(args) <= index || args[index] == object.UNDEFINED || args[index] == object.NULL {
		return nil
	}
	return objectToGoValue(args[index])
}

func executeTemplate(pos ast.Position, name, source string, data interface{}, opts templateOptions) object.Object {
	var buf bytes.Buffer
	if opts.html {
		tpl, err := htmltemplate.New(opts.name).Funcs(htmlTemplateFuncs()).Option("missingkey=" + opts.missingKey).Parse(source)
		if err != nil {
			return object.NewError(pos, "%s: %v", name, err)
		}
		if err := tpl.Execute(&buf, data); err != nil {
			return object.NewError(pos, "%s: %v", name, err)
		}
		return &object.String{Value: buf.String()}
	}
	tpl, err := texttemplate.New(opts.name).Funcs(textTemplateFuncs()).Option("missingkey=" + opts.missingKey).Parse(source)
	if err != nil {
		return object.NewError(pos, "%s: %v", name, err)
	}
	if err := tpl.Execute(&buf, data); err != nil {
		return object.NewError(pos, "%s: %v", name, err)
	}
	return &object.String{Value: buf.String()}
}

func textTemplateFuncs() texttemplate.FuncMap {
	return texttemplate.FuncMap{
		"json":  templateJSON,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"trim":  strings.TrimSpace,
		"join":  templateJoin,
	}
}

func htmlTemplateFuncs() htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"json":  templateJSON,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"trim":  strings.TrimSpace,
		"join":  templateJoin,
	}
}

func templateJSON(value interface{}) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func templateJoin(values interface{}, sep string) string {
	switch v := values.(type) {
	case []interface{}:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = objectToTemplateText(item)
		}
		return strings.Join(parts, sep)
	case []string:
		return strings.Join(v, sep)
	default:
		return objectToTemplateText(values)
	}
}

func objectToTemplateText(value interface{}) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	data, err := json.Marshal(value)
	if err == nil {
		return string(data)
	}
	return fmt.Sprint(value)
}
