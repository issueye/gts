package stdlib

import (
	"io"
	"net/mail"
	"strings"

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

func mailAddressObject(addr *mail.Address) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "name", &object.String{Value: addr.Name})
	setHashMember(out, "address", &object.String{Value: addr.Address})
	return out
}

func mailHeaderObject(header mail.Header) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for key, values := range header {
		setHashMember(out, key, strSliceToArray(values))
	}
	return out
}
