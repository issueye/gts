package main

import (
	"fmt"
	"os"

	"github.com/issueye/goscript/plugins/im-bot/internal/imbot"
)

func main() {
	if err := imbot.NewService().Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
