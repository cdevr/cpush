package utils

import "testing"

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
