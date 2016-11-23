local pkg = {}

local polyphony = 4

function voice(midi, idx)
    local pitch = synth.Multiple()
    local high = {
        osc = synth.Osc(),
    }
    local mid = {
        pitch = synth.Multiply(),
        osc   = synth.Osc(),
    }
    local low = {
        pitch = synth.Multiply(),
        osc   = synth.Osc(),
    }
    local mix  = synth.Mix()
    local adsr = synth.ADSR()
    local mult = synth.Multiply()

    pitch:set { input = midi:scope(idx):output('pitch') }

    high.osc:set  { pitch = pitch:output(0) }
    mid.pitch:set { a = pitch:output(1), b = 0.75 }
    mid.osc:set   { pitch = mid.pitch:output() }
    low.pitch:set { a = pitch:output(2), b = 0.25 }
    low.osc:set   { pitch = low.pitch:output() }

    mix:scope(0):set { input = high.osc:output('saw') }
    mix:scope(1):set { input = mid.osc:output('saw') }
    mix:scope(2):set { input = low.osc:output('saw') }

    adsr:set  {
        gate    = midi:scope(idx):output('gate'),
        attack  = ms(100),
        decay   = ms(50),
        sustain = 0.9,
        release = ms(3000),
    }
    mult:set { a = mix:output(), b = adsr:output() }

    return { 
        output = function() 
            return mult:output()
        end
    }
end

function pkg:build()
    local midi = synth.MIDIController { 
        device    = 2,
        polyphony = polyphony,
    }

    local voices = {}
    for i = 0,polyphony-1 do
        table.insert(voices, i+1, voice(midi, i))
    end

    return {
        midi     = midi,
        voices   = voices,
        mix      = synth.Mix { size = polyphony },
        filter   = synth.LPFilter(),
        delay    = synth.FBComb(),
        compress = synth.Compress(),
        clip     = synth.Clip(),
    }
end

function pkg:patch(modules)
    with(modules, function(m)
        for i = 0,polyphony-1 do
            m.mix:scope(i):set { input = m.voices[i+1]:output() }
        end

        m.filter:set   { input = m.mix:output(), cutoff = hz(5000) }
        m.delay:set    { input = m.filter:output(), gain = 0.4 }
        m.compress:set { input = m.delay:output() }
        m.clip:set     { input = m.compress:output(), max = 3 }
    end)

    return modules.clip:output()
end

return pkg
