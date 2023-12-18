package main

import (
	"os"

	"github.com/carlmjohnson/exitcode"
	"github.com/spotlightpa/pdfcopy/app"
)

func main() {
	exitcode.Exit(app.CLI(os.Args[1:]))
}
