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
                lpf      = synth.Filter(),
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
                lpf      = synth.Filter()
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
                lpf    = synth.Filter(),
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
            t.multiple:set { input = t.osc:output('pulse') }
        end)

        --
        -- High
        --
        with(modules.high, function(t)
            t.sequence:set { 
                clock = modules.clock.multiple:output(0), 
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
                gate    = t.sequence:output('gate'),
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
                pitch  = t.sequence:output('pitch'),
                spread = t.lfo:output('sine')
            }

            t.lpf:set {
                input  = t.supersaw:output(),
                cutoff = hz(4000)
            }

            t.amp:set {
                a = t.lpf:output(),
                b = t.adsr:output()
            }
        end)

        --
        -- Low
        --
        with(modules.low, function(t)
            t.divider:set { 
                input = modules.clock.multiple:output(1), 
                divisor = 16 
            }

            t.sequence:set { clock = t.divider:output() }
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

            t.pitch:set { input = t.sequence:output('pitch') }

            t.osc:set { pitch = t.pitch:output(0) }
            t.fold:set { 
                input = t.osc:output('sine'), 
                level = 0.7
            }

            t.subPitch:set { 
                a = t.pitch:output(1), 
                b = 0.5 
            }
            t.subOsc:set { pitch = t.subPitch:output() }

            t.mix:set {
                { input = t.fold:output() },
                { input = t.subOsc:output('saw'), level = 0.7 },
            }

            t.lpf:set { input = t.mix:output(), cutoff = hz(3000) }
        end)

        --
        -- Glitch
        --
        with(modules.glitch, function(t)
            t.sequence:set { clock = modules.clock.multiple:output(2), mode = 2 }
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

            t.multiplier:set { input = t.sequence:output('gate'), multiplier = 1 }

            t.adsr:set {
                gate = t.multiplier:output(),
                attack = ms(1),
                decay = ms(1),
                sustain = 0
            }

            t.osc:set {
                pitch = t.sequence:output('pitch')
            }

            t.amp:set {
                a = t.osc:output('pulse'),
                b = t.adsr:output()
            }
        end)

        --
        -- Mix
        --
        modules.mix:set {
            { input = modules.high.amp:output(), level = 0.5 },
            { input = modules.low.lpf:output(), level = 0.5 },
            { input = modules.glitch.amp:output(), level = 0.05 },
        }

        --
        -- Effects
        --
        with(modules.effects, function(t)
            t.reverb:set { 
                input    = modules.mix:output(),
                cutoff   = hz(500),
                gain     = 0.4,
                feedback = 0.9
            }

            t.lpf:set { input = t.reverb:output(), cutoff = hz(5000) }
            t.noise:set { input = t.lpf:output(), gain = 0.02 }
        end)

        modules.compressor:set { input = modules.effects.noise:output() }

        return modules.compressor:output()
    end

    return build, patch
end
