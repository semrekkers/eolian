// Package module provides built-in modules
package module

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"

	"buddin.us/eolian/dsp"
)

var moduleSequence uint64

// Patcher is the patching behavior of a module
type Patcher interface {
	Identifier
	PortLister
	Resetter
	io.Closer

	Patch(string, interface{}) error
	Output(string) (*Out, error)
}

// Identifier returns the identity and type of a module
type Identifier interface {
	ID() string
	Type() string
}

// Resetter resets inputs back to their normal values
type Resetter interface {
	Reset() error
	ResetOnly([]string) error
}

// PortLister lists inputs and outputs of a module
type PortLister interface {
	Inputs() map[string]*In
	Outputs() map[string]*Out
}

// IO is the input/output registry of a module. It manages the lifecycles of the ports; fascilitating connects and
// disconnects between them. This struct lazy initializes so it is useful by default. It is intended to just be embedded
// inside other structs that represent a module.
type IO struct {
	id   string
	typ  string
	ins  map[string]*In
	outs map[string]*Out

	forcedActiveOutputs int
}

// ID returns the module's unique identifier
func (io *IO) ID() string {
	return io.id
}

// Type returns the module's type
func (io *IO) Type() string {
	return io.typ
}

// Expose registers inputs and outputs of the module so that they can be used in patching
func (io *IO) Expose(typ string, ins []*In, outs []*Out) error {
	io.typ = typ
	io.id = fmt.Sprintf("%s:%d", typ, atomic.LoadUint64(&moduleSequence))
	atomic.AddUint64(&moduleSequence, 1)

	io.lazyInit()
	for _, in := range ins {
		if err := io.AddInput(in); err != nil {
			return err
		}
	}
	for _, out := range outs {
		if err := io.AddOutput(out); err != nil {
			return err
		}
	}
	return nil
}

// AddInput registers a new input with the module. This is primarily used to allow for lazy-creation of inputs when
// patched instead of at the module's create-time.
func (io *IO) AddInput(in *In) error {
	if _, ok := io.ins[in.Name]; ok {
		return fmt.Errorf(`duplicate input exposed "%s"`, in.Name)
	}

	if b, ok := in.Source.(*dsp.Buffer); ok {
		in.initial = b.Processor
	} else {
		in.initial = in.Source
	}
	in.owner = io
	io.ins[in.Name] = in
	return nil
}

// AddOutput registers a new output with the module. Like AddInput, it is used for lazy-creation of outputs.
func (io *IO) AddOutput(out *Out) error {
	if _, ok := io.outs[out.Name]; ok {
		return fmt.Errorf(`duplicate output exposed "%s"`, out.Name)
	}
	if out.Provider == nil {
		return fmt.Errorf(`provider must be set for output "%s"`, out.Name)
	}
	out.owner = io
	io.outs[out.Name] = out
	return nil
}

// Patch assigns an input's processor to some source (Processor, Value, etc)
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
	processor, err := assertProcessor(t)
	if err != nil {
		return err
	}

	input.SetSource(processor)
	if o, ok := processor.(*Out); ok {
		o.addDestination(input)
	}
	return nil
}

func assertProcessor(t interface{}) (dsp.Processor, error) {
	switch v := t.(type) {
	case Port:
		return v.Patcher.Output(v.Port)
	case dsp.Float64:
		return v, nil
	case dsp.Pitch:
		return v, nil
	case dsp.Hz:
		return v, nil
	case dsp.MS:
		return v, nil
	case string:
		if floatV, err := strconv.ParseFloat(v, 64); err == nil {
			return dsp.Float64(floatV), nil
		}
		r, err := dsp.ParseValueString(v)
		if err != nil {
			return nil, err
		}
		return r.(dsp.ProcessValuer), nil
	case int:
		return dsp.Float64(v), nil
	case float64:
		return dsp.Float64(v), nil
	case dsp.Processor:
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
		if !o.IsActive() {
			o.buffer = dsp.NewBuffer(o.Provider.Processor())
		}
		return o, nil
	}
	return nil, fmt.Errorf(`%s: output "%s" doesn't exist`, io.ID(), name)
}

// OutputsActive returns the total count of actively patched outputs
func (io *IO) OutputsActive(sinking bool) int {
	io.lazyInit()

	if io.forcedActiveOutputs != 0 {
		return io.forcedActiveOutputs
	}

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

// ResetOnly disconnects specific inputs from their sources
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
	Source       dsp.Processor
	Name         string
	ForceSinking bool
	initial      dsp.Processor
	owner        *IO
}

// NewIn returns a new unbuffered input
func NewIn(name string, v dsp.Processor) *In {
	return &In{Name: name, Source: v}
}

// NewInBuffer returns a new buffered input
func NewInBuffer(name string, v dsp.Processor) *In {
	return &In{Name: name, Source: dsp.NewBuffer(v)}
}

