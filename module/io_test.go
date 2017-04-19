package module

import (
	"fmt"
	"testing"

	"buddin.us/eolian/dsp"

	"gopkg.in/go-playground/assert.v1"
)

func TestPatching(t *testing.T) {
	one, err := newModule(false)
	assert.Equal(t, err, nil)

	two, err := newModule(true)
	assert.Equal(t, err, nil)

	err = two.Patch("input", Port{one, "output"})
	assert.Equal(t, err, nil)

	actual, expected := one.OutputsActive(true), 1
	assert.Equal(t, actual, expected)

	err = two.Reset()
	assert.Equal(t, err, nil)

	actual, expected = one.OutputsActive(true), 0
	assert.Equal(t, actual, expected)
}

func TestPatchingUnknownPort(t *testing.T) {
	one, err := newModule(false)
	assert.Equal(t, err, nil)

	two, err := newModule(true)
	assert.Equal(t, err, nil)

	err = two.Patch("input", Port{one, "unknown"})
	assert.NotEqual(t, err, nil)

	actual, expected := one.OutputsActive(true), 0
	assert.Equal(t, actual, expected)
}

func TestListing(t *testing.T) {
	module, err := newModule(false)
	assert.Equal(t, err, nil)

	assert.Equal(t, module.Inputs(), module.ins)
	assert.Equal(t, len(module.Inputs()), 2)
	assert.Equal(t, module.Outputs(), module.outs)
	assert.Equal(t, len(module.Outputs()), 1)
}

func TestPatchingValues(t *testing.T) {
	one, err := newModule(false)
	assert.Equal(t, err, nil)

	err = one.Patch("input", 1)
	assert.Equal(t, err, nil)

	err = one.Patch("input", "1")
	assert.Equal(t, err, nil)

	err = one.Patch("input", 1.0)
	assert.Equal(t, err, nil)

	err = one.Patch("input", "1.0")
	assert.Equal(t, err, nil)

	err = one.Patch("input", dsp.Duration(200))
	assert.Equal(t, err, nil)

	err = one.Patch("input", dsp.Frequency(440))
	assert.Equal(t, err, nil)

	err = one.Patch("input", "C#4")
	assert.Equal(t, err, nil)

	err = one.Patch("input", "Z4")
	assert.NotEqual(t, err, nil)

	pitch, err := dsp.ParsePitch("Eb3")
	assert.Equal(t, err, nil)
	err = one.Patch("input", pitch)
	assert.Equal(t, err, nil)

	err = one.Patch("input", true)
	assert.NotEqual(t, err, nil)
}

type mockOutput struct{}

func (p mockOutput) Process(dsp.Frame) {}

func newModule(forceSinking bool) (*IO, error) {
	io := &IO{}
	if err := io.Expose(
		"Module",
		[]*In{
			{Name: "input", Source: dsp.NewBuffer(dsp.Float64(0)), ForceSinking: forceSinking},
			{Name: "level", Source: dsp.NewBuffer(dsp.Float64(0))},
		},
		[]*Out{{Name: "output", Provider: dsp.Provide(mockOutput{})}},
	); err != nil {
		return nil, err
	}
	return io, nil
}

func TestMultipleOutputDestinations(t *testing.T) {
	one, err := newModule(false)
	assert.Equal(t, err, nil)

	two, err := newModule(false)
	assert.Equal(t, err, nil)

	three, err := newModule(true)
	assert.Equal(t, err, nil)

	err = two.Patch("input", Port{one, "output"})
	assert.Equal(t, err, nil)

	err = three.Patch("input", Port{one, "output"})
	assert.Equal(t, err, nil)

	actual, expected := one.OutputsActive(true), 1
	assert.Equal(t, actual, expected)

	actual, expected = two.OutputsActive(true), 0
	assert.Equal(t, actual, expected)

	actual, expected = three.OutputsActive(true), 0
	assert.Equal(t, actual, expected)

	o, _ := one.Output("output")
	fmt.Println(o.destinations)

	err = two.Reset()
	assert.Equal(t, err, nil)

	actual, expected = one.OutputsActive(true), 0
	assert.Equal(t, actual, expected)

	o, _ = one.Output("output")
	fmt.Println(o.destinations)

	fmt.Println(three.ins["input"].Source)
}
