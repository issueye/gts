package stdlib

import (
	"net"
	"net/netip"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/net/ip", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initNetIPModule(exports)
		return exports, nil
	})
}

func initNetIPModule(exports *object.Hash) {
	setHashMember(exports, "parseIP", &object.Builtin{Name: "netip.parseIP", Fn: netIPParseIP})
	setHashMember(exports, "parseCIDR", &object.Builtin{Name: "netip.parseCIDR", Fn: netIPParseCIDR})
	setHashMember(exports, "contains", &object.Builtin{Name: "netip.contains", Fn: netIPContains})
	setHashMember(exports, "splitHostPort", &object.Builtin{Name: "netip.splitHostPort", Fn: netIPSplitHostPort})
	setHashMember(exports, "joinHostPort", &object.Builtin{Name: "netip.joinHostPort", Fn: netIPJoinHostPort})
	setHashMember(exports, "lookupHost", &object.Builtin{Name: "netip.lookupHost", Fn: netIPLookupHost})
}

func netIPParseIP(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, "netip.parseIP", args, 0, "ip")
	if errObj != nil {
		return errObj
	}
	addr, err := netip.ParseAddr(text)
	if err != nil {
		return object.UNDEFINED
	}
	return netIPAddrObject(addr)
}

func netIPParseCIDR(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, "netip.parseCIDR", args, 0, "cidr")
	if errObj != nil {
		return errObj
	}
	prefix, err := netip.ParsePrefix(text)
	if err != nil {
		return object.UNDEFINED
	}
	return netIPPrefixObject(prefix)
}

func netIPContains(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cidr, errObj := requiredString(pos, "netip.contains", args, 0, "cidr")
	if errObj != nil {
		return errObj
	}
	ip, errObj := requiredString(pos, "netip.contains", args, 1, "ip")
	if errObj != nil {
		return errObj
	}
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return object.NewError(pos, "netip.contains: invalid cidr")
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return object.NewError(pos, "netip.contains: invalid ip")
	}
	return object.NativeBool(prefix.Contains(addr))
}

func netIPSplitHostPort(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "netip.splitHostPort", args, 0, "address")
	if errObj != nil {
		return errObj
	}
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return object.NewError(pos, "netip.splitHostPort: %v", err)
	}
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "host", &object.String{Value: host})
	setHashMember(out, "port", &object.String{Value: port})
	return out
}

func netIPJoinHostPort(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	host, errObj := requiredString(pos, "netip.joinHostPort", args, 0, "host")
	if errObj != nil {
		return errObj
	}
	port, errObj := requiredString(pos, "netip.joinHostPort", args, 1, "port")
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: net.JoinHostPort(host, port)}
}

func netIPLookupHost(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	host, errObj := requiredString(pos, "netip.lookupHost", args, 0, "host")
	if errObj != nil {
		return errObj
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return object.NewError(pos, "netip.lookupHost: %v", err)
	}
	return strSliceToArray(addrs)
}

func netIPAddrObject(addr netip.Addr) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "value", &object.String{Value: addr.String()})
	setHashMember(out, "is4", object.NativeBool(addr.Is4()))
	setHashMember(out, "is6", object.NativeBool(addr.Is6()))
	setHashMember(out, "isLoopback", object.NativeBool(addr.IsLoopback()))
	setHashMember(out, "isPrivate", object.NativeBool(addr.IsPrivate()))
	setHashMember(out, "isMulticast", object.NativeBool(addr.IsMulticast()))
	return out
}

func netIPPrefixObject(prefix netip.Prefix) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	addr := prefix.Addr()
	setHashMember(out, "value", &object.String{Value: prefix.String()})
	setHashMember(out, "addr", &object.String{Value: addr.String()})
	setHashMember(out, "bits", &object.Number{Value: float64(prefix.Bits())})
	setHashMember(out, "is4", object.NativeBool(addr.Is4()))
	setHashMember(out, "is6", object.NativeBool(addr.Is6()))
	return out
}
