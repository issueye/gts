package stdlib

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

const (
	wsOpContinuation = 0
	wsOpText         = 1
	wsOpBinary       = 2
	wsOpClose        = 8
	wsOpPing         = 9
	wsOpPong         = 10
)

type wsConn struct {
	conn   net.Conn
	reader *bufio.Reader
}

func makeWSConn(conn net.Conn) *wsConn {
	return &wsConn{conn: conn, reader: bufio.NewReader(conn)}
}

func (w *wsConn) Close() error { return w.conn.Close() }

func (w *wsConn) WriteText(data string) error {
	return w.writeFrame(wsOpText, []byte(data))
}

func (w *wsConn) WriteBinary(data []byte) error {
	return w.writeFrame(wsOpBinary, data)
}

func (w *wsConn) ReadMessage() (int, []byte, error) {
	for {
		opcode, payload, err := w.readFrame()
		if err != nil {
			return 0, nil, err
		}
		switch opcode {
		case wsOpText, wsOpBinary:
			return opcode, payload, nil
		case wsOpClose:
			return wsOpClose, payload, nil
		case wsOpPing:
			w.writeFrame(wsOpPong, payload)
		case wsOpPong:
		}
	}
}

func (w *wsConn) writeFrame(opcode int, payload []byte) error {
	w.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	frame := make([]byte, 2)
	frame[0] = 0x80 | byte(opcode)
	length := len(payload)
	if length <= 125 {
		frame[1] = byte(length)
		frame = append(frame, payload...)
	} else if length <= 65535 {
		frame[1] = 126
		frame = append(frame, 0, 0)
		binary.BigEndian.PutUint16(frame[2:4], uint16(length))
		frame = append(frame, payload...)
	} else {
		frame[1] = 127
		frame = append(frame, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(frame[2:10], uint64(length))
		frame = append(frame, payload...)
	}
	_, err := w.conn.Write(frame)
	return err
}

func (w *wsConn) readFrame() (int, []byte, error) {
	for {
		w.conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		header := make([]byte, 2)
		if _, err := io.ReadFull(w.reader, header); err != nil {
			return 0, nil, err
		}
		fin := (header[0] & 0x80) != 0
		opcode := int(header[0] & 0x0F)
		masked := (header[1] & 0x80) != 0
		length := uint64(header[1] & 0x7F)
		switch length {
		case 126:
			ext := make([]byte, 2)
			if _, err := io.ReadFull(w.reader, ext); err != nil {
				return 0, nil, err
			}
			length = uint64(binary.BigEndian.Uint16(ext))
		case 127:
			ext := make([]byte, 8)
			if _, err := io.ReadFull(w.reader, ext); err != nil {
				return 0, nil, err
			}
			length = binary.BigEndian.Uint64(ext)
		}
		var maskKey []byte
		if masked {
			maskKey = make([]byte, 4)
			if _, err := io.ReadFull(w.reader, maskKey); err != nil {
				return 0, nil, err
			}
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(w.reader, payload); err != nil {
			return 0, nil, err
		}
		if masked {
			for i := range payload {
				payload[i] ^= maskKey[i%4]
			}
		}
		if fin {
			return opcode, payload, nil
		}
	}
}

func wsClientConnect(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "ws.connect requires a WebSocket URL")
	}
	urlStr, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "ws.connect: first argument must be a string URL")
	}
	var headers map[string]string
	if len(args) >= 2 {
		h, ok := args[1].(*object.Hash)
		if ok {
			headers = make(map[string]string)
			for _, pair := range h.Pairs {
				headers[pair.Key.Inspect()] = pair.Value.Inspect()
			}
		}
	}
	conn, err := dialWebSocket(urlStr.Value, headers)
	if err != nil {
		return object.NewError(pos, "ws.connect: %v", err)
	}
	return newWSConnObject(conn)
}

func wsServerUpgrade(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "ws.upgrade requires an HTTP request object")
	}
	reqHash, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "ws.upgrade: argument must be an HTTP request object")
	}
	rawReq, ok := hashValue(reqHash, "_raw")
	if !ok {
		return object.NewError(pos, "ws.upgrade: invalid request object (missing _raw)")
	}
	req := rawReq.(*object.GoObject).Value.(*http.Request)
	writerVal, ok := reqHash.Pairs[hashKey(&object.String{Value: "_writer"})]
	if !ok {
		return object.NewError(pos, "ws.upgrade: invalid request object (missing _writer)")
	}
	w := writerVal.Value.(*object.GoObject).Value.(http.ResponseWriter)

	hj, ok := w.(http.Hijacker)
	if !ok {
		return object.NewError(pos, "ws.upgrade: server does not support hijacking")
	}
	netConn, bufrw, err := hj.Hijack()
	if err != nil {
		return object.NewError(pos, "ws.upgrade: %v", err)
	}

	accept := computeAcceptKey(req.Header.Get("Sec-WebSocket-Key"))
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n" +
		"\r\n"
	if _, err := bufrw.WriteString(resp); err != nil {
		netConn.Close()
		return object.NewError(pos, "ws.upgrade: %v", err)
	}
	if err := bufrw.Flush(); err != nil {
		netConn.Close()
		return object.NewError(pos, "ws.upgrade: %v", err)
	}

	ws := makeWSConn(netConn)
	return newWSConnObject(ws)
}

