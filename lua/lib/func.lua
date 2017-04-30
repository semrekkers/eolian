local function with(o, fn)
    return fn(o)
end

local function actsLikeModule(m)
    return type(m)      == 'table' and
        type(m.inputs)  == 'function' and
        type(m.outputs) == 'function' and
        type(m.out)     == 'function' and
        type(m.set)     == 'function' and
        type(m.id)      == 'function'
end

local function set(m, ...)
    assert(m, 'attempt to set inputs on nil value')
    assert(actsLikeModule(m), 'attempt to set inputs on non-module value')
    return m:set(unpack(arg))
end

local function out(m, name)
    assert(m, 'attempt to get output "' .. tostring(name) .. '" from nil value')
    assert(actsLikeModule(m), 'attempt to get output "' .. tostring(name) .. '" from non-module value')
    return m:out(name)
end

return {
    with = with,
    set  = set,
    out  = out,
}
