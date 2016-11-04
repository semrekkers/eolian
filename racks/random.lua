local pkg = {}

function pkg.build(self)
    local modules = {
        clock  = {
            osc      = synth.Osc(),
            multiple = synth.Multiple(),
        },

        random = {
            trigger = synth.ClockDivide(),
            noise  = synth.Noise(),
            series = synth.SeriesHold(),
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
        r.noise:set {
            max = pitch('C3'),
            min = pitch('C1'),
        }
        r.series:set {
            input   = r.noise:output(),
            clock   = modules.clock.multiple:output(1),
            trigger = r.trigger:output(),
            size    = 8,
        }
    end)

    modules.adsr:set {
        gate    = modules.clock.multiple:output(2),
        attack  = ms(50),
        decay   = ms(50),
        sustain = 0.4,
        release = ms(1000),
    }
    modules.osc:set {
        pitch = modules.random.series:output(),
    }

    modules.mix:scope(0):set { input = modules.osc:output('pulse') }
    modules.mix:scope(1):set { input = modules.osc:output('saw'), level = 0.5 }
    modules.amp:set {
        a = modules.mix:output(),
        b = modules.adsr:output(),
    }
end

return pkg
