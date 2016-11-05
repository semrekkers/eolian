local pkg = {}

function pkg.build(self)
    local modules = {
        clock  = {
            osc      = synth.Osc(),
            multiple = synth.Multiple(),
        },

        random = {
            trigger = synth.ClockDivide(),
            series = synth.RandomSeries(),
            quant  = synth.Quantize { size = 6 },
        },

        adsr   = synth.ADSR(),
        osc    = synth.Osc(),
        mix    = synth.Mix(),
        amp    = synth.BinaryMultiply(),
    }
    return {
        modules = modules,
        output = function()
            return modules.amp:output()
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
            max     = 1,
            min     = 0,
        }
        r.quant:set { input = r.series:output('values') }
        r.quant:scope(0):set { pitch = pitch('C2') }
        r.quant:scope(1):set { pitch = pitch('Eb2') }
        r.quant:scope(2):set { pitch = pitch('F2') }
        r.quant:scope(3):set { pitch = pitch('G2') }
        r.quant:scope(4):set { pitch = pitch('Bb2') }
        r.quant:scope(5):set { pitch = pitch('C3') }
    end)

    modules.adsr:set {
        gate    = modules.random.series:output('gate'),
        attack  = ms(50),
        decay   = ms(50),
        sustain = 0.1,
        release = ms(300),
    }
    modules.osc:set {
        pitch = modules.random.quant:output(),
    }

    modules.mix:scope(0):set { input = modules.osc:output('pulse') }
    modules.mix:scope(1):set { input = modules.osc:output('saw'), level = 0.5 }
    modules.amp:set {
        a = modules.mix:output(),
        b = modules.adsr:output(),
    }
end

return pkg
