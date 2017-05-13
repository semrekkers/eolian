local time = require('eolian.time')

function ping(m, input)
    m:set(input, -1)
    time.sleep(50)
    m:set(input, 1)
    time.sleep(50)
    m:set(input, -1)
end
