package dsp

// DCBlock is an operation that blocks DC signal to keep a signal centered around zero
type DCBlock struct {
	lastIn, lastOut Float64
}

// Tick advances the operation
func (dc *DCBlock) Tick(in Float64) Float64 {
	out := in - dc.lastIn + dc.lastOut*0.995
	dc.lastIn, dc.lastOut = in, out
	return out
}
