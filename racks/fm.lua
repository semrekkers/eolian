return function(env)
    local synth  = require('eolian.synth')
    local interp = require('eolian.synth.interpolate')

    local function build()
        return {
            clock = {
                osc      = interp(synth.Oscillator(), { pitch = { min = hz(1), max = hz(20) }}),
                multiple = synth.Multiple(),
            },

            midi   = synth.MIDIController { device = 'Launch Control' },
            osc1   = interp(synth.Oscillator(), { pitch = { min = hz(1), max = hz(100) } }),
            osc2   = interp(synth.Oscillator(), { pitch = { min = hz(100), max = hz(2000) } }),
            filter = interp(synth.Filter(), { cutoff = { max = hz(5000) }, resonance = { max = 100 } }),
            mix    = synth.Mix(),

            adsr   = interp(synth.ADSR(), {
                attack  = { min = ms(1), max = ms(500) },
                decay   = { min = ms(1), max = ms(500) },
                release = { min = ms(1), max = ms(500) },
            }),
            amp    = synth.Multiply(),
        }
    end

    local function patch(m)
        local cc = function(n) return out(m.midi:ns('cc/1'), n) end

        local clock = with(m.clock, function(c)
            set(c.osc, { pitch = cc(28), pulseWidth = cc(27) })
            set(c.multiple, { input = out(c.osc, 'pulse') })
            return c.multiple
        end)

        set(m.adsr, {
            gate    = out(clock, 0),
            attack  = cc(45),
            decay   = cc(46),
            sustain = cc(47),
            release = cc(48),
        })

        set(m.osc1, { pitch = cc(21) })
        set(m.osc2, { pitch = cc(23), pitchModAmount = cc(22), pitchMod = out(m.osc1, 'saw') })
        set(m.filter, { input = out(m.osc2, 'sine'), cutoff = cc(41), resonance = cc(42) })

        set(m.mix, {
            { input = out(m.filter, 'lowpass'), level = cc(43) },
            { input = out(m.filter, 'bandpass'), level = cc(44) },
            -- { input = out(m.filter, 'highpass'), level = cc(45) },
        })

        set(m.amp, { a = out(m.mix), b = out(m.adsr) })
        return out(m.amp)
    end

    return build, patch
end
