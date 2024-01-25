package cisco

import (
	"strings"
	"testing"

	"github.com/go-test/deep"
)

// dedent is a helper function to allow for nicely formatted
// multiline strings in tests. It removes the indentation of the first
// (or second) line from all lines in the string.
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
		{
			"2 multiline sections",
			dedent(`
			interface loopback0
			 description boembabies
			interface loopback1
			 description boembabies2`,
			),
			"interface loopback0\ninterface loopback0 description boembabies\ninterface loopback1\ninterface loopback1 description boembabies2",
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
			"empty example",
			"",
			ConfLine{
				"",
				nil,
			},
		},
		{
			"trivial example",
			"description boembabies",
			ConfLine{
				"",
				[]ConfLine{
					{"description boembabies", nil},
				},
			},
		},
		{
			"trivial multiline example",
			dedent(`
			line1
			line2`),
			ConfLine{
				"",
				[]ConfLine{
					{"line1", nil},
					{"line2", nil},
				},
			},
		},
		{
			"one section example",
			dedent(`
			interface loopback0
			 description boembabies
			 ip address 1.0.0.1 255.255.255.0`),
			ConfLine{
				"",
				[]ConfLine{
					{"interface loopback0", []ConfLine{
						{"description boembabies", nil},
						{"ip address 1.0.0.1 255.255.255.0", nil},
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
			 ip address 2.0.0.1 255.255.255.0`),
			ConfLine{
				"",
				[]ConfLine{
					{"interface loopback0", []ConfLine{
						{"description boembabies", nil},
						{"ip address 1.0.0.1 255.255.255.0", nil},
					}},
					{"interface loopback1", []ConfLine{
						{"description alsoboembabies", nil},
						{"ip address 2.0.0.1 255.255.255.0", nil},
					}},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Description, func(t *testing.T) {
			got, err := Parse(test.Input)
			if err != nil {
				t.Errorf("error in test %q: %v", test.Description, err)
			}
			if diff := deep.Equal(got, test.Want); diff != nil {
				t.Errorf("test %q: differences found: got\n%s\nwant\n%s\ndiff\n%s\n", test.Description, got, test.Want, strings.Join(diff, "\n"))
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		Description string
		Input       ConfLine
		Want        string
	}{
		{
			"empty example",
			ConfLine{
				"",
				nil,
			},
			"",
		},
		{
			"trivial example",
			ConfLine{
				"",
				[]ConfLine{
					{"description boembabies", nil},
				},
			},
			"description boembabies",
		},
		{
			"trivial multiline example",
			ConfLine{
				"",
				[]ConfLine{
					{"line1", nil},
					{"line2", nil},
				},
			},
			dedent(`
			line1
			line2`),
		},
		{
			"one section example",
			ConfLine{
				"",
				[]ConfLine{
					{"interface loopback0", []ConfLine{
						{"description boembabies", nil},
						{"ip address 1.0.0.1 255.255.255.0", nil},
					}},
				},
			},
			dedent(`
			interface loopback0
			 description boembabies
			 ip address 1.0.0.1 255.255.255.0`),
		},
		{
			"two sections test",
			ConfLine{
				"",
				[]ConfLine{
					{"interface loopback0", []ConfLine{
						{"description boembabies", nil},
						{"ip address 1.0.0.1 255.255.255.0", nil},
					}},
					{"interface loopback1", []ConfLine{
						{"description alsoboembabies", nil},
						{"ip address 2.0.0.1 255.255.255.0", nil},
					}},
				},
			},
			dedent(`
			interface loopback0
			 description boembabies
			 ip address 1.0.0.1 255.255.255.0
			interface loopback1
			 description alsoboembabies
			 ip address 2.0.0.1 255.255.255.0`),
		},
	}

	for _, test := range tests {
		t.Run(test.Description, func(t *testing.T) {
			backToString := test.Input.String()
			if diff := deep.Equal(backToString, test.Want); diff != nil {
				t.Errorf("test %q: differences found: got\n%q\nwant\n%q\ndiff\n%s\n", test.Description, backToString, test.Want, strings.Join(diff, "\n"))
			}
		})
	}
}

func TestApply(t *testing.T) {
	tests := []struct {
		Description string
		Config      string
		Apply       string
		Want        string
	}{
		{
			"simple hostname change",
			"hostname boem",
			"hostname babies",
			"hostname babies",
		},
		{
			"simple multilevel change",
			"interface loopback0\n description loopback",
			"interface loopback0\n description newdesc",
			"interface loopback0\n description newdesc",
		},
		{
			"two multilevel changes",
			"interface loopback0\n description loopback0\ninterface loopback1\n description loopback1",
			"interface loopback0\n description boembabies0\ninterface loopback1\n description boembabies1",
			"interface loopback0\n description boembabies0\ninterface loopback1\n description boembabies1",
		},
		{
			"two multilevel changes changing order",
			"interface loopback0\n description loopback0\ninterface loopback1\n description loopback1",
			"interface loopback1\n description boembabies1\ninterface loopback0\n description boembabies0",
			"interface loopback0\n description boembabies0\ninterface loopback1\n description boembabies1",
		},
		{
			"multilevel, multiple statements",
			"interface loopback0\n ip address 1.0.0.1 255.255.255.255\n description loopback0\n shutdown",
			"interface loopback0\n description boembabies",
			"interface loopback0\n ip address 1.0.0.1 255.255.255.255\n description boembabies\n shutdown",
		},
	}

	for _, test := range tests {
		got, err := Apply(test.Config, test.Apply)
		if err != nil {
			t.Errorf("failed to apply: %v", err)
		}

		if diff := deep.Equal(got, test.Want); diff != nil {
			t.Errorf("test %q: differences found: got\n%s\nwant\n%s\ndiff\n%s\n", test.Description, got, test.Want, strings.Join(diff, "\n"))
		}
	}
}
