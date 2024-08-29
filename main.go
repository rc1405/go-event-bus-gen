package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"io"
	"os"
	"reflect"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/yoheimuta/go-protoparser/v4"
	"github.com/yoheimuta/go-protoparser/v4/parser"
	"gopkg.in/yaml.v3"
)

//go:embed codegen.tmpl
var event_bus_tmpl string
var rootCmd *cobra.Command
var inFile string
var outFile string
var confFile string
var logger zerolog.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

var protoToGoTypes = map[string]string{
	"double":   "float64",
	"float":    "float32",
	"int32":    "int32",
	"int64":    "int64",
	"uint32":   "uint32",
	"uint64":   "uint64",
	"sint32":   "int32",
	"sint64":   "int64",
	"fixed32":  "uint32",
	"fixed64":  "uint64",
	"sfixed32": "int32",
	"sfixed64": "int64",
	"bool":     "bool",
	"string":   "string",
	"bytes":    "[]byte",
}

type Attribute struct {
	Name     string
	Type     string
	RawName  string
	Optional bool
	Repeated bool
}

type Struct struct {
	Name       string
	Attributes []Attribute
}

type Method struct {
	Name      string
	Input     string
	HasOutput bool
	Output    string
}

type EnumMember struct {
	Index string
	Name  string
}

type Enum struct {
	Name    string
	Members []EnumMember
}

type Template struct {
	Package string
	Structs []Struct
	Methods []Method
	Enums   []Enum
	Imports []string
}

func contains(data []string, item string) bool {
	for _, i := range data {
		if i == item {
			return true
		}
	}
	return false
}

func New(imports []string, proto io.Reader) (Template, error) {
	tmplData := Template{
		Imports: imports,
	}

	parsedBuf, err := protoparser.Parse(proto)
	if err != nil {
		logger.Error().Err(err).Msgf("error parsing protobuf in %s", inFile)
		return tmplData, err
	}

L:
	for _, body := range parsedBuf.ProtoBody {
		switch b := body.(type) {
		case *parser.Package:
			tmplData.Package = b.Name

		case *parser.Service:
			for _, visitee := range b.ServiceBody {
				m, ok := visitee.(*parser.RPC)
				if !ok {
					logger.Warn().Msgf("unsupported service type %v", b)
					continue L
				}

				method := Method{
					Name: strcase.ToCamel(m.RPCName),
				}

				if strings.Contains(m.RPCRequest.MessageType, ".") {
					method.Input = m.RPCRequest.MessageType
				} else {
					method.Input = strcase.ToCamel(m.RPCRequest.MessageType)
				}

				if m.RPCResponse.MessageType != "google.protobuf.Empty" {
					method.HasOutput = true
					method.Output = strcase.ToCamel(m.RPCResponse.MessageType)
				}
				tmplData.Methods = append(tmplData.Methods, method)
			}

		case *parser.Message:
			var msg Struct
			msg.Name = strcase.ToCamel(b.MessageName)
			for _, attribute := range b.MessageBody {
				switch f := attribute.(type) {
				case *parser.Field:
					gType, ok := protoToGoTypes[f.Type]
					if !ok {
						gType = f.Type
					}

					switch gType {
					case "google.protobuf.Timestamp":
						if !contains(tmplData.Imports, "time") {
							tmplData.Imports = append(tmplData.Imports, "time")
						}
						gType = "time.Time"
					}

					msg.Attributes = append(msg.Attributes, Attribute{
						Name:     strcase.ToCamel(f.FieldName),
						Type:     gType,
						RawName:  f.FieldName,
						Optional: f.IsOptional,
						Repeated: f.IsRepeated,
					})
				case *parser.MapField:
					key, ok := protoToGoTypes[f.KeyType]
					if !ok {
						key = f.Type
					}

					value, ok := protoToGoTypes[f.KeyType]
					if !ok {
						value = f.Type
					}

					switch value {
					case "google.protobuf.Timestamp":
						if !contains(tmplData.Imports, "time") {
							tmplData.Imports = append(tmplData.Imports, "time")
						}
						value = "time.Time"
					}

					msg.Attributes = append(msg.Attributes, Attribute{
						Name:    strcase.ToCamel(f.MapName),
						Type:    fmt.Sprintf("map[%s]%s", key, value),
						RawName: f.MapName,
					})

				default:
					logger.Warn().Msgf("unsupported message attribute %s", reflect.TypeOf(f))
				}
			}
			tmplData.Structs = append(tmplData.Structs, msg)
		case *parser.Enum:
			enum := Enum{
				Name: b.EnumName,
			}

			for _, e := range b.EnumBody {
				switch m := e.(type) {
				case *parser.EnumField:
					enum.Members = append(enum.Members, EnumMember{
						Name:  m.Ident,
						Index: m.Number,
					})
				default:
					logger.Warn().Msgf("unsupported message attribute %s", reflect.TypeOf(m))
				}
			}

			tmplData.Enums = append(tmplData.Enums, enum)
		default:
			logger.Debug().Msgf("unsupported type %s", reflect.TypeOf(b))
		}
	}

	if len(tmplData.Enums) > 0 {
		enumMap := make(map[string]Enum)
		for _, enum := range tmplData.Enums {
			enumMap[enum.Name] = enum
		}

		for index, str := range tmplData.Structs {
			for i, attr := range str.Attributes {
				if enum, ok := enumMap[attr.Type]; ok {
					attr.Type = fmt.Sprintf("%sEnum", enum.Name)
				}
				str.Attributes[i] = attr
			}
			tmplData.Structs[index] = str
		}
	}

	if len(tmplData.Methods) > 0 {
		processedMethods := make(map[string]Method)
		for _, method := range tmplData.Methods {
			val, ok := processedMethods[method.Name]
			if !ok {
				processedMethods[method.Name] = method
				continue
			}

			if val.Input != method.Input {
				logger.Error().Err(err).Msgf("Method %s has multiple inputs: %s | %s", method.Name, method.Input, val.Input)
				return tmplData, fmt.Errorf("Method %s has multiple inputs: %s | %s", method.Name, method.Input, val.Input)
			}

			if val.HasOutput != method.HasOutput {
				logger.Error().Err(err).Msgf("Method %s has multiple return signatures", method.Name)
				return tmplData, fmt.Errorf("Method %s has multiple return signatures", method.Name)
			}

			if val.Output != method.Output {
				logger.Error().Err(err).Msgf("Method %s has multiple outputs: %s | %s", method.Name, method.Output, val.Output)
				return tmplData, fmt.Errorf("Method %s has multiple outputs: %s | %s", method.Name, method.Output, val.Output)
			}
		}
	}

	return tmplData, nil
}

