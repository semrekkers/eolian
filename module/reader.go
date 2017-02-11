package module

import (
	"fmt"
	"io"
)

// Reader is the interface that wraps reading to Frames.
type Reader interface {
	Read(Frame)
}

// SourceSetter is something that can set its source
type SourceSetter interface {
	SetSource(Reader)
}

type readValuer interface {
	Reader
	Valuer
}

// Buffer is a buffered source. Every read Frame is stored
type Buffer struct {
	Reader
	Frame
}

// NewBuffer returns a new Buffer
func NewBuffer(r Reader) *Buffer {
	return &Buffer{
		Reader: r,
		Frame:  make(Frame, FrameSize),
	}
}

// SetSource establishes the Reader that is called when the Buffer is read
func (b *Buffer) SetSource(r Reader) {
	switch v := b.Reader.(type) {
	case SourceSetter:
		v.SetSource(r)
	case Reader:
		b.Reader = r
	}
}

// ReadFrame reads a frame of data from the source
func (b *Buffer) ReadFrame() Frame {
	b.Reader.Read(b.Frame)
	return b.Frame
}

// Close closes the underlying Reader if necessary
func (b *Buffer) Close() error {
	if c, ok := b.Reader.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (b *Buffer) String() string {
	return fmt.Sprintf("%v", b.Reader)
}

// Provide is a helper function that creates a ReaderProvider with a directly given Reader
func Provide(r Reader) ReaderProvider {
	return ReaderProviderFunc(func() Reader { return r })
}

// ReaderProvider allows for delayed retrieval of Readers. This is typically used in outputs where the output should
// only be created when requested in a patch.
type ReaderProvider interface {
	Reader() Reader
}

// ReaderProviderFunc is a function that acts as as ReaderProvider
type ReaderProviderFunc func() Reader

// Reader yields the new Reader
func (fn ReaderProviderFunc) Reader() Reader {
	return fn()
}
