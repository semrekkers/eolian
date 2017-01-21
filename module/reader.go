package module

import "fmt"

type Reader interface {
	Read(Frame)
}

type SourceSetter interface {
	SetSource(Reader)
}

type ReadValuer interface {
	Reader
	Valuer
}

type Closer interface {
	Close() error
}

type Inspecter interface {
	Inspect() string
}

type Buffer struct {
	Reader
	Frame
}

func NewBuffer(r Reader) *Buffer {
	return &Buffer{
		Reader: r,
		Frame:  make(Frame, FrameSize),
	}
}

func (b *Buffer) SetSource(r Reader) {
	switch v := b.Reader.(type) {
	case SourceSetter:
		v.SetSource(r)
	case Reader:
		b.Reader = r
	}
}

func (b *Buffer) ReadFrame() Frame {
	b.Reader.Read(b.Frame)
	return b.Frame
}

func (b *Buffer) Close() error {
	if c, ok := b.Reader.(Closer); ok {
		return c.Close()
	}
	return nil
}

func (b *Buffer) String() string {
	return fmt.Sprintf("%v", b.Reader)
}

func Provide(r Reader) ReaderProvider {
	return ReaderProviderFunc(func() Reader { return r })
}

type ReaderProvider interface {
	Reader() Reader
}

type ReaderProviderFunc func() Reader

func (fn ReaderProviderFunc) Reader() Reader {
	return fn()
}