type Config struct {
	Imports []string `yaml:"imports,omitempty"`
}

func init() {
	rootCmd = &cobra.Command{
		Use:  "",
		RunE: parse,
		PreRun: func(cmd *cobra.Command, args []string) {
			logger = zerolog.New(cmd.OutOrStdout()).With().Timestamp().Logger()
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		},
	}

	rootCmd.PersistentFlags().StringVar(&inFile, "in", "", "Protobuf input file")
	rootCmd.PersistentFlags().StringVar(&outFile, "out", "", "Generated Code output file")
	rootCmd.PersistentFlags().StringVar(&confFile, "config", "", "Config file for code generation")
}

func parse(cmd *cobra.Command, args []string) error {
	if err := cmd.ParseFlags(args); err != nil {
		return err
	}

	var config Config
	if confFile != "" {
		confStr, err := os.ReadFile(confFile)
		if err != nil {
			logger.Error().Err(err).Msgf("error reading config file %s", confFile)
			return err
		}

		if err := yaml.Unmarshal(confStr, &config); err != nil {
			logger.Error().Err(err).Msgf("error parsing config file %s", confFile)
			return err
		}
	}

	protoBuf, err := os.Open(inFile)
	if err != nil {
		logger.Error().Err(err).Msgf("error reading input file %s", inFile)
		return err
	}
	defer protoBuf.Close()

	tmplData, err := New(config.Imports, protoBuf)
	if err != nil {
		logger.Error().Err(err).Msgf("error processing input file %s", inFile)
		return err
	}

	processedInputs := map[string]struct{}{}
	processedMethods := map[string]struct{}{}
	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ProcessedInputs": func(name string) bool {
			_, ok := processedInputs[name]
			if !ok {
				processedInputs[name] = struct{}{}
				return false
			}
			return true
		},
		"ProcessedMethods": func(name string) bool {
			_, ok := processedMethods[name]
			if !ok {
				processedMethods[name] = struct{}{}
				return false
			}
			return true
		},
	}

	tmpl, err := template.New("test").Funcs(funcMap).Parse(event_bus_tmpl)
	if err != nil {
		logger.Error().Err(err).Msg("failed parsing generation template")
		return err
	}

	fout, err := os.Create(outFile)
	if err != nil {
		logger.Error().Err(err).Msgf("error creating output file %s", outFile)
		return err
	}
	defer fout.Close()

	buf := bytes.NewBuffer(nil)

	err = tmpl.Execute(buf, tmplData)
	if err != nil {
		logger.Error().Err(err).Msg("failed to render template")
		return err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		logger.Error().Err(err).Msgf("failed running gofmt on %s", outFile)
		return err
	}

	_, err = fout.Write(formatted)
	if err != nil {
		logger.Error().Err(err).Msgf("failed writing output file %s", outFile)
		return err
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("failed running code generation")
	}
}
