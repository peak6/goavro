package goavro

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type CodecGenerator struct {
	writeDecoderSrc          func(w io.Writer) error
	genDecodeInstanceSrc     func() string
	genNativeTypeNameSrc     func() string
	genNativeDefaultValueSrc func() string
	getImports               func() []string
	isWritable               bool
}

func NewBoolCodecGenerator() *CodecGenerator {
	return &CodecGenerator{
		genDecodeInstanceSrc:     func() string { return "goavro.BoolNativeFromBinary" },
		genNativeTypeNameSrc:     func() string { return "bool" },
		genNativeDefaultValueSrc: func() string { return "false" },
		getImports:               func() []string { return []string{} },
	}
}

func NewDoubleCodecGenerator() *CodecGenerator {
	return &CodecGenerator{
		genDecodeInstanceSrc:     func() string { return "goavro.DoubleNativeFromBinary" },
		genNativeTypeNameSrc:     func() string { return "float64" },
		genNativeDefaultValueSrc: func() string { return "0.0" },
		getImports:               func() []string { return []string{} },
	}
}

func NewFloatCodecGenerator() *CodecGenerator {
	return &CodecGenerator{
		genDecodeInstanceSrc:     func() string { return "goavro.FloatNativeFromBinary" },
		genNativeTypeNameSrc:     func() string { return "float32" },
		genNativeDefaultValueSrc: func() string { return "0.0" },
		getImports:               func() []string { return []string{} },
	}
}

func NewLongCodecGenerator() *CodecGenerator {
	return &CodecGenerator{
		genDecodeInstanceSrc:     func() string { return "goavro.LongNativeFromBinary" },
		genNativeTypeNameSrc:     func() string { return "int64" },
		genNativeDefaultValueSrc: func() string { return "0" },
		getImports:               func() []string { return []string{} },
	}
}

func NewIntCodecGenerator() *CodecGenerator {
	return &CodecGenerator{
		genDecodeInstanceSrc:     func() string { return "goavro.IntNativeFromBinary" },
		genNativeTypeNameSrc:     func() string { return "int32" },
		genNativeDefaultValueSrc: func() string { return "0" },
		getImports:               func() []string { return []string{} },
	}
}

func NewStringCodecGenerator() *CodecGenerator {
	return &CodecGenerator{
		genDecodeInstanceSrc:     func() string { return "goavro.StringNativeFromBinary" },
		genNativeTypeNameSrc:     func() string { return "string" },
		genNativeDefaultValueSrc: func() string { return "\"\"" },
		getImports:               func() []string { return []string{} },
	}
}

func NewIntDateCodecGenerator() *CodecGenerator {
	return &CodecGenerator{
		genDecodeInstanceSrc:     func() string { return "goavro.NativeFromBinaryDate" },
		genNativeTypeNameSrc:     func() string { return "time.Time" },
		genNativeDefaultValueSrc: func() string { return "time.Time{}" },
		getImports:               func() []string { return []string{"time"} },
	}
}

func NewDecimalBytesCodecGenerator(precision, scale int) *CodecGenerator {
	return &CodecGenerator{
		getImports:               func() []string { return []string{"math/big"} },
		genNativeTypeNameSrc:     func() string { return "*big.Rat" },
		genNativeDefaultValueSrc: func() string { return "&big.Rat{}" },
		genDecodeInstanceSrc: func() string {
			return fmt.Sprintf("func (buf []byte) (*big.Rat, []byte, error) {\nreturn goavro.NativeFromBinaryDecimalBytes(buf, %d, %d)\n}",
				precision, scale)
		},
	}
}

func NewUnionCodecGenerator(codecFromIndex []*Codec) (*CodecGenerator, error) {
	var realCodec *Codec

	if len(codecFromIndex) != 2 {
		return nil, fmt.Errorf("the codec generator ONLY supports unions with 2 values, one of which must be null")
	}
	if codecFromIndex[0].typeName.fullName != "null" && codecFromIndex[1].typeName.fullName == "null" {
		realCodec = codecFromIndex[0]
	} else if codecFromIndex[1].typeName.fullName != "null" && codecFromIndex[0].typeName.fullName == "null" {
		realCodec = codecFromIndex[1]
	} else {
		return nil, fmt.Errorf("invalid union configuration for codec generator %s -- %s", codecFromIndex[0].typeName.fullName, codecFromIndex[1].typeName.fullName)
	}

	gen := &CodecGenerator{
		getImports:               func() []string { return append([]string{"fmt"}, realCodec.generator.getImports()...) },
		genNativeTypeNameSrc:     func() string { return realCodec.generator.genNativeTypeNameSrc() },
		genNativeDefaultValueSrc: func() string { return realCodec.generator.genNativeDefaultValueSrc() },
	}

	gen.genDecodeInstanceSrc = func() string {
		var w bytes.Buffer

		w.WriteString(fmt.Sprintf("func(buf []byte) (%s, []byte, error) {\n", gen.genNativeTypeNameSrc()))
		w.WriteString("tmpBuf := buf\n")
		w.WriteString("idx, tmpBuf, err := goavro.LongNativeFromBinary(tmpBuf)\n")
		w.WriteString(fmt.Sprintf("if err != nil { return %s, buf, err }\n", gen.genNativeDefaultValueSrc()))
		w.WriteString("switch idx {\n")

		for i, fieldCodec := range codecFromIndex {
			w.WriteString(fmt.Sprintf("case %d:\n", i))
			if codecFromIndex[i].typeName.fullName == "null" {
				w.WriteString("// Null case, use empty value\n")
				w.WriteString(fmt.Sprintf("return %s, tmpBuf, nil\n", gen.genNativeDefaultValueSrc()))
			} else {
				w.WriteString(fmt.Sprintf("return  %s(tmpBuf)\n", fieldCodec.generator.genDecodeInstanceSrc()))
			}
		}
		w.WriteString(fmt.Sprintf("default:\n"))
		w.WriteString(fmt.Sprintf("return %s, buf, fmt.Errorf(\"union index out of bounds\")\n", gen.genNativeDefaultValueSrc()))
		w.WriteString("}\n")
		w.WriteString("}")

		return w.String()
	}

	return gen, nil
}

