package lua

import (
	"testing"

	assert "gopkg.in/go-playground/assert.v1"
)

func TestPitchTransposition(t *testing.T) {
	vm := newVM(t)
	defer vm.Close()
	err := vm.DoString(`
		local theory = require('eolian.theory')
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

func TestScaleTransposition(t *testing.T) {
	vm := newVM(t)
	defer vm.Close()
	err := vm.DoString(`
		local theory = require('eolian.theory')
		local dorian = theory.scale('C4', 'dorian', 1)

		local pitches = dorian:pitches()
		assert(dorian:size() == 7)
		assert(dorian:size() == #pitches)
		assert(pitches[1]:name() == 'C4')
		assert(pitches[2]:name() == 'D4')
		assert(pitches[3]:name() == 'D#4')
		assert(pitches[4]:name() == 'F4')
		assert(pitches[5]:name() == 'G4')
		assert(pitches[6]:name() == 'A4')
		assert(pitches[7]:name() == 'A#4')

		local transposed = dorian:transpose(theory.minor(2)):pitches()
		assert(transposed[1]:name() == 'C#4')
		assert(transposed[2]:name() == 'D#4')
		assert(transposed[3]:name() == 'E4')
		assert(transposed[4]:name() == 'F#4')
		assert(transposed[5]:name() == 'G#4')
		assert(transposed[6]:name() == 'A#4')
		assert(transposed[7]:name() == 'B4')
	`)
	assert.Equal(t, err, nil)
}

func TestChordTransposition(t *testing.T) {
	vm := newVM(t)
	defer vm.Close()
	err := vm.DoString(`
		local theory = require('eolian.theory')
		local triad  = theory.chord('C4', 'maj')

		local pitches = triad:pitches()
		assert(triad:size() == 3)
		assert(triad:size() == #pitches)
		assert(pitches[1]:name() == 'C4')
		assert(pitches[2]:name() == 'E4')
		assert(pitches[3]:name() == 'G4')

		local transposed = triad:transpose(theory.minor(2)):pitches()
		assert(transposed[1]:name() == 'C#4')
		assert(transposed[2]:name() == 'F4')
		assert(transposed[3]:name() == 'G#4')
	`)
	assert.Equal(t, err, nil)
}
