package module

type DCBlock struct {
	lastIn, lastOut Value
}

func (dc *DCBlock) Tick(in Value) Value {
	out := in - dc.lastIn + dc.lastOut*0.995
	dc.lastIn = in
	dc.lastOut = out
	return out
}
