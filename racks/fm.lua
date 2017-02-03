return function(env)
    local synth  = require('eolian.synth')
    local interp = require('eolian.synth.interpolate')

    local function build()
        return {
            midi   = synth.MIDIController { device = 'Launch Control' },
            osc1   = interp(synth.Oscillator(), { pitch = { min = hz(1), max = hz(100) } }),
            osc2   = interp(synth.Oscillator(), { pitch = { min = hz(100), max = hz(2000) } }),
            filter = interp(synth.Filter(), { cutoff = { max = hz(5000) }, resonance = { max = 100 } }),
            mix    = synth.Mix(),
        }
    end

    local function patch(m)
        local ch1 = m.midi:ns('cc/1')

        set(m.osc1, { pitch = out(ch1, 21) })
        set(m.osc2, { pitch = out(ch1, 23), pitchModAmount = out(ch1, 22), pitchMod = out(m.osc1, 'saw') })
        set(m.filter, { input = out(m.osc2, 'sine'), cutoff = out(ch1, 41), resonance = out(ch1, 42) })

        set(m.mix, {
            { input = out(m.filter, 'lowpass'), level = out(ch1, 43) },
            { input = out(m.filter, 'bandpass'), level = out(ch1, 44) },
            { input = out(m.filter, 'highpass'), level = out(ch1, 45) },
        })

        return out(m.mix)
    end

    return build, patch
end
