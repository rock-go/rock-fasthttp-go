package fasthttp

import (
	"github.com/rock-go/rock/logger"
	"github.com/rock-go/rock/lua"
	"github.com/rock-go/rock/xcall"
)

func newLuaServer(L *lua.LState) int {
	cfg := newConfig(L)
	proc := L.NewProc(cfg.name, fasthttpTypeOf)
	if proc.IsNil() {
		proc.Set(newServer(cfg))
	} else {
		proc.Value.(*server).cfg = cfg
	}

	L.Push(proc)
	return 1
}

func (fss *server) vHost(L *lua.LState) int {
	n := L.GetTop()
	hostname := L.CheckString(1)
	var r *vRouter

	switch n {
	case 1:
		r = newRouter(L , lua.LNil)

	case 2:
		r = newRouter(L , L.CheckTable(2))

	default:
		L.RaiseError("invalid router options")
		return 0
	}

	fss.vhost.insert(hostname , r)
	logger.Errorf("add %s router succeed" , hostname)
	L.Push(L.NewAnyData(r))
	return 1
}

func (fss *server) Index(L *lua.LState , key string) lua.LValue {
	switch key {
	case "vhost":
		return L.NewFunction(fss.vHost)

	}

	return lua.LNil
}

func LuaInjectApi(env xcall.Env) {
	fs := lua.NewUserKV()
	fs.Set("context", Context)

	fs.Set("server", lua.NewFunction(newLuaServer))
	fs.Set("handle", lua.NewFunction(newLuaHandle))
	fs.Set("H"     , lua.NewFunction(newLuaHandle))
	fs.Set("router", lua.NewFunction(newLuaRouter))
	fs.Set("filter", lua.NewFunction(newLuaFilter))
	fs.Set("header", lua.NewFunction(newLuaHeader))
	fs.Set("vhost" , lua.NewFunction(newLuaFasthttpHost))

	env.SetGlobal("fasthttp", fs)
}
