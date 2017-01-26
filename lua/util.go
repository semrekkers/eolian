package lua

var luaUtil = `
function with(o, fn)
	return fn(o)
end

function set(m, arg1, arg2)
	if type(arg1) == "table" then
		m:set(arg1)
		return m
	end
	m:set({ [tostring(arg1)] = arg2 })
	return m
end

function out(m, name)
	return m:output(name)
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
				if v['__type'] == 'module' then
					print(prefix .. k .. " (" .. v:id() .. ")")
				else
					print(prefix .. k)
				end
				inspect(v, ' - ')
			end
		end
	end
end

`
