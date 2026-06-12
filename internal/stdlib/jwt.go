package stdlib

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/jwt", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "sign", &object.Builtin{Name: "jwt.sign", Fn: jwtSign})
		setHashMember(exports, "verify", &object.Builtin{Name: "jwt.verify", Fn: jwtVerify})
		setHashMember(exports, "decode", &object.Builtin{Name: "jwt.decode", Fn: jwtDecode})
		return exports, nil
	})
}

func jwtSign(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "jwt.sign requires payload and secret")
	}

	payload, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "jwt.sign expects hash payload")
	}
	secret, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "jwt.sign expects string secret")
	}

	// 转换 payload 为 map
	payloadMap := hashToMap(payload)

	// 添加默认的 iat
	if _, exists := payloadMap["iat"]; !exists {
		payloadMap["iat"] = time.Now().Unix()
	}

	// Header
	header := map[string]interface{}{"alg": "HS256", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Payload
	payloadJSON, _ := json.Marshal(payloadMap)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Signature
	message := headerB64 + "." + payloadB64
	h := hmac.New(sha256.New, []byte(secret.Value))
	h.Write([]byte(message))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	token := message + "." + signature
	return &object.String{Value: token}
}

func jwtVerify(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "jwt.verify requires token and secret")
	}

	token, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "jwt.verify expects string token")
	}
	secret, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "jwt.verify expects string secret")
	}

	parts := strings.Split(token.Value, ".")
	if len(parts) != 3 {
		return object.FALSE
	}

	message := parts[0] + "." + parts[1]
	h := hmac.New(sha256.New, []byte(secret.Value))
	h.Write([]byte(message))
	expectedSig := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	if parts[2] != expectedSig {
		return object.FALSE
	}

	// 检查过期
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return object.FALSE
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return object.FALSE
	}

	if exp, ok := payload["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return object.FALSE
		}
	}

	return object.TRUE
}

func jwtDecode(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "jwt.decode requires token")
	}

	token, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "jwt.decode expects string token")
	}

	parts := strings.Split(token.Value, ".")
	if len(parts) != 3 {
		return object.NewError(pos, "jwt.decode: invalid token format")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return object.NewError(pos, "jwt.decode: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return object.NewError(pos, "jwt.decode: %v", err)
	}

	return mapToHash(payload)
}

func hashToMap(h *object.Hash) map[string]interface{} {
	result := make(map[string]interface{})
	for _, pair := range h.Pairs {
		if key, ok := pair.Key.(*object.String); ok {
			result[key.Value] = objectToInterface(pair.Value)
		}
	}
	return result
}

func objectToInterface(obj object.Object) interface{} {
	switch o := obj.(type) {
	case *object.String:
		return o.Value
	case *object.Number:
		return o.Value
	case *object.Boolean:
		return o.Value
	default:
		return nil
	}
}

func mapToHash(m map[string]interface{}) *object.Hash {
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for k, v := range m {
		key := &object.String{Value: k}
		var value object.Object
		switch val := v.(type) {
		case string:
			value = &object.String{Value: val}
		case float64:
			value = &object.Number{Value: val}
		case bool:
			value = &object.Boolean{Value: val}
		default:
			value = object.NULL
		}
		result.Pairs[object.HashKeyFor(key)] = object.HashPair{Key: key, Value: value}
	}
	return result
}
