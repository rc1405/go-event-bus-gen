package main

import (
	"bytes"
	_ "embed"
	"go/format"
	"os"
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
var logger zerolog.Logger

type Attribute struct {
	Name    string
	Type    string
	RawName string
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

type Template struct {
	Package string
	Structs []Struct
	Methods []Method
	Imports []string
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

	parsedBuf, err := protoparser.Parse(protoBuf)
	if err != nil {
		logger.Error().Err(err).Msgf("error parsing protobuf in %s", inFile)
		return err
	}

	tmplData := Template{
		Imports: config.Imports,
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
					msg.Attributes = append(msg.Attributes, Attribute{
						Name:    strcase.ToCamel(f.FieldName),
						Type:    f.Type,
						RawName: f.FieldName,
					})
				default:
					logger.Warn().Msgf("unsupported message attribute %v", b)
				}
			}
			tmplData.Structs = append(tmplData.Structs, msg)
		default:
			logger.Debug().Msgf("unsupported type %s", b)
		}
	}

	tmpl, err := template.New("test").Parse(event_bus_tmpl)
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
