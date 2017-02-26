package module

import (
	"testing"

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

	err = two.Unpatch("input")
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

	err = one.Patch("input", Duration(200))
	assert.Equal(t, err, nil)

	err = one.Patch("input", Frequency(440))
	assert.Equal(t, err, nil)

	err = one.Patch("input", "C#4")
	assert.Equal(t, err, nil)

	err = one.Patch("input", "Z4")
	assert.NotEqual(t, err, nil)

	pitch, err := ParsePitch("Eb3")
	assert.Equal(t, err, nil)
	err = one.Patch("input", pitch)
	assert.Equal(t, err, nil)

	err = one.Patch("input", true)
	assert.NotEqual(t, err, nil)
}

type mockOutput struct{}

func (p mockOutput) Read(Frame) {}

func newModule(forceSinking bool) (*IO, error) {
	io := &IO{}
	if err := io.Expose(
		"Module",
		[]*In{
			{Name: "input", Source: NewBuffer(zero), ForceSinking: forceSinking},
			{Name: "level", Source: NewBuffer(zero)},
		},
		[]*Out{{Name: "output", Provider: Provide(mockOutput{})}},
	); err != nil {
		return nil, err
	}
	return io, nil
}