// Process reads the output of the source into a Frame
func (i *In) Process(f dsp.Frame) {
	i.Source.Process(f)
}

// SetSource sets the internal source to some Processor
func (i *In) SetSource(r dsp.Processor) {
	switch v := i.Source.(type) {
	case dsp.SourceSetter:
		v.SetSource(r)
	case dsp.Processor:
		i.Source = r
	}
}

// SourceName returns the name of the connected output
func (i *In) SourceName() string {
	if i.Source == nil {
		return "(none)"
	}
	if in, ok := i.Source.(*In); ok {
		return fmt.Sprintf("%s", in.Source)
	}
	return fmt.Sprintf("%s", i.Source)
}

func (i *In) String() string {
	if in, ok := i.Source.(*In); ok {
		return fmt.Sprintf("%s/%s", in.owner.ID(), i.Name)
	}
	return fmt.Sprintf("%s/%s", i.owner.ID(), i.Name)
}

// ProcessFrame reads an entire frame into the buffered input
func (i *In) ProcessFrame() dsp.Frame {
	return i.Source.(*dsp.Buffer).ProcessFrame()
}

// LastFrame returns the last frame read with ProcessFrame
func (i *In) LastFrame() dsp.Frame {
	return i.Source.(*dsp.Buffer).Frame
}

func (i *In) Normalize() {
	i.SetSource(i.initial)
}

func (i *In) removeFromOutput() {
	switch v := i.Source.(type) {
	case *dsp.Buffer:
		if o, ok := v.Processor.(*Out); ok {
			o.removeDestination(i)
		}
	case *Out:
		v.removeDestination(i)
	}
}

type releaser interface {
	release(*In) error
}

// Close closes the input
func (i *In) Close() error {
	i.removeFromOutput()

	var err error
	switch v := i.Source.(type) {
	case *dsp.Buffer:
		if r, ok := v.Processor.(releaser); ok {
			err = r.release(i)
		} else {
			err = v.Close()
		}
	case releaser:
		err = v.release(i)
	case io.Closer:
		err = v.Close()
	}

	i.Normalize()
	return err
}

// IsSinking returns whether the input is sinking to audio output
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
	Name         string
	Provider     dsp.ProcessorProvider
	buffer       *dsp.Buffer
	destinations []*In
	owner        *IO
	reads        int
}

func (o *Out) String() string {
	return fmt.Sprintf("%s/%s", o.owner, o.Name)
}

// IsActive returns whether or not there is a realized Processor assigned
func (o *Out) IsActive() bool {
	return o.buffer != nil
}

// Process proxies to the internal processor if its set
func (o *Out) Process(out dsp.Frame) {
	if o.buffer == nil {
		return
	}

	if len(o.destinations) == 1 {
		o.buffer.Process(out)
		return
	}

	if o.reads == 0 {
		o.buffer.Process(out)
		copy(o.buffer.Frame, out)
	} else {
		copy(out, o.buffer.Frame)
	}
	var sinking int
	for _, d := range o.destinations {
		if d.IsSinking() {
			sinking++
		}
	}
	if sinking > 0 {
		o.reads = (o.reads + 1) % sinking
	}
}

func (o *Out) addDestination(in *In) {
	o.destinations = append(o.destinations, in)
}

func (o *Out) removeDestination(in *In) {
	filtered := []*In{}
	for _, d := range o.destinations {
		if d != in {
			filtered = append(filtered, d)
		}
	}
	o.destinations = filtered
}

// DestinationNames returns the name of the destination input
func (o *Out) DestinationNames() []string {
	names := []string{}
	for _, d := range o.destinations {
		names = append(names, d.String())
	}
	return names
}

// IsSinking returns whether the output is sinking to audio output
func (o *Out) IsSinking() bool {
	var sinking bool
	for _, d := range o.destinations {
		if d.IsSinking() {
			sinking = true
		}
	}
	return sinking
}

func (o *Out) release(i *In) error {
	filtered := o.destinations[:0]
	for _, d := range o.destinations {
		if d != i {
			filtered = append(filtered, d)
		}
	}
	o.destinations = filtered
	if len(o.destinations) == 0 {
		return o.Close()
	}
	return nil
}

// Close closes the output
func (o *Out) Close() error {
	var err error
	if o.buffer != nil {
		err = o.buffer.Close()
	}

	o.buffer = nil
	o.destinations = nil
	o.reads = 0

	return err
}

// Port represents the address of a specific port on a Patcher
type Port struct {
	Patcher
	Port string
}

func canonicalPort(v string) string {
	return strings.Replace(v, ".", "/", -1)
}

// LuaMethod is a function exposed to the Lua layer. If Lock is true, the synthesizer module graph will be locked when
// it is called to prevent race conditions.
type LuaMethod struct {
	Lock bool
	Func interface{}
}
