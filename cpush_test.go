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
	testStr := "boembabies"

	// Create a Buffer
	b := ThreadSafeBuffer{}

	if b.Len() != 0 {
		t.Errorf("Unexpected buffer length before writing to ThreadSafeBuffer")
	}

	b.Write([]byte(testStr))

	if b.Len() != len(testStr) {
		t.Errorf("Unexpected buffer length after writing to ThreadSafeBuffer")
	}

	if b.String() != testStr {
		t.Errorf("Can't write and read back from ThreadSafeBuffer")
	}
}
