// Package module provides built-in modules
package module

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// IO is the input/output registry of a module. It manages the lifecycles of the ports; fascilitating connects and
// disconnects between them. This struct lazy initializes so it is useful by default. It is intended to just be embedded
// inside other structs that represent a module.
type IO struct {
	sync.Mutex
	ins  map[string]*In
	outs map[string]*Out
}

// Expose registers inputs and outputs of the module so that they can be used in patching
func (io *IO) Expose(ins []*In, outs []*Out) error {
	io.Lock()
	defer io.Unlock()
	io.lazyInit()
	for _, in := range ins {
		if _, ok := io.ins[in.Name]; ok {
			return fmt.Errorf(`duplicate input exposed "%s"`, in.Name)
		}
		in.io = io
		if b, ok := in.Source.(*Buffer); ok {
			in.initial = b.Reader
		} else {
			in.initial = in.Source
		}
		io.ins[in.Name] = in
	}
	for _, out := range outs {
		if _, ok := io.outs[out.Name]; ok {
			return fmt.Errorf(`duplicate output exposed "%s"`, out.Name)
		}
		if out.Provider == nil {
			return fmt.Errorf(`provider must be set for output "%s"`, out.Name)
		}
		out.io = io
		io.outs[out.Name] = out
	}
	return nil
}

// Patch assigns an input's reader to some source (Reader, Value, etc)
func (inout *IO) Patch(name string, t interface{}) error {
	inout.Lock()
	defer inout.Unlock()
	inout.lazyInit()
	input, ok := inout.ins[name]
	if !ok {
		return fmt.Errorf(`unknown input "%s"`, name)
	}

	if err := input.Close(); err != nil {
		return err
	}

	var (
		reader Reader
		err    error
	)

	switch v := t.(type) {
	case Port:
		if reader, err = v.Patcher.Output(v.Port); err != nil {
			return err
		}
	case Value:
		reader = v
	case Pitch:
		reader = v
	case Hz:
		reader = v
	case MS:
		reader = v
	case string:
		if floatV, err := strconv.ParseFloat(v, 64); err == nil {
			reader = Value(floatV)
		} else if intV, err := strconv.Atoi(v); err == nil {
			reader = Value(intV)
		} else {
			r, err := ParseValueString(v)
			if err != nil {
				return err
			}
			reader = r.(ReadValuer)
		}
	case int:
		reader = Value(v)
	case float64:
		reader = Value(v)
	case Reader:
		reader = v
	default:
		return fmt.Errorf(`unpatchable source value %v (%T)`, v, v)
	}

	input.SetSource(reader)

	return nil
}

// Inputs lists all registered inputs
func (io *IO) Inputs() map[string]*In {
	io.Lock()
	defer io.Unlock()
	io.lazyInit()
	return io.ins
}

// Outputs lists all registered outputs
func (io *IO) Outputs() map[string]*Out {
	io.Lock()
	defer io.Unlock()
	io.lazyInit()
	return io.outs
}

// Output realizes a registered output and returns it for patching
func (io *IO) Output(name string) (Reader, error) {
	io.Lock()
	defer io.Unlock()
	io.lazyInit()
	if o, ok := io.outs[name]; ok {
		if o.IsActive() {
			return nil, fmt.Errorf(`output "%s" is already patched`, name)
		}

		r := &ReaderCloser{io, name, o.Provider.Reader()}
		io.outs[name].reader = r
		return r, nil
	}
	return nil, fmt.Errorf(`output "%s" doesn't exist`, name)
}

// OutputsActive returns the total count of actively patched outputs
func (io *IO) OutputsActive() int {
	io.Lock()
	defer io.Unlock()
	io.lazyInit()
	var i int
	for _, out := range io.outs {
		if out.IsActive() {
			i++
		}
	}
	return i
}

// Inspect returns a formatted string detailing the internal state of the module
func (io *IO) Inspect() string {
	io.Lock()
	defer io.Unlock()
	out := "inputs:\n"
	for name, in := range io.ins {
		out += fmt.Sprintf("- %s: %v\n", name, in)
	}
	out += "outputs:\n"
	for name, e := range io.outs {
		if e.IsActive() {
			out += fmt.Sprintf("- %s (active)\n", name)
		} else {
			out += fmt.Sprintf("- %s\n", name)
		}
	}
	return strings.TrimRight(out, "\n")
}

func (inout *IO) closeOutput(name string) error {
	inout.Lock()
	defer inout.Unlock()
	inout.lazyInit()
	if o, ok := inout.outs[name]; ok {
		o.reader = nil
		return nil
	}
	return fmt.Errorf(`output "%s" doesn't exist`, name)
}

func (io *IO) lazyInit() {
	if io.ins == nil {
		io.ins = map[string]*In{}
	}
	if io.outs == nil {
		io.outs = map[string]*Out{}
	}
}

// Reset disconnects all inputs from their sources (closing them in the process) and re-assigns the input to its
// original default value
func (io *IO) Reset() error {
	io.Lock()
	defer io.Unlock()
	for _, in := range io.ins {
		if nested, ok := in.Source.(*In); ok {
			if err := nested.Close(); err != nil {
				return err
			}
		} else {
			if err := in.Close(); err != nil {
				return err
			}
			in.SetSource(in.initial)
		}
	}
	return nil
}

type Patcher interface {
	Patch(string, interface{}) error
	Output(string) (Reader, error)
	Reset() error
}

type Lister interface {
	Inputs() map[string]*In
	Outputs() map[string]*Out
}

// In is a module input
type In struct {
	sync.Mutex

	Source Reader
	Name   string

	io      *IO
	initial Reader
}

// Read reads the output of the source into a Frame
func (reader *In) Read(f Frame) {
	reader.Lock()
	reader.Source.Read(f)
	reader.Unlock()
}

// SetSource sets the internal source to some Reader
func (setter *In) SetSource(r Reader) {
	setter.Lock()
	defer setter.Unlock()

	switch v := setter.Source.(type) {
	case SourceSetter:
		v.SetSource(r)
	case Reader:
		setter.Source = r
	}
}

func (i *In) String() string {
	return fmt.Sprintf("%v", i.Source)
}

// ReadFrame reads an entire frame into the buffered input
func (i *In) ReadFrame() Frame {
	i.Lock()
	defer i.Unlock()
	return i.Source.(*Buffer).ReadFrame()
}

// LastFrame returns the last frame read with ReadFrame
func (i *In) LastFrame() Frame {
	i.Lock()
	defer i.Unlock()
	return i.Source.(*Buffer).Frame
}

// Close closes the input
func (i *In) Close() error {
	i.Lock()
	defer i.Unlock()
	if closer, ok := i.Source.(Closer); ok {
		return closer.Close()
	}
	return nil
}

// Out is a module output
type Out struct {
	Name     string
	Provider ReaderProvider

	io     *IO
	reader Reader
}

// IsActive returns whether or not there is a realized Reader assigned
func (o *Out) IsActive() bool {
	return o.reader != nil
}

// Close closes the output
func (closer *Out) Close() error {
	if v, ok := closer.reader.(Closer); ok {
		return v.Close()
	}
	closer.reader = nil
	return nil
}

type ReaderCloser struct {
	io   *IO
	Name string
	Reader
}

func (closer *ReaderCloser) Close() error {
	return closer.io.closeOutput(closer.Name)
}

// Port represents the address of a specific port on a Patcher
type Port struct {
	Patcher
	Port string
}
