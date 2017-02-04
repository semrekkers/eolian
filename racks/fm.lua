return function(env)
    local synth  = require('eolian.synth')
    local theory = require('eolian.theory')
    local interp = require('eolian.synth.interpolate')

    local function build()
        return {
            clock = {
                osc      = interp(synth.Oscillator(), { pitch = { min = hz(1), max = hz(20) }}),
                multiple = synth.Multiple(),
            },

            midi = synth.MIDIController { device = 'Launch Control' },
            debug = synth.Debug(),

            oscA = {
                quantize = synth.Quantize(),
                multiple = synth.Multiple(),
            },
            oscB = {
                quantize = synth.Quantize(),
                multiple = synth.Multiple(),
            },

            op1 = {
                multiplierFloor = interp(synth.Floor(), { input = { min = 1, max = 10 } }),
                multiplier      = synth.Multiply(),
                osc             = synth.Oscillator { algorithm = 'simple' },
                noise           = synth.Noise(),
            },
            op2 = {
                multiplierFloor = interp(synth.Floor(), { input = { min = 1, max = 10 } }),
                multiplier      = synth.Multiply(),
                osc             = synth.Oscillator(),
                noise           = synth.Noise(),
            },
            op3 = {
                multiplierFloor = interp(synth.Floor(), { input = { min = 1, max = 10 } }),
                multiplier      = synth.Multiply(),
                osc             = synth.Oscillator(),
                noise           = synth.Noise(),
            },
            op4 = {
                multiplierFloor = interp(synth.Floor(), { input = { min = 1, max = 10 } }),
                multiplier      = synth.Multiply(),
                osc             = synth.Oscillator(),
                noise           = synth.Noise(),
            },

            mix    = synth.Mix(),
            filter = interp(synth.Filter(), { cutoff = { max = hz(5000) }, resonance = { max = 100 } }),

            adsr   = interp(synth.ADSR(), {
                attack  = { min = ms(1), max = ms(500) },
                decay   = { min = ms(1), max = ms(500) },
                release = { min = ms(1), max = ms(500) },
            }),
            amp    = synth.Multiply(),
            delay  = synth.FBComb(),
        }
    end

    local function patch(m)
        local cc = function(n) return out(m.midi:ns('cc/1'), n) end

        local clock = with(m.clock, function(c)
            set(c.osc, { pitch = cc(28) })
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

        local oscA = with(m.oscA, function(o)
            set(o.quantize, { input = cc(25) })
            local scale = theory.scale('C2', 'minorPentatonic', 2)
            for i,p in ipairs(scale) do
                set(o.quantize, i-1 .. '/pitch', p)
            end
            set(o.multiple, { input = out(o.quantize) })
            return o.multiple
        end)

        local oscB = with(m.oscB, function(o)
            set(o.quantize, { input = cc(26) })
            local scale = theory.scale('C2', 'minorPentatonic', 2)
            for i,p in ipairs(scale) do
                set(o.quantize, i-1 .. '/pitch', p)
            end
            set(o.multiple, { input = out(o.quantize) })
            return o.multiple
        end)

        local op1 = with(m.op1, function(op)
            set(op.multiplierFloor, { input = cc(21) })
            set(op.multiplier, { a = out(op.multiplierFloor), b = out(oscA, 0) })
            set(op.osc, { pitch = out(op.multiplier), amp = cc(22) })
            return out(op.osc, 'sine')
        end)

        local op2 = with(m.op2, function(op)
            set(op.multiplierFloor, { input = cc(41) })
            set(op.multiplier, { a = out(op.multiplierFloor), b = out(oscA, 1) })
            set(op.osc, { pitch = out(op.multiplier), pitchMod = op1, amp = cc(42) })
            set(op.noise, { input = out(op.osc, 'sine'), gain = 0.01 })
            return out(op.noise)
        end)

        local op3 = with(m.op3, function(op)
            set(op.multiplierFloor, { input = cc(23) })
            set(op.multiplier, { a = out(op.multiplierFloor), b = out(oscB, 0) })
            set(op.osc, { pitch = out(op.multiplier), amp = cc(24) })
            set(op.noise, { input = out(op.osc, 'saw'), gain = 0.1 })
            return out(op.noise)
        end)

        local op4 = with(m.op4, function(op)
            set(op.multiplierFloor, { input = cc(43) })
            set(op.multiplier, { a = out(op.multiplierFloor), b = out(oscB, 1) })
            set(op.osc, { pitch = out(op.multiplier), amp = cc(44) })
            set(op.noise, { input = out(op.osc, 'pulse'), gain = 0.1 })
            return out(op.noise)
        end)

        set(m.mix, {
            { input = op2 },
            { input = op3 },
            { input = op4 },
        })
        set(m.filter, { input = out(m.mix), 
                        cutoff = cc(27) })

        set(m.amp, { a = out(m.filter, 'lowpass'), b = out(m.adsr) })
        set(m.delay, { input = out(m.amp), gain = 0.3, duration = ms(200) })
        return out(m.delay)
    end

    return build, patch
end
