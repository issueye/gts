package stdlib

import (
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/mail", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initMailModule(exports)
		return exports, nil
	})
}

func initMailModule(exports *object.Hash) {
	setHashMember(exports, "parseAddress", &object.Builtin{Name: "mail.parseAddress", Fn: mailParseAddress})
	setHashMember(exports, "parseAddressList", &object.Builtin{Name: "mail.parseAddressList", Fn: mailParseAddressList})
	setHashMember(exports, "parseMessage", &object.Builtin{Name: "mail.parseMessage", Fn: mailParseMessage})
	setHashMember(exports, "formatAddress", &object.Builtin{Name: "mail.formatAddress", Fn: mailFormatAddress})
	setHashMember(exports, "formatAddressList", &object.Builtin{Name: "mail.formatAddressList", Fn: mailFormatAddressList})
	setHashMember(exports, "parseDate", &object.Builtin{Name: "mail.parseDate", Fn: mailParseDate})
	setHashMember(exports, "formatDate", &object.Builtin{Name: "mail.formatDate", Fn: mailFormatDate})
	setHashMember(exports, "getHeader", &object.Builtin{Name: "mail.getHeader", Fn: mailGetHeader})
}

func mailParseAddress(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "mail.parseAddress", args, 0, "address")
	if errObj != nil {
		return errObj
	}
	addr, err := mail.ParseAddress(value)
	if err != nil {
		return object.NewError(pos, "mail.parseAddress: %v", err)
	}
	return mailAddressObject(addr)
}

func mailParseAddressList(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "mail.parseAddressList", args, 0, "addresses")
	if errObj != nil {
		return errObj
	}
	addrs, err := mail.ParseAddressList(value)
	if err != nil {
		return object.NewError(pos, "mail.parseAddressList: %v", err)
	}
	out := make([]object.Object, len(addrs))
	for i, addr := range addrs {
		out[i] = mailAddressObject(addr)
	}
	return &object.Array{Elements: out}
}

func mailParseMessage(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "mail.parseMessage", args, 0, "message")
	if errObj != nil {
		return errObj
	}
	msg, err := mail.ReadMessage(strings.NewReader(value))
	if err != nil {
		return object.NewError(pos, "mail.parseMessage: %v", err)
	}
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return object.NewError(pos, "mail.parseMessage: %v", err)
	}
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "headers", mailHeaderObject(msg.Header))
	setHashMember(out, "body", &object.String{Value: string(body)})
	return out
}

func mailFormatAddress(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	addr, errObj := mailAddressFromObject(pos, "mail.formatAddress", args, 0)
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: addr.String()}
}

func mailFormatAddressList(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "mail.formatAddressList requires addresses")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "mail.formatAddressList: addresses must be an array")
	}
	addrs := make([]string, 0, len(arr.Elements))
	for _, item := range arr.Elements {
		addr, errObj := mailAddressObjectFromValue(pos, "mail.formatAddressList", item)
		if errObj != nil {
			return errObj
		}
		addrs = append(addrs, addr.String())
	}
	return &object.String{Value: strings.Join(addrs, ", ")}
}

func mailParseDate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "mail.parseDate", args, 0, "date")
	if errObj != nil {
		return errObj
	}
	t, err := mail.ParseDate(value)
	if err != nil {
		return object.NewError(pos, "mail.parseDate: %v", err)
	}
	return &object.Date{Time: t}
}

func mailFormatDate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	t := time.Now()
	if len(args) >= 1 && args[0] != object.UNDEFINED && args[0] != object.NULL {
		parsed, errObj := timeFromObject(pos, "mail.formatDate", args, 0)
		if errObj != nil {
			return errObj
		}
		t = parsed
	}
	return &object.String{Value: t.Format(time.RFC1123Z)}
}

func mailGetHeader(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "mail.getHeader requires headers")
	}
	headers, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "mail.getHeader: headers must be an object")
	}
	name, errObj := requiredString(pos, "mail.getHeader", args, 1, "name")
	if errObj != nil {
		return errObj
	}
	for _, pair := range headers.Pairs {
		if strings.EqualFold(pair.Key.Inspect(), name) {
			if arr, ok := pair.Value.(*object.Array); ok {
				if len(arr.Elements) == 0 {
					return object.UNDEFINED
				}
				return arr.Elements[0]
			}
			return pair.Value
		}
	}
	return object.UNDEFINED
}

func mailAddressObject(addr *mail.Address) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "name", &object.String{Value: addr.Name})
	setHashMember(out, "address", &object.String{Value: addr.Address})
	return out
}

func mailAddressFromObject(pos ast.Position, name string, args []object.Object, index int) (*mail.Address, *object.Error) {
	if len(args) <= index {
		return nil, object.NewError(pos, "%s requires address", name)
	}
	return mailAddressObjectFromValue(pos, name, args[index])
}

func mailAddressObjectFromValue(pos ast.Position, name string, value object.Object) (*mail.Address, *object.Error) {
	if s, ok := value.(*object.String); ok {
		addr, err := mail.ParseAddress(s.Value)
		if err != nil {
			return nil, object.NewError(pos, "%s: %v", name, err)
		}
		return addr, nil
	}
	hash, ok := value.(*object.Hash)
	if !ok {
		return nil, object.NewError(pos, "%s: address must be a string or object", name)
	}
	address, ok := hashString(hash, "address")
	if !ok || address == "" {
		return nil, object.NewError(pos, "%s: address.address is required", name)
	}
	displayName, _ := hashString(hash, "name")
	return &mail.Address{Name: displayName, Address: address}, nil
}

func mailHeaderObject(header mail.Header) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for key, values := range header {
		setHashMember(out, key, strSliceToArray(values))
	}
	return out
}
