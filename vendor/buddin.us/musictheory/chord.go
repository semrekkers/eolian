package musictheory

// Chord is a series of Pitches intended to be played at the same time
type Chord []Pitch

// NewChord returns a new Chord with a specific set of intervals
func NewChord(root Transposer, intervals []Interval) Chord {
	c := Chord{}
	for _, v := range intervals {
		c = append(c, root.Transpose(v).(Pitch))
	}
	return c
}

// Transpose transposes the Chord
func (c Chord) Transpose(i Interval) Transposer {
	chord := Chord{}
	for _, p := range c {
		chord = append(chord, p.Transpose(i).(Pitch))
	}
	return chord
}

// Invert performs a chord inversion of some degree
func (c Chord) Invert(degree int) Chord {
	octaves := degree / len(c)
	pos := degree % len(c)

	chord := Chord{}
	for _, p := range c[pos:] {
		if octaves > 0 {
			p = p.Transpose(Octave(octaves)).(Pitch)
		}
		chord = append(chord, p)
	}
	for _, p := range c[:pos] {
		chord = append(chord, p.Transpose(Octave(octaves+1)).(Pitch))
	}
	return chord
}
