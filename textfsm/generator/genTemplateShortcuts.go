// genTemplateShortcuts generates shortcut functions for all the templates included.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/cdevr/cpush/textfsm"
	"github.com/cdevr/cpush/utils"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	out = flag.String("out", "", "where to place the output")
	dir = flag.String("dir", "templates", "where to find the textfsm templates")
)

type shortCutFuncData struct {
	Name           string
	QuotedTemplate string
}

type typedShortCutFuncData struct {
	Name string
	FSM  *textfsm.TextFSM
}

func IsList(v textfsm.Value) bool {
	for _, o := range v.Options {
		if o == "List" {
			return true
		}
	}
	return false
}

const (
	parseShortcutFnTemplate = `
const {{.Name}}Template = {{.QuotedTemplate}}

func Parse{{.Name}}(input string)  ([]map[string]interface{}, error) {
	return Parse({{.Name}}Template, input, true)
}`
	parseTypeShortcutFnTemplate = `
type {{.Name}}Row struct { {{range $name, $val := .FSM.Values}}
	{{FixFieldName $name}} {{if IsList $val}}[]string{{else}}string{{end}}{{end}}
}

func ParseTyped{{.Name}}(input string) ([]{{.Name}}Row, error) {
	result, err := ParseIntoStruct([]{{.Name}}Row{}, {{.Name}}Template, input, true)
	return result.([]{{.Name}}Row), err
}
`
)

func main() {
	flag.Parse()

	templates, err := filepath.Glob(fmt.Sprintf("%s/*.textfsm", *dir))
	if err != nil {
		log.Fatalf("failed to find templates in directory %q: %v", *dir, err)
	}

	var b = &bytes.Buffer{}

	fmt.Fprintf(b, "package textfsm\n\n")

	typedShortcutFuncs := template.FuncMap{
		"IsList":       IsList,
		"FixFieldName": textfsm.FixFieldName,
	}

	tmpl := template.Must(template.New("parseShortcutFnTemplate").Parse(parseShortcutFnTemplate))
	typedTmpl := template.Must(template.New("parseTypedShortcutFnTempalte").Funcs(typedShortcutFuncs).Parse(parseTypeShortcutFnTemplate))
	for _, templateFn := range templates {
		tTmpl, err := os.ReadFile(templateFn)
		if err != nil {
			log.Fatalf("failed to read template %q: %v", templateFn, err)
		}
		tTmplStr := string(tTmpl)
		fsm, err := textfsm.NewTextFSM(tTmplStr)
		if err != nil {
			log.Fatalf("failed to parse template %q into textFSM: %v", templateFn, err)
		}

		fn := path.Base(templateFn)
		camelName := textfsm.ToCamel(strings.TrimSuffix(fn, filepath.Ext(fn)))

		data := shortCutFuncData{camelName, fmt.Sprintf("%q", tTmpl)}
		err = tmpl.Execute(b, data)
		if err != nil {
			log.Fatalf("failed to execute function generation template: %v", err)
		}

		data2 := typedShortCutFuncData{camelName, fsm}
		err = typedTmpl.Execute(b, data2)
		if err != nil {
			log.Fatalf("failed to execute function generation template: %v", err)
		}
	}

	fmt.Fprintf(b, "\n")

	err = utils.ReplaceFile(*out, b.String())
	if err != nil {
		log.Fatalf("failed to replace file %q: %v", *out, err)
	}
}
