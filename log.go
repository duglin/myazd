package main

import (
	"fmt"
	"os"
	"strings"
)

var logIndentSpace = ""

func Log(depth int, format string, args ...interface{}) {
	outdent := false

	if depth > Verbose {
		return
	}

	if len(format) > 0 && format[0] == '<' && len(logIndentSpace) > 1 {
		logIndentSpace = logIndentSpace[:len(logIndentSpace)-2]
		format = format[1:]

		// Log(X, "<") means just outdent, don't print anything
		// If you want a blank line add a space after the "<"
		if format == "" {
			return
		}
	}

	if len(format) > 0 && format[0] == '>' {
		outdent = true
		format = format[1:]
	}

	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}

	fmt.Fprintf(os.Stderr, logIndentSpace+format, args...)

	if outdent {
		logIndentSpace += "| "
	}

}
