package main

import (
	"fmt"
	"os"

	"github.com/issueye/goscript/internal/gtp/imbot"
)

func main() {
	if err := imbot.NewService().Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
