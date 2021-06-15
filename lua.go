package fasthttp

import (
	"github.com/rock-go/rock/xcall"
	"github.com/rock-go/rock/lua"
)


func (ser *server) Index(L *lua.LState , key string) lua.LValue {
	return lua.LNil
}

func newLuaServer(L *lua.LState) int {
	cfg := newConfig(L)
	if e := cfg.verify(); e != nil {
		L.RaiseError("%v" , e)
		return 0
	}

	var fs *server
	var ok bool

	proc := L.NewProc(cfg.name)
	if proc.Value == nil {
		proc.Value = newServer( cfg )
		goto done
	}

	fs , ok = proc.Value.(*server)
	if !ok {
		L.RaiseError("invalid type %s running" , cfg.name)
		return 0
	}

	//重启服务
	if e := fs.Close(); e != nil {
		L.RaiseError("%v" , e)
		return 0
	}
	proc.Value = newServer(cfg)

done:
	L.Push(proc)
	return 1
}

func LuaInjectApi(env xcall.Env) {
	fs := lua.NewUserKV()
	fs.Set("ctx", instance)

	fs.Set("server" , lua.NewFunction(newLuaServer))
	fs.Set("handle" , lua.NewFunction(newLuaHandle))
	fs.Set("router" , lua.NewFunction(newLuaRouter))
	fs.Set("filter" , lua.NewFunction(newLuaFilter))
	fs.Set("header" , lua.NewFunction(newLuaHeader))

	fsJson := lua.NewUserKV()
	fsJson.Set("decode" , lua.NewFunction(newLuaJsonDecode))
	fsJson.Set("encode" , lua.NewFunction(newLuaJsonEncode))
	fs.Set("json" , fsJson)

	env.SetGlobal("fasthttp" , fs)
}
