package main

import (
	"fmt"
	"os"

	"github.com/issueye/goscript/internal/gtp/scheduler"
)

func main() {
	if err := scheduler.NewService().Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
