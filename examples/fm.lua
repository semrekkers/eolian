local synth  = require('eolian.synth')
local theory = require('eolian.theory')
local ctrl   = require('eolian.synth.control')

return function(_)
    local function build()
        return {
            clock = ctrl(synth.Clock(), { tempo = { min = hz(1), max = hz(20) }}),
            midi  = synth.MIDIController { device = 'Launch Control' },
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
                multiplier = ctrl(synth.Multiply(), { a = { min = 1, max = 10 } }),
                osc        = synth.Oscillator(),
            },
            op2 = {
                multiplier = ctrl(synth.Multiply(), { a = { min = 1, max = 10 } }),
                osc        = synth.Oscillator(),
                noise      = synth.Noise(),
            },
            op3 = {
                multiplier = ctrl(synth.Multiply(), { a = { min = 1, max = 10 } }),
                osc        = synth.Oscillator(),
            },
            op4 = {
                multiplier = ctrl(synth.Multiply(), { a = { min = 1, max = 10 } }),
                osc        = synth.Oscillator(),
                noise      = synth.Noise(),
            },

            mix    = synth.Mix(),
            filter = ctrl(synth.Filter(), { cutoff = { max = hz(5000) }, resonance = { max = 100 } }),

            adsr   = ctrl(synth.ADSR(), {
                attack  = { min = ms(1), max = ms(500) },
                decay   = { min = ms(1), max = ms(500) },
                release = { min = ms(1), max = ms(500) },
            }),
            amp    = synth.Multiply(),
            delay  = synth.FBDelay(),
        }
    end

    local function patch(m)
        local cc = function(n) return out(m.midi:ns('cc/1'), n) end

        set(m.clock, { tempo = cc(28) })

        set(m.adsr, {
            gate    = out(m.clock),
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
            return o.quantize
        end)

        local oscB = with(m.oscB, function(o)
            set(o.quantize, { input = cc(26) })
            local scale = theory.scale('C2', 'minorPentatonic', 2)
            for i,p in ipairs(scale) do
                set(o.quantize, i-1 .. '/pitch', p)
            end
            return o.quantize
        end)

        local op1 = with(m.op1, function(op)
            set(op.multiplier, { a = cc(21), b = out(oscA) })
            set(op.osc, { pitch = out(op.multiplier), amp = cc(22) })
            return out(op.osc, 'sine')
        end)

        local op2 = with(m.op2, function(op)
            set(op.multiplier, { a = cc(41), b = out(oscA) })
            set(op.osc, { pitch = out(op.multiplier), pitchMod = op1, amp = cc(42) })
            set(op.noise, { input = out(op.osc, 'sine'), gain = 0.1 })
            return out(op.noise)
        end)

        local op3 = with(m.op3, function(op)
            set(op.multiplier, { a = cc(23), b = out(oscB) })
            set(op.osc, { pitch = out(op.multiplier), amp = cc(24) })
            return out(op.osc, 'saw')
        end)

        local op4 = with(m.op4, function(op)
            set(op.multiplier, { a = cc(43), b = out(oscB) })
            set(op.osc, { pitch = out(op.multiplier), pitchMod = op3, amp = cc(44) })
            set(op.noise, { input = out(op.osc, 'sine'), gain = 0.1 })
            return out(op.noise)
        end)

        set(m.mix, {
            { input = op2 },
            { input = op4 },
        })

        set(m.amp, { a = out(m.mix), b = out(m.adsr) })
        set(m.delay, { input = out(m.amp), gain = 0.4, duration = ms(100) })
        set(m.filter, { input = out(m.delay), cutoff = cc(27) })

        local sink = out(m.filter, 'lowpass')

        return sink, sink
    end

    return build, patch
end
