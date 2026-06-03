package stdlib

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type xmlNode struct {
	name      string
	attrs     map[string]string
	children  []*xmlNode
	textParts []string
	parent    *xmlNode
}

func init() {
	module.RegisterNative("@std/xml", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initXMLModule(exports)
		return exports, nil
	})
}

func initXMLModule(exports *object.Hash) {
	setHashMember(exports, "parse", &object.Builtin{Name: "xml.parse", Fn: xmlParse})
	setHashMember(exports, "stringify", &object.Builtin{Name: "xml.stringify", Fn: xmlStringify})
	setHashMember(exports, "readFileSync", &object.Builtin{Name: "xml.readFileSync", Fn: xmlReadFileSync})
	setHashMember(exports, "writeFileSync", &object.Builtin{Name: "xml.writeFileSync", Fn: xmlWriteFileSync})
}

func xmlParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, "xml.parse", args, 0, "text")
	if errObj != nil {
		return errObj
	}
	root, err := parseXMLDocument(text)
	if err != nil {
		return object.NewError(pos, "xml.parse: %v", err)
	}
	return xmlNodeToObject(root)
}

func xmlStringify(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "xml.stringify requires a node")
	}
	node, err := objectToXMLNode(args[0])
	if err != nil {
		return object.NewError(pos, "xml.stringify: %v", err)
	}
	var out bytes.Buffer
	if err := writeXMLNode(&out, node); err != nil {
		return object.NewError(pos, "xml.stringify: %v", err)
	}
	return &object.String{Value: out.String()}
}

func xmlReadFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "xml.readFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return object.NewError(pos, "xml.readFileSync: %v", err)
	}
	return xmlParse(env, pos, &object.String{Value: string(data)})
}

func xmlWriteFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "xml.writeFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return object.NewError(pos, "xml.writeFileSync requires node")
	}
	encoded := xmlStringify(env, pos, args[1])
	if err, ok := encoded.(*object.Error); ok {
		return err
	}
	text, ok := encoded.(*object.String)
	if !ok {
		return object.NewError(pos, "xml.writeFileSync: stringify did not return text")
	}
	if err := os.WriteFile(path, []byte(text.Value), 0644); err != nil {
		return object.NewError(pos, "xml.writeFileSync: %v", err)
	}
	return object.UNDEFINED
}

func parseXMLDocument(text string) (*xmlNode, error) {
	decoder := xml.NewDecoder(strings.NewReader(text))
	var root *xmlNode
	var current *xmlNode
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			node := &xmlNode{
				name:  t.Name.Local,
				attrs: make(map[string]string),
			}
			for _, attr := range t.Attr {
				node.attrs[attr.Name.Local] = attr.Value
			}
			if current != nil {
				node.parent = current
				current.children = append(current.children, node)
			} else {
				root = node
			}
			current = node
		case xml.CharData:
			if current != nil {
				text := string([]byte(t))
				if strings.TrimSpace(text) != "" {
					current.textParts = append(current.textParts, text)
				}
			}
		case xml.EndElement:
			if current != nil {
				current = current.parent
			}
		}
	}
	if root == nil {
		return nil, &xml.SyntaxError{Msg: "empty XML document"}
	}
	return root, nil
}

func xmlNodeToObject(node *xmlNode) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "name", &object.String{Value: node.name})
	attrs := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for _, key := range sortedStringMapKeys(node.attrs) {
		setHashMember(attrs, key, &object.String{Value: node.attrs[key]})
	}
	setHashMember(out, "attributes", attrs)
	children := make([]object.Object, len(node.children))
	for i, child := range node.children {
		children[i] = xmlNodeToObject(child)
	}
	setHashMember(out, "children", &object.Array{Elements: children})
	setHashMember(out, "text", &object.String{Value: strings.TrimSpace(strings.Join(node.textParts, ""))})
	return out
}

func objectToXMLNode(value object.Object) (*xmlNode, error) {
	hash, ok := value.(*object.Hash)
	if !ok {
		return nil, &xml.SyntaxError{Msg: "node must be an object"}
	}
	name, ok := hashString(hash, "name")
	if !ok || name == "" {
		return nil, &xml.SyntaxError{Msg: "node.name must be a string"}
	}
	node := &xmlNode{name: name, attrs: make(map[string]string)}
	if attrsObj, ok := hashValue(hash, "attributes"); ok {
		if attrs, ok := attrsObj.(*object.Hash); ok {
			for _, pair := range attrs.Pairs {
				if s, ok := pair.Value.(*object.String); ok {
					node.attrs[objectToMapKey(pair.Key)] = s.Value
				}
			}
		}
	}
	if text, ok := hashString(hash, "text"); ok && text != "" {
		node.textParts = append(node.textParts, text)
	}
	if childrenObj, ok := hashValue(hash, "children"); ok {
		if children, ok := childrenObj.(*object.Array); ok {
			for _, childObj := range children.Elements {
				child, err := objectToXMLNode(childObj)
				if err != nil {
					return nil, err
				}
				child.parent = node
				node.children = append(node.children, child)
			}
		}
	}
	return node, nil
}

func writeXMLNode(out *bytes.Buffer, node *xmlNode) error {
	out.WriteByte('<')
	out.WriteString(node.name)
	for _, key := range sortedStringMapKeys(node.attrs) {
		out.WriteByte(' ')
		out.WriteString(key)
		out.WriteString(`="`)
		xml.EscapeText(out, []byte(node.attrs[key]))
		out.WriteByte('"')
	}
	if len(node.children) == 0 && len(node.textParts) == 0 {
		out.WriteString("/>")
		return nil
	}
	out.WriteByte('>')
	for _, text := range node.textParts {
		xml.EscapeText(out, []byte(text))
	}
	for _, child := range node.children {
		if err := writeXMLNode(out, child); err != nil {
			return err
		}
	}
	out.WriteString("</")
	out.WriteString(node.name)
	out.WriteByte('>')
	return nil
}

func sortedStringMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
