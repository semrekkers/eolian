local pkg = {}

function pkg.build(self)
    local modules = {
        clock  = {
            osc      = synth.Osc(),
            multiple = synth.Multiple(),
        },
        random = {
            trigger = synth.ClockDivide(),
            series  = synth.RandomSeries(),
            quant   = synth.Quantize(),
        },


        voice = {
            adsr = synth.ADSR(),
            osc  = synth.Osc(),
            mix  = synth.Mix(),
            amp  = synth.Multiply(),
        },

        delay = {
            cutoff = synth.Osc(),
            gain   = synth.Osc(),
            delay  = synth.FilteredDelay(),
        },
    }
    return {
        modules = modules,
        output = function()
            return modules.delay.delay:output()
        end
    }
end

function pkg.patch(self, modules)
    with(modules.clock, function(c)
        c.osc:set      { pitch = hz(9) }
        c.multiple:set { input = c.osc:output('pulse') }
    end)

    with(modules.random, function(r)
        r.trigger:set {
            input   = modules.clock.multiple:output(0),
            divisor = 32,
        }
        r.series:set {
            clock   = modules.clock.multiple:output(1),
            trigger = r.trigger:output(),
            size    = 8,
        }
        r.quant:set { input = r.series:output('values') }

        -- Cmin Penatonic
        r.quant:scope(0):set { pitch = pitch('C3') }
        r.quant:scope(1):set { pitch = pitch('Eb3') }
        r.quant:scope(2):set { pitch = pitch('F3') }
        r.quant:scope(3):set { pitch = pitch('G3') }
        r.quant:scope(4):set { pitch = pitch('Bb3') }
        r.quant:scope(5):set { pitch = pitch('C4') }
        r.quant:scope(6):set { pitch = pitch('Eb4') }
        r.quant:scope(7):set { pitch = pitch('F4') }
        r.quant:scope(8):set { pitch = pitch('G4') }
        r.quant:scope(9):set { pitch = pitch('Bb4') }
    end)

    with(modules.voice, function(v)
        v.adsr:set {
            gate    = modules.clock.multiple:output(2),
            attack  = ms(30),
            decay   = ms(50),
            sustain = 0.2,
            release = ms(1000),
        }

        v.osc:set {
            pitch = modules.random.quant:output(),
        }
        v.mix:scope(0):set { input = v.osc:output('pulse') }
        v.mix:scope(1):set { input = v.osc:output('saw'), level = 0.5 }

        v.amp:set {
            a = v.mix:output(),
            b = v.adsr:output(),
        }
    end)

    with(modules.delay, function(d)
        d.cutoff:set {
            pitch = hz(0.1),
            amp   = hz(5000),
        }
        d.gain:set {
            pitch  = hz(0.2),
            amp    = 0.2,
            offset = 0.7
        }
        d.delay:set {
            input  = modules.voice.amp:output(),
            gain   = d.gain:output('sine'),
            cutoff = d.cutoff:output('sine'),
        }
    end)
end

return pkg
