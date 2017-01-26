package lua

import (
	"sync"
	"testing"

	"github.com/brettbuddin/eolian/module"

	assert "gopkg.in/go-playground/assert.v1"
)

func TestCreate(t *testing.T) {
	init, err := module.Lookup("Direct")
	assert.Equal(t, err, nil)
	direct, err := init(nil)

	vm, err := NewVM(direct, &sync.Mutex{})
	assert.Equal(t, err, nil)

	err = vm.REPL()
	assert.Equal(t, err, nil)

	err = vm.DoString(`
		local synth = require 'eolian.synth'

		-- input patching
		local direct = synth.Direct()
		direct:set { input = 1 }
		local mix = synth.Mix { size = 4 }
		mix:ns(0):set { input = 2 }
		mix:ns(1):set { input = direct:output() }
		mix:set(2, { input = 2 })
		mix:inspect()
		mix:close()

		-- value helpers
		hz(440)
		pitch('A4')
		ms(100)

		-- constructor parameters
		local interpolate = synth.Interpolate { min = 0, max = 10 }
		interpolate:reset()

		-- proxying
		local proxy = require 'eolian.synth.proxy'
		local directInputs = proxy.inputs(direct)
		local directOutputs = proxy.outputs(direct)
		directInputs(_, { inputs = 2 })
		directOutputs(_, nil)
		local osc = synth.Oscillator()
		directOutputs = proxy.outputs(osc)
		directOutputs(_, 'sine')
	`)
	assert.Equal(t, err, nil)

	err = vm.DoString(`
		local theory = require 'eolian.theory'

		local tonic = theory.pitch('C4')
		assert(tonic:transpose(theory.octave(1)):name() == 'C5', 'octave transposition failed')
		assert(tonic:transpose(theory.minor(2)):name() == 'C#4', 'minor transposition failed')
		assert(tonic:transpose(theory.major(3)):name() == 'E4', 'major transposition failed')
		assert(tonic:transpose(theory.perfect(5)):name() == 'G4', 'perfect transposition failed')
		assert(tonic:transpose(theory.augmented(4)):name() == 'F#4', 'augmented transposition failed')
		assert(tonic:transpose(theory.doublyAugmented(4)):name() == 'G4', 'doubly augmented transposition failed')
		assert(tonic:transpose(theory.diminished(6)):name() == 'G4', 'diminished transposition failed')
		assert(tonic:transpose(theory.doublyDiminished(6)):name() == 'F#4', 'doubly diminished transposition failed')
	`)
	assert.Equal(t, err, nil)
}
