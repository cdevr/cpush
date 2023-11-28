package cisco

import (
	"strings"
	"testing"

	"github.com/go-test/deep"
)

func dedent(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return ""
	}
	if strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	if len(lines) == 0 {
		return ""
	}
	firstLine := lines[0]
	indentLevel := len(firstLine) - len(strings.TrimSpace(firstLine))
	indent := firstLine[:indentLevel]

	var result []string
	for _, line := range lines {
		result = append(result, strings.TrimPrefix(line, indent))
	}
	return strings.Join(result, "\n")
}

func TestDedent(t *testing.T) {
	got := dedent(`
	boem
	  babies`)

	want := "boem\n  babies"
	if got != want {
		t.Errorf("dedent: got %q want %q", got, want)
	}
}

func TestLinesToFormalSimple(t *testing.T) {
	tests := []struct {
		Description string
		Input       string
		Want        string
	}{
		{
			"trivial example",
			`description boembabies`,
			`description boembabies`,
		},
		{
			"basic section",
			dedent(`
			interface loopback0
			 description boembabies`,
			),
			"interface loopback0\ninterface loopback0 description boembabies",
		},
		{
			"multiline section",
			dedent(`
			interface loopback0
			 description boembabies
			 ip address 1.0.0.1 255.255.255.252`,
			),
			"interface loopback0\ninterface loopback0 description boembabies\ninterface loopback0 ip address 1.0.0.1 255.255.255.252",
		},
	}

	for _, test := range tests {
		got := ConfigToFormal(test.Input)
		if diff := deep.Equal(got, test.Want); diff != nil {
			t.Error(diff)
			t.Errorf("test %q: differences found: got\n%s\nwant\n%s\ndiff\n%s\n", test.Description, got, test.Want, strings.Join(diff, "\n"))
		}
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		Description string
		Input       string
		WantEqual   bool
		WantDiff    []string
		WantErr     error
	}{
		{
			"trivial example",
			"description boembabies",
			true,
			[]string{
				`description boembabies`,
			},
			nil,
		},
	}

	for _, test := range tests {
		gotEqual, gotDiff, gotErr := Compare(test.Input, "")
		if gotEqual != test.WantEqual {
			t.Errorf("test %s: equal got %t want %t", test.Description, gotEqual, test.WantEqual)
		}

		if diff := deep.Equal(gotDiff, test.WantDiff); diff != nil {
			t.Errorf("test %q: differences found: got\n%s\nwant\n%s\ndiff\n%s\n", test.Description, gotDiff, test.Input, strings.Join(diff, "\n"))
		}

		if gotErr != test.WantErr {
			t.Errorf("test %q: err got %v want %v", test.Description, gotErr, test.WantErr)
		}
	}
}
