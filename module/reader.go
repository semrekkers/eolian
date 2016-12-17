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

type Buffer struct {
	Reader
	Frame
}

type Closer interface {
	Close() error
}

type ReaderCloser interface {
	Reader
	Closer
}

type Inspecter interface {
	Inspect() string
}

func NewBuffer(r Reader) *Buffer {
	return &Buffer{r, make(Frame, FrameSize)}
}

func (setter *Buffer) SetSource(r Reader) {
	switch v := setter.Reader.(type) {
	case SourceSetter:
		v.SetSource(r)
	case Reader:
		setter.Reader = r
	}
}

func (reader *Buffer) ReadFrame() Frame {
	reader.Reader.Read(reader.Frame)
	return reader.Frame
}

func (closer *Buffer) Close() error {
	if c, ok := closer.Reader.(Closer); ok {
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
