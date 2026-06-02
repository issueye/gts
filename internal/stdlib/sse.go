package stdlib

import (
	"io"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type sseReader struct {
	stream *readableStream
	done   bool
}

func init() {
	module.RegisterNative("@std/sse", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initSSEModule(exports)
		return exports, nil
	})
}

func initSSEModule(exports *object.Hash) {
	setHashMember(exports, "reader", &object.Builtin{Name: "sse.reader", Fn: sseReaderFromStream})
	setHashMember(exports, "parse", &object.Builtin{Name: "sse.parse", Fn: sseParseString})
}

func sseReaderFromStream(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "sse.reader requires a stream")
	}
	stream, errObj := streamFromObject(pos, "sse.reader", args[0])
	if errObj != nil {
		return errObj
	}
	reader := &sseReader{stream: stream}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "__sseReader", &object.GoObject{Value: reader})
	setHashMember(obj, "next", &object.Builtin{Name: "sse.next", Fn: sseNext, Extra: &object.GoObject{Value: reader}})
	setHashMember(obj, "readAll", &object.Builtin{Name: "sse.readAll", Fn: sseReadAll, Extra: &object.GoObject{Value: reader}})
	return obj
}

func sseParseString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, "sse.parse", args, 0, "text")
	if errObj != nil {
		return errObj
	}
	events := parseSSEBlock(strings.Split(text, "\n"))
	elements := make([]object.Object, len(events))
	for i, ev := range events {
		elements[i] = ev
	}
	return &object.Array{Elements: elements}
}

func sseNext(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	reader, errObj := boundSSEReader(pos, env, "sse.next")
	if errObj != nil {
		return errObj
	}
	event, err := readSSEEvent(reader)
	if err != nil {
		return object.NewError(pos, "sse.next: %v", err)
	}
	if event == nil {
		return object.NULL
	}
	return event
}

func sseReadAll(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	reader, errObj := boundSSEReader(pos, env, "sse.readAll")
	if errObj != nil {
		return errObj
	}
	events := []object.Object{}
	for {
		event, err := readSSEEvent(reader)
		if err != nil {
			return object.NewError(pos, "sse.readAll: %v", err)
		}
		if event == nil {
			break
		}
		events = append(events, event)
	}
	return &object.Array{Elements: events}
}

func readSSEEvent(reader *sseReader) (object.Object, error) {
	if reader.done {
		return nil, nil
	}
	lines := []string{}
	for {
		line, err := reader.stream.reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			if err != io.EOF {
				return nil, err
			}
			reader.done = true
			if len(lines) == 0 {
				return nil, nil
			}
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if len(lines) == 0 {
				continue
			}
			break
		}
		lines = append(lines, line)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			reader.done = true
			break
		}
	}
	events := parseSSEBlock(lines)
	if len(events) == 0 {
		return readSSEEvent(reader)
	}
	return events[0], nil
}

func parseSSEBlock(lines []string) []*object.Hash {
	blocks := [][]string{}
	current := []string{}
	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r\n")
		if line == "" {
			if len(current) > 0 {
				blocks = append(blocks, current)
				current = []string{}
			}
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		blocks = append(blocks, current)
	}

	events := make([]*object.Hash, 0, len(blocks))
	for _, block := range blocks {
		eventType := "message"
		eventID := ""
		retry := ""
		dataParts := []string{}
		for _, line := range block {
			if strings.HasPrefix(line, ":") {
				continue
			}
			field := line
			value := ""
			if idx := strings.Index(line, ":"); idx >= 0 {
				field = line[:idx]
				value = line[idx+1:]
				if strings.HasPrefix(value, " ") {
					value = value[1:]
				}
			}
			switch field {
			case "event":
				eventType = value
			case "data":
				dataParts = append(dataParts, value)
			case "id":
				eventID = value
			case "retry":
				retry = value
			}
		}
		event := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(event, "type", &object.String{Value: eventType})
		setHashMember(event, "data", &object.String{Value: strings.Join(dataParts, "\n")})
		if eventID != "" {
			setHashMember(event, "id", &object.String{Value: eventID})
		}
		if retry != "" {
			setHashMember(event, "retry", &object.String{Value: retry})
		}
		events = append(events, event)
	}
	return events
}

func boundSSEReader(pos ast.Position, env *object.Environment, name string) (*sseReader, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing SSE reader receiver", name)
	}
	reader, ok := goObj.Value.(*sseReader)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid SSE reader receiver", name)
	}
	return reader, nil
}

func streamFromObject(pos ast.Position, name string, obj object.Object) (*readableStream, *object.Error) {
	hash, ok := obj.(*object.Hash)
	if !ok {
		return nil, object.NewError(pos, "%s: argument must be a stream object", name)
	}
	streamObj, ok := hashValue(hash, "__stream")
	if !ok {
		return nil, object.NewError(pos, "%s: argument must be a stream object", name)
	}
	goObj, ok := streamObj.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid stream object", name)
	}
	stream, ok := goObj.Value.(*readableStream)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid stream object", name)
	}
	return stream, nil
}
