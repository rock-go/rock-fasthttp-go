package fasthttp

import "github.com/rock-go/rock/lua"

var (
	thread_uv_key = "__thread_co__"
	eof_uv_key = "__handle_eof__"
)

func checkLuaEof( ctx *RequestCtx) bool {
	uv := ctx.UserValue(eof_uv_key)
	if uv == nil {
		return false
	}

	v , ok := uv.(bool)
	if !ok {
		return false
	}

	return v
}

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
