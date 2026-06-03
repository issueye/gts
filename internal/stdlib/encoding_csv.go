package stdlib

import (
	"bytes"
	"encoding/csv"
	"os"
	"sort"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/encoding/csv", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initEncodingCSVModule(exports)
		return exports, nil
	})
}

func initEncodingCSVModule(exports *object.Hash) {
	setHashMember(exports, "parse", &object.Builtin{Name: "csv.parse", Fn: csvParse})
	setHashMember(exports, "stringify", &object.Builtin{Name: "csv.stringify", Fn: csvStringify})
	setHashMember(exports, "readFileSync", &object.Builtin{Name: "csv.readFileSync", Fn: csvReadFileSync})
	setHashMember(exports, "writeFileSync", &object.Builtin{Name: "csv.writeFileSync", Fn: csvWriteFileSync})
}

func csvParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, "csv.parse", args, 0, "text")
	if errObj != nil {
		return errObj
	}
	opts, errObj := csvOptions(pos, "csv.parse", args, 1)
	if errObj != nil {
		return errObj
	}
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = opts.comma
	reader.Comment = opts.comment
	reader.FieldsPerRecord = opts.fieldsPerRecord
	reader.TrimLeadingSpace = opts.trimLeadingSpace
	records, err := reader.ReadAll()
	if err != nil {
		return object.NewError(pos, "csv.parse: %v", err)
	}
	return csvRecordsToObject(records, opts.header)
}

func csvStringify(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "csv.stringify requires rows")
	}
	opts, errObj := csvOptions(pos, "csv.stringify", args, 1)
	if errObj != nil {
		return errObj
	}
	records, errObj := csvRowsFromObject(pos, "csv.stringify", args[0], opts.header)
	if errObj != nil {
		return errObj
	}
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = opts.comma
	if err := writer.WriteAll(records); err != nil {
		return object.NewError(pos, "csv.stringify: %v", err)
	}
	return &object.String{Value: buf.String()}
}

func csvReadFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "csv.readFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return object.NewError(pos, "csv.readFileSync: %v", err)
	}
	parseArgs := []object.Object{&object.String{Value: string(data)}}
	if len(args) >= 2 {
		parseArgs = append(parseArgs, args[1])
	}
	return csvParse(env, pos, parseArgs...)
}

func csvWriteFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "csv.writeFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return object.NewError(pos, "csv.writeFileSync requires rows")
	}
	stringifyArgs := []object.Object{args[1]}
	if len(args) >= 3 {
		stringifyArgs = append(stringifyArgs, args[2])
	}
	out := csvStringify(env, pos, stringifyArgs...)
	if object.IsRuntimeError(out) {
		return out
	}
	text, ok := out.(*object.String)
	if !ok {
		return object.NewError(pos, "csv.writeFileSync: stringify did not return text")
	}
	if err := os.WriteFile(path, []byte(text.Value), 0644); err != nil {
		return object.NewError(pos, "csv.writeFileSync: %v", err)
	}
	return object.UNDEFINED
}

type csvModuleOptions struct {
	header           bool
	comma            rune
	comment          rune
	fieldsPerRecord  int
	trimLeadingSpace bool
}

