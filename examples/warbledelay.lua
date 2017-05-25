return function()
    local synth = require('eolian.synth')

    local function build()
        return {
            flux = {
                control = synth.Control(),
                amount  = synth.Control(),
                random  = synth.Random(),
            },
            mix = synth.PanMix(),
        }
    end

    local function patch(m)
        m.mix:set {
        }
        return m.mix:out('a', 'b')
    end

    return build, patch
end
