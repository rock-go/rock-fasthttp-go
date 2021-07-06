package fasthttp

import "github.com/rock-go/rock/lua"

func newLuaThread( ctx *RequestCtx ) *lua.LState {
	uv := ctx.UserValue(thread_uv_key)
	if uv != nil {
		return uv.(*lua.LState)
	}

	//设置ctx
	co := lua.State()
	co.D = ctx
	ctx.SetUserValue(thread_uv_key , co)

	return co
}

func freeLuaThread( ctx *RequestCtx ) {
	co := ctx.UserValue(thread_uv_key)
	if co == nil {
		return
	}

	lua.FreeState(co.(*lua.LState))
}
