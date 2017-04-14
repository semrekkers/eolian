return function(env)
    local synth = require('eolian.synth')
    local ctrl  = require('eolian.synth.control')

    local function build()
        return {
            debug = synth.Debug(),
            ctrl  = synth.MIDIController {
                device    = 'Arturia BeatStep Pro Arturia BeatStepPro',
                polyphony = 4,
            },
            lfo  = synth.Oscillator { algorithm = 'simple' },
            lfo2 = synth.Oscillator { algorithm = 'simple' },
            voice1 = {
                osc  = synth.Oscillator(),
                adsr = synth.ADSR(),
                gate = synth.LPGate(),
                mix  = synth.Mix(),
            },
            survey = ctrl(synth.Survey, {
                survey = { min = -1, max = 1 },
            }),
            mix    = synth.PanMix(),
            reverb = synth.TankReverb(),
        }
    end

    local function patch(modules)
        modules.lfo:set { pitch = hz(0.2), amp = 0.4, offset = 0.5 }
        -- modules.lfo2:set { pitch = hz(5) }

        local v1 = with(modules.voice1, function(v)
            v.osc:set { pitch = modules.ctrl:out('0/pitch'), pulseWidth = modules.lfo:out('sine') }
            v.mix:set {
                { input = v.osc:out('sine') },
                { input = v.osc:out('pulse'), level = 0.1 },
                { input = v.osc:out('sub'), level = 0.1 },
            }
            v.adsr:set {
                gate    = modules.ctrl:out('0/gate'),
                attack  = ms(10),
                decay   = ms(50),
                sustain = 0.2,
                release = ms(10),
            }
            v.gate:set {
                input   = v.mix:out(),
                control = v.adsr:out(),
            }
            return v.gate:out()
        end)

        -- modules.survey:set {
        --     a      = v1,
        --     survey = modules.ctrl:out('cc/1/20'),
        -- }
        modules.mix:set {
            { input = v1, pan = 0 },
        }
        modules.mix:set { master = 0.3 }
        modules.reverb:set {
            a      = modules.mix:out('a'),
            b      = modules.mix:out('b'),
            cutoff = hz(500),
            decay  = 0.1,
            bias   = -0.5,
        }

        return modules.reverb:out('a'),
               modules.reverb:out('b')
    end

    return build, patch
end
