# Music Theory

[![Build Status](https://travis-ci.org/brettbuddin/musictheory.svg?branch=master)](https://travis-ci.org/brettbuddin/musictheory)
[![GoDoc](https://godoc.org/github.com/brettbuddin/musictheory?status.svg)](https://godoc.org/github.com/brettbuddin/musictheory)

Explorations in music theory.

## Usage

```go
package main

import (
	mt "buddin.us/musictheory"
)

func main() {
	root := mt.NewPitch(mt.C, mt.Natural, 4)

	root.Name(mt.AscNames) // C4
	root.Freq()            // 261.625565 (Hz)
	root.MIDI()            // 72

	P5 := mt.Perfect(5)   // Perfect 5th
	A4 := mt.Augmented(4) // Augmented 4th

	root.Transpose(P5).(mt.Pitch).Name(mt.AscNames)          // G4
	root.Transpose(A4).(mt.Pitch).Name(mt.AscNames)          // F#4
	root.Transpose(P5.Negate()).(mt.Pitch).Name(mt.AscNames) // F3

	mt.NewScale(root, mt.DorianIntervals, 1)
	// [C4, D4, Eb4, F4, G4, A4, Bb4]

	mt.NewScale(root, mt.MixolydianIntervals, 2)
	// [C4, D4, E4, F4, G4, A4, Bb4, C5, D5, E5, F5, G5, A5, Bb5]

    note := mt.NewNote(root, mt.D16) // C4 sixteenth note
    note.Time(mt.D4, 120)            // 125ms (quarter note getting the beat at 120 BPM)
}
```

楽しみます！
