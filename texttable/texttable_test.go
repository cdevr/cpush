package texttable

import (
	"testing"

	"gotest.tools/assert"
)

func TestTextTable(t *testing.T) {
	want := `one   four seven ten
two   five eight eleven
three six  nine  twelve
`
	input := []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve"}
	got := Columns(input, 4)

	assert.Equal(t, got, want)
}

func TestUnevenTextTable(t *testing.T) {
	want := `one   four seven ten
two   five eight
three six  nine
`
	input := []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}
	got := Columns(input, 4)

	assert.Equal(t, got, want)
}
