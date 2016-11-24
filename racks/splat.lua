local function set(inputs, module, whitelist)
    for _,v in ipairs(whitelist) do
        if inputs[v] ~= nil then
            module:set { [v] = inputs[v] }
        end
    end
end

local function splat()
    local envelope = synth.ADSR()
    local amp      = synth.Multiply()
    local osc      = synth.Osc()
    local filter   = synth.LPFilter()

    envelope:set {
        disableSustain = 1,
        attack         = ms(800),
        decay          = ms(100),
        sustain        = 0.1,
        release        = ms(4500),
    }
    amp:set {
        a = envelope:output(),
        b = hz(4000),
    }
    filter:set {
        input     = osc:output('pulse'),
        cutoff    = amp:output(),
        resonance = 0,
    }

    return {
        set = function(_, inputs)
            set(inputs, envelope, {
                'gate',
                'attack',
                'decay',
                'sustain',
                'release'
            })
            set(inputs, osc, {'pitch'})
            set(inputs, filter, {'cutoff', 'resonance'})
        end,
        output = function()
            return filter:output()
        end
    }
end

local rack = {}

function rack.build(self)
    return {
        clock = {
            osc      = synth.Osc(),
            multiple = synth.Multiple(),
        },

        multiplier  = synth.ClockMultiply(),
        splat1      = splat(),
        splat2      = splat(),
        splat3      = splat(),
        delay       = synth.FBComb(),

        noise      = synth.Noise(),
        mixer      = synth.Mix(),
        compressor = synth.Compress(),
        clipper    = synth.Clip(),
    }
end

function rack.patch(self, modules)
    with(modules.clock, function(c)
        c.osc:set      { pitch = hz(0.25) }
        c.multiple:set { input = c.osc:output('pulse') }
    end)

    modules.splat1:set {
        gate  = modules.clock.multiple:output(0),
        pitch = pitch('C3')
    }

    modules.splat2:set {
        gate  = modules.clock.multiple:output(1),
        pitch = pitch('C2')
    }

    modules.splat3:set {
        gate  = modules.clock.multiple:output(2),
        pitch = pitch('C1')
    }

    modules.mixer:scope(0):set { input = modules.splat1:output() }
    modules.mixer:scope(1):set { input = modules.splat2:output() }
    modules.mixer:scope(2):set { input = modules.splat3:output() }

    modules.delay:set {
        input    = modules.mixer:output(),
        duration = 0.1,
        gain     = 0.5,
    }

    modules.noise:set { input = modules.delay:output(), max = 0.002 }
    modules.compressor:set {
        input = modules.noise:output(),
    }
    modules.clipper:set {
        input = modules.compressor:output(),
        max   = 2,
    }
    return modules.clipper:output()
end

return rack
