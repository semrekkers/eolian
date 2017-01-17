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
                filter = synth.LPFilter(),
            },
            compress = synth.Compress(),
        }
    end

    local function patch(modules)
        --
        -- Clock
        --
        with(modules.clock, function(t)
            t.osc:set { pitch = hz(1) }
            t.multiple:set { input = t.osc:output('pulse') }
        end)

        --
        -- Karplus-Strong
        --
        with(modules.ks, function(t)
            t.adsr:set { 
                gate           = modules.clock.multiple:output(0),
                disableSustain = 1,
                attack         = ms(5),
                decay          = ms(10),
                sustain        = 0
            }
            t.amp:set { 
                a = t.adsr:output(), 
                b = t.noise:output()
            }
            t.delay:set {
                input = t.amp:output(),
                duration = 0.07,
                cutoff = hz(3000)
            }
            t.filter:set {
                input  = t.delay:output(),
                cutoff = hz(2500)
            }
        end)

        --
        -- Mix
        -- 
        modules.mix:scope(0):set { input = modules.ks.filter:output() }
        modules.compress:set { input = modules.mix:output() }

        return modules.compress:output()
    end

    return build, patch
end