func csvOptions(pos ast.Position, name string, args []object.Object, index int) (csvModuleOptions, *object.Error) {
	opts := csvModuleOptions{header: true, comma: ',', fieldsPerRecord: 0}
	if len(args) <= index || args[index] == object.UNDEFINED || args[index] == object.NULL {
		return opts, nil
	}
	hash, ok := args[index].(*object.Hash)
	if !ok {
		return opts, object.NewError(pos, "%s: options must be an object", name)
	}
	if value, ok := hashValue(hash, "header"); ok {
		if b, ok := value.(*object.Boolean); ok {
			opts.header = b.Value
		} else {
			return opts, object.NewError(pos, "%s: options.header must be a boolean", name)
		}
	}
	if value, ok := hashValue(hash, "comma"); ok {
		ch, errObj := singleRuneOption(pos, name, "comma", value)
		if errObj != nil {
			return opts, errObj
		}
		opts.comma = ch
	}
	if value, ok := hashValue(hash, "comment"); ok {
		ch, errObj := singleRuneOption(pos, name, "comment", value)
		if errObj != nil {
			return opts, errObj
		}
		opts.comment = ch
	}
	if value, ok := hashValue(hash, "fieldsPerRecord"); ok {
		if n, ok := value.(*object.Number); ok {
			opts.fieldsPerRecord = int(n.Value)
		} else {
			return opts, object.NewError(pos, "%s: options.fieldsPerRecord must be a number", name)
		}
	}
	if value, ok := hashValue(hash, "trimLeadingSpace"); ok {
		if b, ok := value.(*object.Boolean); ok {
			opts.trimLeadingSpace = b.Value
		} else {
			return opts, object.NewError(pos, "%s: options.trimLeadingSpace must be a boolean", name)
		}
	}
	return opts, nil
}

func singleRuneOption(pos ast.Position, name, option string, value object.Object) (rune, *object.Error) {
	text, ok := value.(*object.String)
	if !ok {
		return 0, object.NewError(pos, "%s: options.%s must be a string", name, option)
	}
	runes := []rune(text.Value)
	if len(runes) != 1 {
		return 0, object.NewError(pos, "%s: options.%s must be a single character", name, option)
	}
	return runes[0], nil
}

func csvRecordsToObject(records [][]string, header bool) object.Object {
	if !header {
		rows := make([]object.Object, len(records))
		for i, record := range records {
			rows[i] = strSliceToArray(record)
		}
		return &object.Array{Elements: rows}
	}
	if len(records) == 0 {
		return &object.Array{Elements: []object.Object{}}
	}
	headers := records[0]
	rows := make([]object.Object, 0, len(records)-1)
	for _, record := range records[1:] {
		row := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		for i, key := range headers {
			value := ""
			if i < len(record) {
				value = record[i]
			}
			setHashMember(row, key, &object.String{Value: value})
		}
		rows = append(rows, row)
	}
	return &object.Array{Elements: rows}
}

func csvRowsFromObject(pos ast.Position, name string, rows object.Object, header bool) ([][]string, *object.Error) {
	arr, ok := rows.(*object.Array)
	if !ok {
		return nil, object.NewError(pos, "%s: rows must be an array", name)
	}
	if len(arr.Elements) == 0 {
		return [][]string{}, nil
	}
	if first, ok := arr.Elements[0].(*object.Array); ok {
		out := make([][]string, len(arr.Elements))
		for i, item := range arr.Elements {
			row, ok := item.(*object.Array)
			if !ok {
				return nil, object.NewError(pos, "%s: rows must be all arrays or all objects", name)
			}
			out[i] = csvArrayRow(row)
		}
		if first == nil {
			return [][]string{}, nil
		}
		return out, nil
	}
	headers := csvHeaders(arr)
	out := make([][]string, 0, len(arr.Elements)+1)
	if header {
		out = append(out, headers)
	}
	for i, item := range arr.Elements {
		row, ok := item.(*object.Hash)
		if !ok {
			return nil, object.NewError(pos, "%s: row %d must be an object", name, i)
		}
		record := make([]string, len(headers))
		for j, key := range headers {
			if value, ok := hashValue(row, key); ok {
				record[j] = value.Inspect()
			}
		}
		out = append(out, record)
	}
	return out, nil
}

func csvArrayRow(row *object.Array) []string {
	out := make([]string, len(row.Elements))
	for i, item := range row.Elements {
		out[i] = item.Inspect()
	}
	return out
}

func csvHeaders(rows *object.Array) []string {
	seen := map[string]bool{}
	headers := []string{}
	for _, item := range rows.Elements {
		row, ok := item.(*object.Hash)
		if !ok {
			continue
		}
		for _, pair := range row.Pairs {
			key := pair.Key.Inspect()
			if !seen[key] {
				seen[key] = true
				headers = append(headers, key)
			}
		}
	}
	sort.Strings(headers)
	return headers
}
