// Package module provides built-in modules
package module

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
)

var moduleSequence uint64

// Patcher is the patching behavior of a module
type Patcher interface {
	ID() string
	Patch(string, interface{}) error
	Output(string) (*Out, error)
	Reset() error
	ResetOnly([]string) error
	Close() error
}

// IO is the input/output registry of a module. It manages the lifecycles of the ports; fascilitating connects and
// disconnects between them. This struct lazy initializes so it is useful by default. It is intended to just be embedded
// inside other structs that represent a module.
type IO struct {
	id   string
	ins  map[string]*In
	outs map[string]*Out
}

// ID returns the module's unique identifier
func (io *IO) ID() string {
	return io.id
}

// Expose registers inputs and outputs of the module so that they can be used in patching
func (io *IO) Expose(name string, ins []*In, outs []*Out) error {
	io.id = fmt.Sprintf("%s:%d", name, atomic.LoadUint64(&moduleSequence))
	atomic.AddUint64(&moduleSequence, 1)

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
		in.owner = io
		io.ins[in.Name] = in
	}
	for _, out := range outs {
		if _, ok := io.outs[out.Name]; ok {
			return fmt.Errorf(`duplicate output exposed "%s"`, out.Name)
		}
		if out.Provider == nil {
			return fmt.Errorf(`provider must be set for output "%s"`, out.Name)
		}
		out.owner = io
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
	reader, err := assertReader(t)
	if err != nil {
		return err
	}

	input.SetSource(reader)
	if o, ok := reader.(*Out); ok {
		o.setDestination(input)
	}
	return nil
}

func assertReader(t interface{}) (Reader, error) {
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
		}
		r, err := ParseValueString(v)
		if err != nil {
			return nil, err
		}
		return r.(readValuer), nil
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
			return nil, fmt.Errorf(`%s: output "%s" is already patched`, io.ID(), name)
		}
		o.reader = o.Provider.Reader()
		return o, nil
	}
	return nil, fmt.Errorf(`%s: output "%s" doesn't exist`, io.ID(), name)
}

// OutputsActive returns the total count of actively patched outputs
func (io *IO) OutputsActive(sinking bool) int {
	io.lazyInit()
	var i int
	for _, out := range io.outs {
		if sinking {
			if out.IsActive() && out.IsSinking() {
				i++
			}
		} else if out.IsActive() {
			i++
		}
	}
	return i
}

func (io *IO) String() string {
	return io.ID()
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
		if err := in.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (io *IO) ResetOnly(names []string) error {
	for _, n := range names {
		if in, ok := io.ins[n]; ok {
			if err := in.Close(); err != nil {
				return err
			}
		} else {
			return fmt.Errorf(`unknown input "%s"`, n)
		}
	}
	return nil
}

// Close makes IO a noop io.Closer
func (io *IO) Close() error {
	return nil
}

// In is a module input
type In struct {
	Source       Reader
	Name         string
	ForceSinking bool
	initial      Reader
	owner        *IO
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

func (i *In) SourceName() string {
	if i.Source == nil {
		return "(none)"
	}
	return fmt.Sprintf("%s", i.Source)
}

func (i *In) String() string {
	return fmt.Sprintf("%s/%s", i.owner.ID(), i.Name)
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
	if c, ok := i.Source.(*In); ok {
		return c.Close()
	} else if c, ok := i.Source.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return err
		}
	}
	i.SetSource(i.initial)
	return nil
}

func (i *In) IsSinking() bool {
	if i == nil {
		return false
	}
	if i.ForceSinking {
		return true
	}
	return i.owner.OutputsActive(true) > 0
}

// Out is a module output
type Out struct {
	Name        string
	Provider    ReaderProvider
	reader      Reader
	destination *In
	owner       *IO
}

func (o *Out) String() string {
	return fmt.Sprintf("%s/%s", o.owner, o.Name)
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

func (o *Out) setDestination(in *In) {
	o.destination = in
}

func (o *Out) DestinationName() string {
	if o.destination == nil {
		return "(none)"
	}
	return fmt.Sprintf("%s", o.destination)
}

func (o *Out) IsSinking() bool {
	return o.destination.IsSinking()
}

// Close closes the output
func (o *Out) Close() error {
	defer func() {
		o.reader = nil
		o.destination = nil
	}()
	if c, ok := o.reader.(io.Closer); ok {
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
	return strings.Replace(v, ".", "/", -1)
}

type outputCounter interface {
	OutputsActive(sinking bool) int
}

type manyReadTracker struct {
	counter outputCounter
	reads   int
}

func (t *manyReadTracker) incr() {
	if outs := t.counter.OutputsActive(true); outs > 0 {
		t.reads = (t.reads + 1) % outs
	}
}

func (t *manyReadTracker) count() int {
	return t.reads
}

type trackedReader interface {
	readMany(Frame)
}

type manyOut struct {
	reader trackedReader
	cache  *Frame
}

func (o *manyOut) Read(out Frame) {
	o.reader.readMany(out)
	copy(out, *o.cache)
}