func wsServerCreateServer(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "ws.createServer requires a port and connection handler")
	}
	port, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "ws.createServer: first argument must be a port number")
	}
	handler, ok := args[1].(*object.Function)
	if !ok {
		return object.NewError(pos, "ws.createServer: second argument must be a function")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			http.Error(w, "websocket expected", 400)
			return
		}
		accept := computeAcceptKey(r.Header.Get("Sec-WebSocket-Key"))
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijacking not supported", 500)
			return
		}
		netConn, bufrw, err := hj.Hijack()
		if err != nil {
			return
		}
		resp := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + accept + "\r\n" +
			"\r\n"
		bufrw.WriteString(resp)
		bufrw.Flush()
		ws := makeWSConn(netConn)
		wsObj := newWSConnObject(ws)
		scope := handler.Env.NewScope()
		if len(handler.Parameters) > 0 {
			scope.Set(handler.Parameters[0].Name, wsObj)
		}
		handler.Env.VM().Go(func() {
			handler.Env.VM().EvalNode(handler.Body, scope)
		})
	})

	go func() {
		addr := fmt.Sprintf(":%d", int(port.Value))
		http.ListenAndServe(addr, nil)
	}()

	return object.UNDEFINED
}

func newWSConnObject(ws *wsConn) object.Object {
	connObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(connObj, "_ws", &object.GoObject{Value: ws})
	setHashMember(connObj, "send", &object.Builtin{
		Name: "ws.send",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "ws.send requires data")
			}
			data := args[0].Inspect()
			if err := ws.WriteText(data); err != nil {
				return object.NewError(pos, "ws.send: %v", err)
			}
			return object.UNDEFINED
		},
	})
	setHashMember(connObj, "recv", &object.Builtin{
		Name: "ws.recv",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			_, data, err := ws.ReadMessage()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return object.NULL
				}
				return object.NewError(pos, "ws.recv: %v", err)
			}
			return &object.String{Value: string(data)}
		},
	})
	setHashMember(connObj, "close", &object.Builtin{
		Name: "ws.close",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			ws.Close()
			return object.UNDEFINED
		},
	})
	setHashMember(connObj, "sendText", &object.Builtin{
		Name: "ws.sendText",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "ws.sendText requires a string")
			}
			data := args[0].Inspect()
			if err := ws.WriteText(data); err != nil {
				return object.NewError(pos, "ws.sendText: %v", err)
			}
			return object.UNDEFINED
		},
	})
	return connObj
}

func dialWebSocket(urlStr string, headers map[string]string) (*wsConn, error) {
	u := urlStr
	isSecure := strings.HasPrefix(u, "wss://")
	if strings.HasPrefix(u, "ws://") || isSecure {
		u = strings.TrimPrefix(u, "ws://")
		u = strings.TrimPrefix(u, "wss://")
	}
	host := u
	path := "/"
	if idx := strings.IndexByte(u, '/'); idx >= 0 {
		host = u[:idx]
		path = u[idx:]
	}
	if !strings.Contains(host, ":") {
		if isSecure {
			host += ":443"
		} else {
			host += ":80"
		}
	}
	conn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		return nil, err
	}
	nonce, err := generateNonce()
	if err != nil {
		conn.Close()
		return nil, err
	}
	req := fmt.Sprintf("GET %s HTTP/1.1\r\n", path) +
		fmt.Sprintf("Host: %s\r\n", host) +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		fmt.Sprintf("Sec-WebSocket-Key: %s\r\n", nonce) +
		"Sec-WebSocket-Version: 13\r\n"
	for k, v := range headers {
		req += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	req += "\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if resp.StatusCode != 101 {
		conn.Close()
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	accept := resp.Header.Get("Sec-WebSocket-Accept")
	expected := computeAcceptKey(nonce)
	if accept != expected {
		conn.Close()
		return nil, fmt.Errorf("invalid Sec-WebSocket-Accept")
	}
	return makeWSConn(conn), nil
}

func computeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
