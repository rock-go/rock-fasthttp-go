package fasthttp

import (
	"github.com/rock-go/rock/lua"
	"github.com/rock-go/rock/xcall"
)

func newLuaServer(L *lua.LState) int {
	cfg := newConfig(L)
	proc := L.NewProc(cfg.name, fasthttpTypeOf)
	if proc.IsNil() {
		proc.Set(newServer(cfg))
		goto done
	}
	proc.Value.(*server).cfg = cfg

done:
	L.Push(proc)
	return 1
}

func LuaInjectApi(env xcall.Env) {
	fs := lua.NewUserKV()
	fs.Set("ctx", Context)

	fs.Set("server", lua.NewFunction(newLuaServer))
	fs.Set("handle", lua.NewFunction(newLuaHandle))
	fs.Set("router", lua.NewFunction(newLuaRouter))
	fs.Set("filter", lua.NewFunction(newLuaFilter))
	fs.Set("header", lua.NewFunction(newLuaHeader))

	env.SetGlobal("fasthttp", fs)
}
