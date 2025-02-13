package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/concreteit/greenlight"
	"github.com/concreteit/greenlight/internal"
	"github.com/concreteit/greenlight/js"
	"github.com/jedib0t/go-pretty/v6/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate NeTEx files",
		Run:   validate,
	}
	scripts = map[string]*js.Script{}
)

func init() {
	if scriptMap, err := compileBuiltin(); err != nil {
		log.Fatal(err)
	} else {
		scripts = scriptMap
	}

	validateCmd.Flags().StringP("input", "i", "", "XML file, dir or archive to validate")
	validateCmd.Flags().StringP("log-level", "l", "debug", "Set level of log output (one of \"trace\", \"debug\", \"info\", \"warn\", \"error\")")
	validateCmd.Flags().StringP("output", "o", "pretty", "Set which output format to use (one of \"json\", \"xml\", \"csv\", \"pretty\"")
	validateCmd.Flags().StringP("profile", "p", "", "Set path of validation profile (note: flags 'rules' and 'schema' is ignored)")
	validateCmd.Flags().StringSliceP("rules", "r", []string{}, "Set which validation rules to run (defaults to all inside the builtin dir)")
	validateCmd.Flags().StringP("schema", "s", "netex@1.2", "Which xsd schema to use (supported \"netex@1.2\", \"netex@1.2-nc\", \"epip@1.1.1\", \"epip@1.1.1-nc\")")
	validateCmd.Flags().BoolP("silent", "", false, "Running in silent will only output the result in a boolean fashion")

	// read properties from environment
	viper.SetEnvPrefix("GREENLIGHT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// bind from cli input
	viper.BindPFlag("input", validateCmd.Flags().Lookup("input"))
	viper.BindPFlag("log.level", validateCmd.Flags().Lookup("log-level"))
	viper.BindPFlag("output", validateCmd.Flags().Lookup("output"))
	viper.BindPFlag("profile", validateCmd.Flags().Lookup("profile"))
	viper.BindPFlag("rules", validateCmd.Flags().Lookup("rules"))
	viper.BindPFlag("schema", validateCmd.Flags().Lookup("schema"))
	viper.BindPFlag("silent", validateCmd.Flags().Lookup("silent"))

	rootCmd.AddCommand(validateCmd)
}

func openWithContext(ctx *FileContext, input string) error {
	fi, err := os.Stat(input)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		entries, err := os.ReadDir(input)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if err := openWithContext(ctx, fmt.Sprintf("%s/%s", input, entry.Name())); err != nil {
				return err
			}
		}
	} else {
		file, err := os.Open(input)
		if err != nil {
			return err
		}
		if err := ctx.Open(input, file); err != nil {
			return err
		}
		defer file.Close()
	}

	return nil
}

func createValidation(input string) (*greenlight.Validation, error) {
	validation, err := greenlight.NewValidation()
	if err != nil {
		return nil, err
	}

	if path := viper.GetString("profile"); path != "" {
		profile, err := OpenProfile(path)
		if err != nil {
			return nil, err
		}

		log.Debugf("validating using profile at '%s'", path)

		for _, script := range profile.Scripts {
			for name, s := range scripts {
				if name == script.Name {
					validation.AddScript(s, script.Config)
				}
			}
		}
	} else {
		schema := viper.GetString("schema")
		if schema == "" {
			return nil, fmt.Errorf("no schema version defined")
		}

		validation.AddScript(scripts["xsd"], map[string]interface{}{
			"schema": schema,
		})

		rules := viper.GetStringSlice("rules")
		for name, script := range scripts {
			if name == "xsd" {
				continue
			}
			if rules == nil || len(rules) == 0 {
				validation.AddScript(script, nil)
			} else {
				for _, r := range rules {
					if r == name {
						validation.AddScript(script, nil)
					}
				}
			}
		}
	}

	fileContext := NewFileContext(context.Background())
	defer fileContext.Close()
	if err := openWithContext(fileContext, input); err != nil {
		return nil, err
	}

	for _, file := range fileContext.Find("xml") {
		f, err := file.Open()
		if err != nil {
			return nil, err
		}

		if err := validation.AddReader(file.Name, f); err != nil {
			return nil, err
		}
	}

	// subscribe to validation updates
	validation.Subscribe(func(event internal.Event) {
		fields := log.Fields{
			"id":    event.ContextID,
			"type":  event.Type,
			"scope": "main",
		}

		if event.Type == internal.EventTypeLog && event.Data != nil {
			level := log.DebugLevel
			message := ""
			for k, v := range event.Data {
				if k == "message" {
					message = v.(string)
				} else if k == "level" {
					if l, ok := v.(string); ok {
						if lvl, err := log.ParseLevel(l); err == nil {
							level = lvl
						}
					}
				} else {
					fields[k] = v
				}
			}

			log.WithTime(event.Timestamp).WithFields(fields).Log(level, message)
		}
	})

	return validation, nil
}

func validate(cmd *cobra.Command, args []string) {
	silent := viper.GetBool("silent")
	if silent {
		log.SetLevel(log.FatalLevel)
	} else {
		level, err := log.ParseLevel(viper.GetString("log.level"))
		if err != nil {
			log.Fatal(err)
		}

		log.SetLevel(level)
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	input := viper.GetString("input")
	if input == "" {
		log.Fatal("no input provided")
	}

	validation, err := createValidation(internal.DirExpand(input))
	if err != nil {
		log.Fatal(err)
	}

	res, err := validation.Validate(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	if silent {
		for _, r := range res {
			if !r.Valid {
				return
			}
		}
		return
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		buf, err := json.MarshalIndent(res, "", " ")
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println(string(buf))
		}
	case "xml":
		buf, err := xml.MarshalIndent(ValidationResults{
			ValidationResult: res,
		}, "", " ")
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(string(buf))
		}
	case "csv":
		for i, result := range res {
			rows := result.CsvRecords(i == 0)
			for _, row := range rows {
				fmt.Println(strings.Join(row, ","))
			}
		}
	case "pretty":
		for _, r := range res {
			tw, _, err := terminal.GetSize(0)
			if err != nil {
				log.Fatal(err)
			}

			rows := r.CsvRecords(true)
			w := table.NewWriter()
			w.SetAllowedRowLength(tw)
			w.SetStyle(table.StyleLight)
			w.SetOutputMirror(os.Stdout)
			w.SetTitle(r.Name)
			for i, row := range rows {
				r := table.Row{}
				if i == 0 {
					r = append(r, "#")
				} else {
					r = append(r, i)
				}
				for _, v := range row {
					r = append(r, v)
				}
				if i == 0 {
					w.AppendHeader(r)
				} else {
					w.AppendRow(r)
				}
			}
			w.AppendSeparator()
			w.AppendFooter(table.Row{"", "", "valid", r.Valid})
			w.Render()
		}
	}
}

type ValidationResults struct {
	ValidationResult []greenlight.ValidationResult `xml:"ValidationResult"`
}
