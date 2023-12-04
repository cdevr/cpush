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

func TestParse(t *testing.T) {
	tests := []struct {
		Description string
		Input       string
		Want        ConfLine
	}{
		{
			"trivial example",
			"description boembabies",
			ConfLine{
				"",
				[]ConfLine{
					ConfLine{"description boembabies", nil},
				},
			},
		},
		{
			"trivial multiline example",
			dedent(`
			line1
			line2
			`),
			ConfLine{
				"",
				[]ConfLine{
					ConfLine{"line1", nil},
					ConfLine{"line2", nil},
				},
			},
		},
		{
			"one section example",
			dedent(`
			interface loopback0
			 description boembabies
			 ip address 1.0.0.1 255.255.255.0
			`),
			ConfLine{
				"",
				[]ConfLine{
					ConfLine{"inerface loopback0", []ConfLine{
						ConfLine{"description boembabies", nil},
						ConfLine{"ip address 1.0.0.01 255.255.255.0", nil},
					}},
				},
			},
		},
		{
			"two sections test",
			dedent(`
			interface loopback0
			 description boembabies
			 ip address 1.0.0.1 255.255.255.0
			interface loopback1
			 description alsoboembabies
			 ip address 2.0.0.1 255.255.255.0
			`),
			ConfLine{
				"",
				[]ConfLine{
					ConfLine{"inerface loopback0", []ConfLine{
						ConfLine{"description boembabies", nil},
						ConfLine{"ip address 1.0.0.1 255.255.255.0", nil},
					}},
					ConfLine{"inerface loopback1", []ConfLine{
						ConfLine{"description boembabies", nil},
						ConfLine{"ip address 2.0.0.1 255.255.255.0", nil},
					}},
				},
			},
		},
	}

	for _, test := range tests {
		got := Parse(test.Input)
		if diff := deep.Equal(got, test.Want); diff != nil {
			t.Errorf("test %q: differences found: got\n%s\nwant\n%s\ndiff\n%s\n", test.Description, got, test.Want, strings.Join(diff, "\n"))
		}
	}
}
