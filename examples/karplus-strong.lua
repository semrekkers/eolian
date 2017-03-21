return function(env)
    local synth = require 'eolian.synth'

    local function build()
        return {
            mix = synth.Mix(),
            clock = {
                osc      = synth.Osc(),
                multiple = synth.Multiple(),
            },
            ks = {
                noise  = synth.Noise(),
                adsr   = synth.ADSR(),
                amp    = synth.Multiply(),
                delay  = synth.FilteredFBComb(),
                filter = synth.Filter(),
            },
            compress = synth.Compress(),
            sink = synth.Multiple { size = 2 },
        }
    end

    local function patch(modules)
        --
        -- Clock
        --
        with(modules.clock, function(t)
            set(t.osc, { pitch = hz(1) })
            set(t.multiple, { input = out(t.osc, 'pulse') })
        end)

        --
        -- Karplus-Strong
        --
        with(modules.ks, function(t)
            set(t.adsr, { 
                gate           = out(modules.clock.multiple, 0),
                disableSustain = 1,
                attack         = ms(2),
                decay          = ms(5),
                sustain        = 0
            })
            set(t.amp, { 
                a = out(t.adsr), 
                b = out(t.noise)
            })
            set(t.delay, {
                input    = out(t.amp),
                gain     = 0.8,
                duration = ms(20),
                cutoff   = hz(5000),
            })
            set(t.filter, {
                input  = out(t.delay),
                cutoff = hz(2500)
            })
        end)

        --
        -- Mix
        -- 
        set(modules.mix, {
            { input = out(modules.ks.filter, 'lowpass') },
        })
        set(modules.compress, { input = out(modules.mix) })
        set(modules.sink, { input = out(modules.compress) })

        return out(modules.sink, 0), out(modules.sink, 1)
    end

    return build, patch
end
