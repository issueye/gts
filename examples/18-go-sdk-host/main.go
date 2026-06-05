package main

import (
	"fmt"
	"time"

	"github.com/issueye/goscript/sdk"
)

func main() {
	rt := sdk.NewRuntime(sdk.Options{
		Timeout: 5 * time.Second,
		Workers: 1,
	})
	defer rt.Close()

	if err := rt.RegisterModule(sdk.Module{
		Name: "@go/demo",
		Values: map[string]any{
			"name": "Go SDK demo",
		},
		Methods: map[string]sdk.Method{
			"add": func(ctx sdk.CallContext, args []sdk.Value) (sdk.Value, error) {
				reader := sdk.NewArgs(ctx, args)
				left, err := reader.Number(0, "left")
				if err != nil {
					return nil, err
				}
				right, err := reader.Number(1, "right")
				if err != nil {
					return nil, err
				}
				return sdk.Number(left + right), nil
			},
		},
		MethodsAny: map[string]sdk.AnyMethod{
			"hostInfo": func(ctx sdk.CallContext, args []sdk.Value) (any, error) {
				return map[string]any{
					"runtime": "go",
					"module":  ctx.Method,
				}, nil
			},
		},
		Docs: []string{
			"add(left, right) -> number",
			"hostInfo() -> object",
		},
	}); err != nil {
		panic(err)
	}

	result, err := rt.RunSource(`
		const demo = require("@go/demo");

		let total = demo.add(20, 22);
		let info = demo.hostInfo();

		({
			moduleName: demo.name,
			total: total,
			info: info,
		});
	`, "examples/18-go-sdk-host/inline.gs")
	if err != nil {
		panic(err)
	}

	fmt.Printf("GTS result: %#v\n", sdk.FromValue(result))

	summary, err := rt.CallExport(
		"examples/18-go-sdk-host/tool.gs",
		"summarize",
		sdk.Object(map[string]sdk.Value{
			"title": sdk.String("SDK host"),
			"items": sdk.Array(
				sdk.String("register Go module"),
				sdk.String("run GTS script"),
				sdk.String("call GTS export"),
			),
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("GTS export result: %#v\n", sdk.FromValue(summary))
}
