package main

import (
	"fmt"

	"flag"

	"github.com/peak6/goavro/v2"
)

func main() {
	var outputDir string
	var packageName string
	var verbose bool
	flag.StringVar(&outputDir, "o", ".", "The directory in which to write the generated files")
	flag.StringVar(&packageName, "p", "records", "The name of the package for generated files")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.Parse()

	files := flag.Args()

	err := goavro.Generate(packageName, files, outputDir, verbose)
	if err != nil {
		fmt.Printf("failed to generate, reason: %v", err)
	}
}
