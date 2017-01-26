package module

type dcBlock struct {
	lastIn, lastOut Value
}

func (dc *dcBlock) Tick(in Value) Value {
	out := in - dc.lastIn + dc.lastOut*0.995
	dc.lastIn = in
	dc.lastOut = out
	return out
}
