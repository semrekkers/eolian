package lua

var luaUtil = `
function with(o, fn)
	return fn(o)
end

function inputProxy(m)
	return function(_, inputs)
		m:set(inputs)
	end
end

function outputProxy(m)
	return function(_, output)
		if output == nil then
			return m:output()
		else
			return m:output(output)
		end
	end
end
`
