package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fuale/eicg/internal"
	"github.com/fuale/eicg/internal/lexer"
	"github.com/fuale/eicg/internal/parser"
	"github.com/fuale/eicg/internal/printer"
)

func main() {
	setupLogger()
	flags := setupFlags()

	// Open file for reading, but not read entire file.
	src, err := os.Open(flags.Source)
	if err != nil {
		log.Fatalf("fail obtaining resource: %s", err)
	}

	// Don't forget to close the file.
	defer src.Close()

	// Main pipeline.

	// 1. Lexer. Splits the file into tokens.
	//    Here lexer is just created and performs no
	//    tokenization, basically, it is in a `idle` state.
	lex := lexer.New(src)

	// 2. Parser. Parses the tokens into ASTs.
	//    When Parser tries to analyze the next token, it will
	//    use lexer to provide one - this way lexer and parser will work simultaneously.
	ast := parser.New(lex).Parse()

	// 3. Printer. Prints the AST at specific format.
	//    Printing is done by simply walking the AST and converting
	//    `parser.Expression` to string.
	python := printer.New(ast).PrintPython()

	// 4. Write output.
	writeOutput(python, flags.Source, ".py")

	internal.DebugBlock("compiled to python", python)
}

// Helper function to write output to file.
func writeOutput(value, source, extension string) {
	for i := len(source) - 1; i >= 0 && !os.IsPathSeparator(source[i]); i-- {
		if source[i] == '.' {
			os.WriteFile(source[:i]+extension, []byte(value), 0644)
		}
	}
}

// Helper function to setup logger, which makes it logs the filename and location.
func setupLogger() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stdout)
}

type Flags struct {
	Source string
}

// Helper function to get arguments and flags.
func setupFlags() Flags {
	flag.Parse()
	source := flag.Arg(0)

	if source == "" {
		fmt.Printf("Usage: %s <file>\n", os.Args[0])
		os.Exit(22)
	}

	return Flags{
		Source: source,
	}
}
