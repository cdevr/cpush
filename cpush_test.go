package main

import "testing"

func TestRemovePromptSuffix(t *testing.T) {
	tests := []struct{
		Input string
		Want string
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

func TestSimpleThreadSafeBuffer(t *testing.T) {
	// Create a Buffer
	b := ThreadSafeBuffer{}

	b.Write([]byte("boembabies"))

	if b.String() != "boembabies" {
		t.Errorf("Can't write and read back from ThreadSafeBuffer")
	}
}