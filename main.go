package main

import (
	"os"

	"github.com/rejisterjack/scrute/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
