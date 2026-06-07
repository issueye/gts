package gtp

import (
	"fmt"

	sdkgtp "github.com/issueye/goscript/sdk/gtp"

	"github.com/issueye/goscript/internal/object"
)

const Version = sdkgtp.Version

type Frame = sdkgtp.Frame
type Error = sdkgtp.Error
type Value = sdkgtp.Value
type Encoder = sdkgtp.Encoder
type Decoder = sdkgtp.Decoder

var (
	Undefined         = sdkgtp.Undefined
	Null              = sdkgtp.Null
	Bool              = sdkgtp.Bool
	Number            = sdkgtp.Number
	String            = sdkgtp.String
	Bytes             = sdkgtp.Bytes
	Array             = sdkgtp.Array
	Object            = sdkgtp.Object
	Resource          = sdkgtp.Resource
	EncodeFrame       = sdkgtp.EncodeFrame
	DecodeFrame       = sdkgtp.DecodeFrame
	NewEncoder        = sdkgtp.NewEncoder
	NewDecoder        = sdkgtp.NewDecoder
	EncodeJSONL       = sdkgtp.EncodeJSONL
	Field             = sdkgtp.Field
	StringField       = sdkgtp.StringField
	NumberField       = sdkgtp.NumberField
	Plain             = sdkgtp.Plain
	RequiredObjectArg = sdkgtp.RequiredObjectArg
	TypeError         = sdkgtp.TypeError
	HostError         = sdkgtp.HostError
	NotFoundError     = sdkgtp.NotFoundError
	OKResult          = sdkgtp.OKResult
	ErrorResult       = sdkgtp.ErrorResult
)

func FromObject(obj object.Object) Value {
	switch v := obj.(type) {
	case *object.Undefined:
		return Undefined()
	case *object.Null:
		return Null()
	case *object.Boolean:
		return Bool(v.Value)
	case *object.Number:
		return Number(v.Value)
	case *object.String:
		return String(v.Value)
	case *object.Array:
		items := make([]Value, len(v.Elements))
		for i, item := range v.Elements {
			items[i] = FromObject(item)
		}
		return Array(items)
	case *object.Hash:
		fields := make(map[string]Value, len(v.Pairs))
		for _, pair := range v.OrderedPairs() {
			if key, ok := pair.Key.(*object.String); ok {
				fields[key.Value] = FromObject(pair.Value)
			}
		}
		return Object(fields)
	case *object.Error:
		return Value{Type: "error", Name: v.Name, Message: v.Message}
	case *object.GoObject:
		return Resource(fmt.Sprintf("%p", v.Value), fmt.Sprintf("%T", v.Value), nil)
	default:
		return Null()
	}
}
