package dsp

type Decimate struct {
	count, last Float64
}

func (d *Decimate) Tick(in, rate, bits Float64) Float64 {
	var (
		step, stepRatio, ratio Float64
	)
	if bits >= 31 || bits < 0 {
		step = 0
		stepRatio = 1
	} else {
		step = Pow(0.5, bits-0.999)
		stepRatio = 1 / step
	}

	if rate >= SampleRate {
		ratio = 1
	} else {
		ratio = rate / SampleRate
	}

	d.count += ratio
	if d.count >= 1 {
		d.count -= 1
		var x Float64 = 1
		if in < 0 {
			x = -1
		}
		_, frac := Modf((in + x*step*0.5) * stepRatio)
		delta := frac * step
		d.last = in - delta
		return d.last
	}
	return d.last
}
