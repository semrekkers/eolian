return function(env)
    local synth    = require('eolian.synth')
    local supersaw = env:require('voice/supersaw.lua')

    local function build()
        return {
            mix = synth.Mix(),
            clock = {
                osc      = synth.Osc(),
                multiple = synth.Multiple(),
            },
            high = {
                divider  = synth.ClockDivide(),
                sequence = synth.Sequence(),
                adsr     = synth.ADSR(),
                supersaw = supersaw(3),
                lfo      = synth.Osc(),
                mix      = synth.Mix(),
                filter   = synth.Filter(),
                amp      = synth.Multiply()
            },
            low = {
                divider  = synth.ClockDivide(),
                sequence = synth.Sequence(),
                pitch    = synth.Multiple(),
                osc      = synth.Osc(),
                subPitch = synth.Multiply(),
                subOsc   = synth.Osc(),
                fold     = synth.Fold(),
                mix      = synth.Mix(),
                filter   = synth.Filter()
            },
            glitch = {
                multiplier = synth.ClockMultiply(),
                sequence   = synth.Sequence(),
                osc        = synth.Osc(),
                adsr       = synth.ADSR(),
                amp        = synth.Multiply(),
            },
            effects = {
                reverb = synth.FilteredReverb(),
                filter = synth.Filter(),
                noise  = synth.Noise()
            },
            compressor = synth.Compress()
        }
    end

    local function patch(modules)
        --
        -- Clock
        --
        with(modules.clock, function(t)
            t.osc:set { pitch = hz(7) }
            t.multiple:set { input = t.osc:out('pulse') }
        end)

        --
        -- High
        --
        with(modules.high, function(t)
            t.sequence:set { 
                clock = modules.clock.multiple:out(0), 
                mode = 1,
                glide = ms(30)
            }
            t.sequence:set {
                { pitch = pitch('C2'), pulses = 2, mode = 2, glide  = 0 },
                { pitch = pitch('C3'), pulses = 2, mode = 2, glide  = 0 },
                { pitch = pitch('C2'), pulses = 1, mode = 2, glide  = 0 },
                { pitch = pitch('C3'), pulses = 2, mode = 2, glide  = 1 },
                { pitch = pitch('C4'), pulses = 2, mode = 2, glide  = 1 },
                { pitch = pitch('C3'), pulses = 1, mode = 2, glide  = 0 },
                { pitch = pitch('C2'), pulses = 2, mode = 2, glide  = 0 },
                { pitch = pitch('C3'), pulses = 1, mode = 2, glide  = 1 },
            }

            t.adsr:set {
                gate    = t.sequence:out('gate'),
                attack  = ms(10),
                decay   = ms(70),
                sustain = 0.2,
                release = ms(50)
            }

            t.lfo:set {
                pitch  = hz(0.05),
                amp    = 0.000001,
                offset = 0.000001
            }

            t.supersaw:set {
                pitch  = t.sequence:out('pitch'),
                spread = t.lfo:out('sine')
            }

            t.filter:set {
                input  = t.supersaw:out(),
                cutoff = hz(4000)
            }

            t.amp:set {
                a = t.filter:out('lowpass'),
                b = t.adsr:out()
            }
        end)

        --
        -- Low
        --
        with(modules.low, function(t)
            t.divider:set { 
                input = modules.clock.multiple:out(1), 
                divisor = 16 
            }

            t.sequence:set { clock = t.divider:out() }
            t.sequence:set {
                { pitch = pitch('C2') },
                { pitch = pitch('C2') },
                { pitch = pitch('F2') },
                { pitch = pitch('C2') },
                { pitch = pitch('C2') },
                { pitch = pitch('C2') },
                { pitch = pitch('F2') },
                { pitch = pitch('G2') },
            }

            t.pitch:set { input = t.sequence:out('pitch') }

            t.osc:set { pitch = t.pitch:out(0) }
            t.fold:set { 
                input = t.osc:out('sine'), 
                level = 0.7
            }

            t.subPitch:set { 
                a = t.pitch:out(1), 
                b = 0.5 
            }
            t.subOsc:set { pitch = t.subPitch:out() }

            t.mix:set {
                { input = t.fold:out() },
                { input = t.subOsc:out('saw'), level = 0.7 },
            }

            t.filter:set { input = t.mix:out(), cutoff = hz(3000) }
        end)

        --
        -- Glitch
        --
        with(modules.glitch, function(t)
            t.sequence:set { clock = modules.clock.multiple:out(2), mode = 2 }
            t.sequence:set {
                { pitch = pitch('C5'), pulses = 1, mode = 2 },
                { pitch = pitch('C6'), pulses = 1, mode = 2 },
                { pitch = pitch('C5'), pulses = 1, mode = 2 },
                { pitch = pitch('C8'), pulses = 1, mode = 2 },
                { pitch = pitch('C6'), pulses = 1, mode = 2 },
                { pitch = pitch('C5'), pulses = 1, mode = 2 },
                { pitch = pitch('C3'), pulses = 1, mode = 2 },
                { pitch = pitch('C7'), pulses = 1, mode = 2 },
            }

            t.multiplier:set { input = t.sequence:out('gate'), multiplier = 1 }

            t.adsr:set {
                gate = t.multiplier:out(),
                attack = ms(1),
                decay = ms(1),
                sustain = 0
            }

            t.osc:set {
                pitch = t.sequence:out('pitch')
            }

            t.amp:set {
                a = t.osc:out('pulse'),
                b = t.adsr:out()
            }
        end)

        --
        -- Mix
        --
        modules.mix:set {
            { input = modules.high.amp:out(), level = 0.5 },
            { input = modules.low.filter:out('lowpass'), level = 0.5 },
            { input = modules.glitch.amp:out(), level = 0.05 },
        }

        --
        -- Effects
        --
        with(modules.effects, function(t)
            t.reverb:set { 
                input    = modules.mix:out(),
                cutoff   = hz(500),
                feedback = 0.84,
                gain     = 0.5,
                bias     = -0.95
            }

            t.filter:set { input = t.reverb:out(), cutoff = hz(5000) }
            t.noise:set { input = t.filter:out('lowpass'), gain = 0.02 }
        end)

        modules.compressor:set { input = modules.effects.noise:out() }

        return modules.compressor:out()
    end

    return build, patch
end
