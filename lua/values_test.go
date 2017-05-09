package lua

import (
	"testing"

	assert "gopkg.in/go-playground/assert.v1"
)

func TestValues(t *testing.T) {
	vm := newVM(t)
	defer vm.Close()

	err := vm.DoString(`
		local hertz = hz(440)
		assert(hertz:string() == '440.00Hz')
	`)
	assert.Equal(t, err, nil)

	err = vm.DoString(`
		local beatsPerMinute = bpm(100)
		assert(beatsPerMinute:string() == '100.00BPM')
	`)
	assert.Equal(t, err, nil)

	err = vm.DoString(`
		local milliseconds = ms(1000)
		assert(milliseconds:string() == '1000.00ms')
	`)
	assert.Equal(t, err, nil)

	err = vm.DoString(`
		local concertPitch = pitch('A4')
		assert(concertPitch:string() == 'A4')
	`)
	assert.Equal(t, err, nil)
}
