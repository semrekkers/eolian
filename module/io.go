// Package module provides built-in modules
package module

import (
	"fmt"
	"strconv"
	"strings"
)

// Patcher is the patching behavior of a module
type Patcher interface {
	Patch(string, interface{}) error
	Output(string) (*Out, error)
	Reset() error
}

// Lister is the port listing behavior of a module
type Lister interface {
	Inputs() map[string]*In
	Outputs() map[string]*Out
}

// IO is the input/output registry of a module. It manages the lifecycles of the ports; fascilitating connects and
// disconnects between them. This struct lazy initializes so it is useful by default. It is intended to just be embedded
// inside other structs that represent a module.
type IO struct {
	ins  map[string]*In
	outs map[string]*Out
}

// Expose registers inputs and outputs of the module so that they can be used in patching
func (io *IO) Expose(ins []*In, outs []*Out) error {
	io.lazyInit()
	for _, in := range ins {
		if _, ok := io.ins[in.Name]; ok {
			return fmt.Errorf(`duplicate input exposed "%s"`, in.Name)
		}
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
		io.outs[out.Name] = out
	}
	return nil
}

// Patch assigns an input's reader to some source (Reader, Value, etc)
func (io *IO) Patch(name string, t interface{}) error {
	io.lazyInit()
	name = canonicalPort(name)
	input, ok := io.ins[name]
	if !ok {
		return fmt.Errorf(`unknown input "%s"`, name)
	}
	if err := input.Close(); err != nil {
		return err
	}
	reader, err := patchReader(t)
	if err != nil {
		return err
	}
	input.SetSource(reader)
	return nil
}

func patchReader(t interface{}) (Reader, error) {
	switch v := t.(type) {
	case Port:
		return v.Patcher.Output(v.Port)
	case Value:
		return v, nil
	case Pitch:
		return v, nil
	case Hz:
		return v, nil
	case MS:
		return v, nil
	case string:
		if floatV, err := strconv.ParseFloat(v, 64); err == nil {
			return Value(floatV), nil
		} else {
			r, err := ParseValueString(v)
			if err != nil {
				return nil, err
			}
			return r.(ReadValuer), nil
		}
	case int:
		return Value(v), nil
	case float64:
		return Value(v), nil
	case Reader:
		return v, nil
	default:
		return nil, fmt.Errorf(`unpatchable source value %v (%T)`, v, v)
	}
}

// Inputs lists all registered inputs
func (io *IO) Inputs() map[string]*In {
	io.lazyInit()
	return io.ins
}

// Outputs lists all registered outputs
func (io *IO) Outputs() map[string]*Out {
	io.lazyInit()
	return io.outs
}

// Output realizes a registered output and returns it for patching
func (io *IO) Output(name string) (*Out, error) {
	io.lazyInit()
	name = canonicalPort(name)
	if o, ok := io.outs[name]; ok {
		if o.IsActive() {
			return nil, fmt.Errorf(`output "%s" is already patched`, name)
		}
		o.reader = o.Provider.Reader()
		return o, nil
	}
	return nil, fmt.Errorf(`output "%s" doesn't exist`, name)
}

// OutputsActive returns the total count of actively patched outputs
func (io *IO) OutputsActive() int {
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

// In is a module input
type In struct {
	Source  Reader
	Name    string
	initial Reader
}

// Read reads the output of the source into a Frame
func (i *In) Read(f Frame) {
	i.Source.Read(f)
}

// SetSource sets the internal source to some Reader
func (i *In) SetSource(r Reader) {
	switch v := i.Source.(type) {
	case SourceSetter:
		v.SetSource(r)
	case Reader:
		i.Source = r
	}
}

func (i *In) String() string {
	return fmt.Sprintf("%v", i.Source)
}

// ReadFrame reads an entire frame into the buffered input
func (i *In) ReadFrame() Frame {
	return i.Source.(*Buffer).ReadFrame()
}

// LastFrame returns the last frame read with ReadFrame
func (i *In) LastFrame() Frame {
	return i.Source.(*Buffer).Frame
}

// Close closes the input
func (i *In) Close() error {
	if c, ok := i.Source.(Closer); ok {
		return c.Close()
	}
	return nil
}

// Out is a module output
type Out struct {
	Name     string
	Provider ReaderProvider
	reader   Reader
}

// IsActive returns whether or not there is a realized Reader assigned
func (o *Out) IsActive() bool {
	return o.reader != nil
}

func (o *Out) Read(out Frame) {
	if o.reader != nil {
		o.reader.Read(out)
	}
}

func (o *Out) Close() error {
	defer func() {
		o.reader = nil
	}()
	if c, ok := o.reader.(Closer); ok {
		return c.Close()
	}
	return nil
}

// Port represents the address of a specific port on a Patcher
type Port struct {
	Patcher
	Port string
}

func canonicalPort(v string) string {
	return strings.Replace(v, "/", ".", -1)
}
