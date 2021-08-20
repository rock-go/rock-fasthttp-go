package fasthttp

import (
	"bytes"
	"github.com/rock-go/rock/lua"
	"io"
)

func newLuaFormFile(co *lua.LState) int {
	ctx := checkRequestCtx(co)
	h, e := ctx.FormFile("file")
	if e != nil {
		co.RaiseError("%v", e)
		return 0
	}

	file, e := h.Open()
	if e != nil {
		co.RaiseError("%v", e)
		return 0
	}
	defer file.Close()
	name := h.Filename
	var buff bytes.Buffer
	n, e := io.Copy(&buff, file)
	if e != nil {
		co.RaiseError("%v", e)
		return 0
	}

	co.Push(lua.LString(name))
	co.Push(lua.LString(buff.String()))
	co.Push(lua.LNumber(n))
	return 3
}
