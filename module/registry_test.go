package module

import (
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

func TestRegister(t *testing.T) {
	name := "UltraMegaSuperCrusher"
	expected := &mockPatcher{}

	Register(name, func(c Config) (Patcher, error) {
		expected.value = c["key"].(string)
		return expected, nil
	})

	init, err := Lookup(name)
	assert.Equal(t, err, nil)

	p, err := init(Config{"key": "hello"})
	p.SetID("WizzBang")

	assert.Equal(t, err, nil)
	assert.Equal(t, expected, p)
	assert.Equal(t, p.ID(), "WizzBang")

	_, err = Lookup("unknown")
	assert.NotEqual(t, err, nil)
}

type mockPatcher struct {
	IO
	value string
}
