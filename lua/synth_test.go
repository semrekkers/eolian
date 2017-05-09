package lua

import (
	"testing"

	assert "gopkg.in/go-playground/assert.v1"
)

func TestSynth(t *testing.T) {
	vm := newVM(t)
	defer vm.Close()

	err := vm.DoString(`
		local synth  = require('eolian.synth')

		-- Multiple inputs
		local osc = synth.Oscillator()
		osc:set { pitch = hz(100), detune = hz(1) }

		-- Single input
		local direct = synth.Direct()
		direct:set('input', osc:out('sine'))

		local mix = synth.Mix { size = 4 }

		-- Namespaces
		mix:ns(1):set { input = direct:out() }
		mix:ns(1):set('input', direct:out())
		mix:set { { input = direct:out() } }

		mix:close()
	`)
	assert.Equal(t, err, nil)
}
