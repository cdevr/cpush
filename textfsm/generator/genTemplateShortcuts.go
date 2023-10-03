// genTemplateShortcuts generates shortcut functions for all the templates included.
package main

import (
	"bytes"
	"flag"
	"fmt"
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

// Converts a string to CamelCase
func toCamel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	n := strings.Builder{}
	n.Grow(len(s))
	capNext := true
	prevIsCap := false
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if capNext {
			if vIsLow {
				v += 'A'
				v -= 'a'
			}
		} else if i == 0 {
			if vIsCap {
				v += 'a'
				v -= 'A'
			}
		} else if prevIsCap && vIsCap {
			v += 'a'
			v -= 'A'
		}
		prevIsCap = vIsCap

		if vIsCap || vIsLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.'
		}
	}
	return n.String()
}

const (
	parseShortcutFnTemplate = `
const {{.Name}}Template = {{.QuotedTemplate}}

func Parse{{.Name}} (input string)  ([]map[string]interface{}, error) {
	return Parse({{.Name}}Template, input, true)
}`
)

type shortCutFuncData struct {
	Name           string
	QuotedTemplate string
}

func main() {
	flag.Parse()

	templates, err := filepath.Glob(fmt.Sprintf("%s/*.textfsm", *dir))
	if err != nil {
		log.Fatalf("failed to find templates in directory %q: %v", *dir, err)
	}

	var b = &bytes.Buffer{}

	fmt.Fprintf(b, "package textfsm\n\n")

	tmpl := template.Must(template.New("parseShortcutFnTemplate").Parse(parseShortcutFnTemplate))
	for _, templateFn := range templates {
		tTmpl, err := os.ReadFile(templateFn)
		if err != nil {
			log.Fatalf("failed to read template %q: %v", templateFn, err)
		}

		fn := path.Base(templateFn)
		camelName := toCamel(strings.TrimSuffix(fn, filepath.Ext(fn)))

		data := shortCutFuncData{camelName, fmt.Sprintf("%q", tTmpl)}
		err = tmpl.Execute(b, data)
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
