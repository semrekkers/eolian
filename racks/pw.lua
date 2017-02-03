return function(env)
    local synth = require('eolian.synth')

    local function build()
        return {
            controller = synth.MIDIController { device = 'Launch Control' },
            debug      = synth.Debug(),
            osc        = synth.Oscillator(),

            interp     = synth.Interpolate { min = hz(100), max = hz(6000) },
            resinterp  = synth.Interpolate { min = 1, max = 50 },

            filter     = synth.Filter(),
            amp        = synth.Multiply(),
            compress   = synth.Compress(),
        }
    end

    local function patch(m)
        local channel = m.controller:ns('cc/1')

        set(m.osc, { pitch = pitch('C2'), pulseWidth = out(channel, 23) })
        set(m.interp, { input = out(channel, 21) })
        set(m.resinterp, { input = out(channel, 22) })
        set(m.filter, { input = out(m.osc, 'pulse'), cutoff = out(m.interp), resonance = out(m.resinterp)})
        set(m.debug, { input = out(m.filter) })
        set(m.amp, { a = out(m.debug), b = 0.5 })
        set(m.compress, { input = out(m.amp) })

        return out(m.compress)
    end

    return build, patch
end