func NewArrayCodecGenerator(realCodec *Codec) *CodecGenerator {
	gen := &CodecGenerator{
		getImports:               func() []string { return append([]string{"fmt"}, realCodec.generator.getImports()...) },
		genNativeTypeNameSrc:     func() string { return fmt.Sprintf("[]%s", realCodec.generator.genNativeTypeNameSrc()) },
		genNativeDefaultValueSrc: func() string { return fmt.Sprintf("[]%s{}", realCodec.generator.genNativeTypeNameSrc()) },
	}

	gen.genDecodeInstanceSrc = func() string {

		var w bytes.Buffer
		w.WriteString(fmt.Sprintf("func(buf []byte) (%s, []byte, error) {\n", gen.genNativeTypeNameSrc()))
		w.WriteString(fmt.Sprintf("var value %s\n", realCodec.generator.genNativeTypeNameSrc()))
		w.WriteString("var err error\n")
		w.WriteString("var blockCount int64\n")
		w.WriteString("tmpBuf := buf\n\n")
		w.WriteString(fmt.Sprintf("blockCount, tmpBuf, err = goavro.DecodeBlockCount(tmpBuf)\n"))
		w.WriteString(fmt.Sprintf("if err != nil { return %s, buf, err }\n\n", gen.genNativeDefaultValueSrc()))

		w.WriteString(fmt.Sprintf("arrayValues := make(%s, 0, blockCount)\n\n", gen.genNativeTypeNameSrc()))

		w.WriteString(`
				for blockCount != 0 {
					// Decode 'blockCount' datum values
					for i := int64(0); i < blockCount; i++ {
`)
		w.WriteString(fmt.Sprintf("if value, tmpBuf, err = %s(tmpBuf); err != nil {\n", realCodec.generator.genDecodeInstanceSrc()))
		w.WriteString(fmt.Sprintf("return %s, buf, fmt.Errorf(\"cannot decode binary array item %%d: %%s\", i+1, err)\n", gen.genNativeDefaultValueSrc()))
		w.WriteString("} else {\n") // End "if value, ..."
		w.WriteString("arrayValues = append(arrayValues, value)\n\n")
		w.WriteString("}\n")
		w.WriteString("}\n") // End "for i := ..."

		w.WriteString(fmt.Sprintf("blockCount, tmpBuf, err = goavro.DecodeBlockCount(tmpBuf)\n"))
		w.WriteString(fmt.Sprintf("if err != nil { return %s, buf, err }\n\n", gen.genNativeDefaultValueSrc()))

		w.WriteString("}\n") // End "for BlockCount != 0"

		w.WriteString("return arrayValues, tmpBuf, nil\n")
		w.WriteString("}")

		return w.String()
	}

	return gen
}

func NewRecordCodecGenerator(recordTypeName *name, codecFromIndex []*Codec, nameFromIndex []string) *CodecGenerator {
	gen := &CodecGenerator{
		getImports:               func() []string { return []string{} },
		genNativeTypeNameSrc:     func() string { return "*" + recordTypeName.short() },
		genNativeDefaultValueSrc: func() string { return "nil" },
		genDecodeInstanceSrc:     func() string { return fmt.Sprintf("New%s", recordTypeName.short()) },
		isWritable:               true,
	}

	gen.writeDecoderSrc = func(w io.Writer) error {
		imports := make([]string, 0)
		for _, fieldCodec := range codecFromIndex {
			if fieldCodec.generator.getImports != nil {
				imports = append(imports, fieldCodec.generator.getImports()...)
			}
		}
		imports = append(imports, "github.com/peak6/goavro/v2")

		w.Write([]byte("import (\n"))
		for _, imp := range imports {
			w.Write([]byte(fmt.Sprintf("\"%s\"\n", imp)))
		}
		w.Write([]byte(")\n\n"))

		// Write the struct
		w.Write([]byte(fmt.Sprintf("type %s struct {\n", recordTypeName.short())))
		for i, fieldCodec := range codecFromIndex {
			w.Write([]byte(fmt.Sprintf("%s %s\n", strings.Title(nameFromIndex[i]), fieldCodec.generator.genNativeTypeNameSrc())))
		}
		w.Write([]byte("}\n\n"))

		// Write the full decoder
		w.Write([]byte(fmt.Sprintf("func New%s(buf []byte) (*%s, []byte, error) {\n",
			recordTypeName.short(), recordTypeName.short())))
		w.Write([]byte(fmt.Sprintf("result := &%s{}\n", recordTypeName.short())))
		w.Write([]byte("newBuf := buf\n"))
		w.Write([]byte("var err error\n\n"))

		for i, fieldCodec := range codecFromIndex {
			name := nameFromIndex[i]
			w.Write([]byte(fmt.Sprintf(
				"if result.%s, newBuf, err = %s(newBuf); err != nil {\nreturn nil, newBuf, err\n\t}\n\n",
				strings.Title(name), fieldCodec.generator.genDecodeInstanceSrc())))
		}
		w.Write([]byte("return result, newBuf, nil\n}\n"))
		return nil
	}

	return gen
}
