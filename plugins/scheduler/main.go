package main

import (
	"fmt"
	"os"

	"github.com/issueye/goscript/plugins/scheduler/pkg/scheduler"
)

func main() {
	if err := scheduler.NewService().Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
