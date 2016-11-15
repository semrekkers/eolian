local pkg = {}

function pkg.build(self, path)
    local supersaw = dofile(path .. '/voice/supersaw.lua')

    local modules = {
        -- controller = synth.MIDIController({ device = 2 }),
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
            lpf      = synth.LPFilter(),
            amp      = synth.BinaryMultiply()
        },
        low = {
            divider  = synth.ClockDivide(),
            sequence = synth.Sequence(),
            pitch    = synth.Multiple(),
            osc      = synth.Osc(),
            subPitch = synth.BinaryMultiply(),
            subOsc   = synth.Osc(),
            fold     = synth.Fold(),
            mix      = synth.Mix(),
            lpf      = synth.LPFilter()
        },
        glitch = {
            multiplier = synth.ClockMultiply(),
            sequence   = synth.Sequence(),
            osc        = synth.Osc(),
            adsr       = synth.ADSR(),
            amp        = synth.BinaryMultiply(),
        },
        effects = {
            reverb = synth.FilteredReverb(),
            lpf    = synth.LPFilter(),
            noise  = synth.Noise()
        },
        compressor = synth.Compress()
    }
    return {
        modules = modules,
        output = function()
            return modules.compressor:output()
        end
    }
end

function pkg.patch(self, modules)
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
        t.sequence:scope(0):set { pitch = pitch('C2'), pulses = 2, mode = 2, glide  = 0 }
        t.sequence:scope(1):set { pitch = pitch('C3'), pulses = 2, mode = 2, glide  = 0 }
        t.sequence:scope(2):set { pitch = pitch('C2'), pulses = 1, mode = 2, glide  = 0 }
        t.sequence:scope(3):set { pitch = pitch('C3'), pulses = 2, mode = 2, glide  = 1 }
        t.sequence:scope(4):set { pitch = pitch('C4'), pulses = 2, mode = 2, glide  = 1 }
        t.sequence:scope(5):set { pitch = pitch('C3'), pulses = 1, mode = 2, glide  = 0 }
        t.sequence:scope(6):set { pitch = pitch('C2'), pulses = 2, mode = 2, glide  = 0 }
        t.sequence:scope(7):set { pitch = pitch('C3'), pulses = 1, mode = 2, glide  = 1 }

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
        t.sequence:scope(0):set { pitch = pitch('C2') }
        t.sequence:scope(1):set { pitch = pitch('C2') }
        t.sequence:scope(2):set { pitch = pitch('F2') }
        t.sequence:scope(3):set { pitch = pitch('C2') }
        t.sequence:scope(4):set { pitch = pitch('C2') }
        t.sequence:scope(5):set { pitch = pitch('C2') }
        t.sequence:scope(6):set { pitch = pitch('F2') }
        t.sequence:scope(7):set { pitch = pitch('G2') }

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

        t.mix:scope(0):set { input = t.fold:output() }
        t.mix:scope(1):set { 
            input = t.subOsc:output('saw'), 
            level = 0.7 
        }

        t.lpf:set { input = t.mix:output(), cutoff = hz(3000) }
    end)

    --
    -- Glitch
    --
    with(modules.glitch, function(t)
        t.sequence:set { clock = modules.clock.multiple:output(2), mode = 2 }
        t.sequence:scope(0):set { pitch = pitch('C5'), pulses = 1, mode = 2 }
        t.sequence:scope(1):set { pitch = pitch('C6'), pulses = 1, mode = 2 }
        t.sequence:scope(2):set { pitch = pitch('C5'), pulses = 1, mode = 2 }
        t.sequence:scope(3):set { pitch = pitch('C8'), pulses = 1, mode = 2 }
        t.sequence:scope(4):set { pitch = pitch('C6'), pulses = 1, mode = 2 }
        t.sequence:scope(5):set { pitch = pitch('C5'), pulses = 1, mode = 2 }
        t.sequence:scope(6):set { pitch = pitch('C3'), pulses = 1, mode = 2 }
        t.sequence:scope(7):set { pitch = pitch('C7'), pulses = 1, mode = 2 }

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
    with(modules.mix, function(t)
        t:scope(0):set { input = modules.high.amp:output(), level = 0.5 }
        t:scope(1):set { input = modules.low.lpf:output(), level = 0.5 }
        t:scope(2):set { input = modules.glitch.amp:output(), level = 0.05 }
    end)

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
end

return pkg
