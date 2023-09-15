package cisco

import "testing"

func TestRemovePromptSuffix(t *testing.T) {
	tests := []struct {
		Input string
		Want  string
	}{
		{
			"output line\noutput line2\nprompt#",
			"output line\noutput line2",
		},
		{
			"just some output",
			"just some output",
		},
	}

	for _, test := range tests {
		got := RemovePromptSuffix(test.Input)

		if got != test.Want {
			t.Errorf("RemovePromptSuffix(%q): got %q want %q", test.Input, got, test.Want)
		}
	}
}
