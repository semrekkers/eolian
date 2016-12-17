package lua

var luaUtil = `
function with(o, fn)
	return fn(o)
end

function inspect(o, prefix)
	if type(o) == 'table' and prefix == nil then
		if o['__type'] == 'module' then
			print(o:info())
			return
		end
	end
	for k, v in pairs(o) do
		if k ~= '__namespace' then
			if prefix == nil then
				prefix = ''
			end
			if type(v) == 'table' then
				print(prefix .. k)
				inspect(v, ' - ')
			end
		end
	end
end

`
