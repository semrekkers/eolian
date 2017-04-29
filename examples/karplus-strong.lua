local synth = require 'eolian.synth'

return function(_)
    local function build()
        return {
            mix   = synth.Mix(),
            clock = synth.Clock(),
            ks = {
                noise  = synth.Noise(),
                adsr   = synth.ADSR(),
                amp    = synth.Multiply(),
                delay  = synth.FilteredFBDelay(),
                filter = synth.Filter(),
            },
            compress = synth.Compress(),
        }
    end

    local function patch(modules)
        -- Clock
        set(modules.clock, { tempo = hz(1) })

        -- Karplus-Strong
        with(modules.ks, function(t)
            set(t.adsr, {
                gate           = out(modules.clock),
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

        -- Mix
        set(modules.mix, {
            { input = out(modules.ks.filter, 'lowpass') },
        })
        set(modules.compress, { input = out(modules.mix) })

        return out(modules.compress)
    end

    return build, patch
end
