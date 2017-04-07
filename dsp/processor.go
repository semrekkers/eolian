package dsp

import (
	"fmt"
	"io"
)

// Processor is the interface that wraps reading to Frames.
type Processor interface {
	Process(Frame)
}

// SourceSetter is something that can set its source
type SourceSetter interface {
	SetSource(Processor)
}

// ProcessValuer is a Processor and Valuer
type ProcessValuer interface {
	Processor
	Valuer
}

// Buffer is a buffered source. Every read Frame is stored
type Buffer struct {
	Processor
	Frame
}

// NewBuffer returns a new Buffer
func NewBuffer(p Processor) *Buffer {
	return &Buffer{
		Processor: p,
		Frame:     make(Frame, FrameSize),
	}
}

// SetSource establishes the Processor that is called when the Buffer is read
func (b *Buffer) SetSource(r Processor) {
	switch v := b.Processor.(type) {
	case SourceSetter:
		v.SetSource(r)
	case Processor:
		b.Processor = r
	}
}

// ProcessFrame reads a frame of data from the source
func (b *Buffer) ProcessFrame() Frame {
	b.Processor.Process(b.Frame)
	return b.Frame
}

// Close closes the underlying Processor if necessary
func (b *Buffer) Close() error {
	if c, ok := b.Processor.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (b *Buffer) String() string {
	return fmt.Sprintf("%v", b.Processor)
}

// Provide is a helper function that creates a ProcessorProvider with a directly given Processor
func Provide(r Processor) ProcessorProvider {
	return ProcessorProviderFunc(func() Processor { return r })
}

// ProcessorProvider allows for delayed retrieval of Processors. This is typically used in outputs where the output should
// only be created when requested in a patch.
type ProcessorProvider interface {
	Processor() Processor
}

// ProcessorProviderFunc is a function that acts as as ProcessorProvider
type ProcessorProviderFunc func() Processor

// Processor yields the new Processor
func (fn ProcessorProviderFunc) Processor() Processor {
	return fn()
}
