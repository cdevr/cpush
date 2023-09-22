package utils

import (
	"bytes"
	"io"
	"sync"
  "strings"
)

type ThreadSafeBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *ThreadSafeBuffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}

func (b *ThreadSafeBuffer) DiscardUntil(p byte) error {
	b.m.Lock()
	defer b.m.Unlock()

	for {
		b, err := b.b.ReadByte()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if b == p {
			return nil
		}
	}
}

func (b *ThreadSafeBuffer) LastLine() string {
  lines := strings.Split(b.b.String(), "\n")
  if len(lines) > 0 {
    return lines[len(lines)-1]
  }
  return ""
}

func (b *ThreadSafeBuffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}

func (b *ThreadSafeBuffer) Len() int {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Len()
}

func (b *ThreadSafeBuffer) Reset() {
	b.m.Lock()
	defer b.m.Unlock()
	b.b.Reset()
}

func (b *ThreadSafeBuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.String()
}
