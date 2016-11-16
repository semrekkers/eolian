local pkg = {}

local polyphony = 10

function voice(midi, idx)
    local pitch = synth.Multiple()
    local bend  = synth.Multiple()
    local high = {
        osc = synth.Osc(),
    }
    local low = {
        pitch = synth.BinaryMultiply(),
        osc   = synth.Osc(),
    }
    local mix  = synth.Mix()
    local adsr = synth.ADSR()
    local mult = synth.BinaryMultiply()

    pitch:set { input = midi:scope(idx):output('pitch') }
    bend:set  { input = midi:scope(idx):output('pitchBend') }

    high.osc:set  { pitch = pitch:output(0), pitchMod = bend:output(0), pitchModAmount = 0.1 }
    low.pitch:set { a = pitch:output(1), b = 0.25 }
    low.osc:set   { pitch = low.pitch:output(), pitchMod = bend:output(1), pitchModAmount = 0.1 }

    mix:scope(0):set { input = high.osc:output('saw') }
    mix:scope(1):set { input = low.osc:output('saw') }

    adsr:set  {
        gate    = midi:scope(idx):output('gate'),
        attack  = ms(100),
        decay   = ms(100),
        sustain = 0.8,
        release = ms(1000),
    }
    mult:set { a = mix:output(), b = adsr:output() }

    return { 
        output = function() 
            return mult:output()
        end
    }
end

function pkg.build(self)
    local midi = synth.MIDIController { 
        device    = 2,
        polyphony = polyphony,
    }

    local voices = {}
    for i = 0,polyphony-1 do
        table.insert(voices, i+1, voice(midi, i))
    end

    local modules = {
        midi     = midi,
        voices   = voices,
        mix      = synth.Mix { size = polyphony },
        filter   = synth.LPFilter(),
        delay    = synth.FBComb(),
        compress = synth.Compress(),
        clip     = synth.Clip(),
    }
    return {
        modules = modules,
        output = function()
            return modules.clip:output()
        end
    }
end

function pkg.patch(self, modules)
    with(modules, function(m)
        for i = 0,polyphony-1 do
            m.mix:scope(i):set { input = m.voices[i+1]:output() }
        end

        m.filter:set   { input = m.mix:output(), cutoff = hz(5000) }
        m.delay:set    { input = m.filter:output(), gain = 0.4 }
        m.compress:set { input = m.delay:output() }
        m.clip:set     { input = m.compress:output(), max = 3 }
    end)
end

return pkg
