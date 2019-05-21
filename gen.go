package goavro

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path"
	"unicode"
)

func Generate(packageName string, inputFiles []string, outputDir string, verbose bool) error {
	symbolTable := newSymbolTable()
	filesLeftToProcess := inputFiles
	var err error

	for {
		prevLen := len(filesLeftToProcess)
		filesLeftToProcess, err = generateCodecs(symbolTable, filesLeftToProcess, verbose)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		newLen := len(filesLeftToProcess)
		if newLen == 0 {
			break
		} else if prevLen == newLen {
			fmt.Println("Not making progress, bailing")
			os.Exit(-1)
		}
	}

	for _, c := range symbolTable {
		if c.generator != nil && c.generator.isWritable {
			outFile := fmt.Sprintf("%s.go", toSnake(c.typeName.short()))
			outPath := path.Join(outputDir, outFile)
			if verbose {
				fmt.Println("Will write", c.typeName.String(), "as", outPath)
			}

			var writer bytes.Buffer

			writer.WriteString("//******************************************\n")
			writer.WriteString("//* This file is is generated, DO NOT EDIT *\n")
			writer.WriteString("//******************************************\n\n")

			writer.WriteString(fmt.Sprintf("package %s\n\n", packageName))
			c.generator.writeDecoderSrc(&writer)

			//fmt.Println(writer.String())

			src, err := format.Source(writer.Bytes())
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}

			f, err := os.Create(outPath)
			if err != nil {
				fmt.Println(err)
			} else {
				_, err := f.Write(src)
				if err != nil {
					fmt.Println(err)
				}
				f.Sync()
				f.Close()
			}
		}
	}

	return nil
}

func generateCodecs(symbolTable map[string]*Codec, inputFiles []string, verbose bool) ([]string, error) {
	errFiles := make([]string, 0)

	for _, f := range inputFiles {
		schemaSpec, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %s", f)
		}

		_, err = newCodecWithSymbolTable(string(schemaSpec), symbolTable)
		if err != nil {
			if verbose {
				fmt.Printf("failed to build codec, file: %s, reason: %v\n", f, err)
			}
			errFiles = append(errFiles, f)
		} else {
			if verbose {
				fmt.Println("parsed", f)
			}
		}
	}

	return errFiles, nil
}

func newCodecWithSymbolTable(schemaSpecification string, st map[string]*Codec) (*Codec, error) {
	var schema interface{}

	if err := json.Unmarshal([]byte(schemaSpecification), &schema); err != nil {
		return nil, fmt.Errorf("cannot unmarshal schema JSON: %s", err)
	}

	c, err := buildCodec(st, nullNamespace, schema)
	if err != nil {
		return nil, err
	}
	c.schemaCanonical, err = parsingCanonicalForm(schema)
	if err != nil {
		return nil, err // should not get here because schema was validated above
	}
	c.schemaOriginal = schemaSpecification
	return c, nil
}

func toSnake(in string) string {
	runes := []rune(in)
	length := len(runes)

	var out []rune
	for i := 0; i < length; i++ {
		if i > 0 && unicode.IsUpper(runes[i]) && ((i+1 < length && unicode.IsLower(runes[i+1])) || unicode.IsLower(runes[i-1])) {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(runes[i]))
	}

	return string(out)
}
