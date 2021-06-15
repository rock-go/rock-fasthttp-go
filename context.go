package fasthttp

import (
	"github.com/rock-go/rock/lua"
	"net"
	"strings"
)

type fsContext struct {
	lua.Super
	meta *pool
}

func newContext() *lua.LightUserData {
	return	lua.NewLightUserData( &fsContext{meta: newPool()})
}

func checkRequestCtx(co *lua.LState) *RequestCtx {
	if co.D == nil {
		return nil
	}

	ctx , ok := co.D.(*RequestCtx)
	if !ok {
		return nil
	}
	return ctx
}

func xPort(addr net.Addr) int {
	x , ok := addr.(*net.TCPAddr)
	if !ok {
		return 0
	}
	return x.Port
}

func regionCityId( ctx *RequestCtx) int {
	uv := ctx.UserValue("region_city")
	v , ok := uv.(int)
	if ok {
		return v
	}
	return 0
}

func regionRaw( ctx *RequestCtx) []byte {
	uv := ctx.UserValue("region_raw")
	v , ok := uv.([]byte)
	if ok {
		return v
	}
	return byteNull
}

func fsSay(co *lua.LState) int {
	n := co.GetTop()
	if n == 0 {
		return 0
	}

	ctx := checkRequestCtx(co)
	data := make([]string , n)
	for i := 1; i<=n;i++ {
		data[i-1] = co.CheckString(i)
	}
	ctx.Response.SetBodyString(strings.Join(data , ""))
	return 0
}

func fsAppend(co *lua.LState) int {
	n := co.GetTop()
	if n == 0 {
		return 0
	}

	data := make([]string , n)
	ctx := checkRequestCtx(co)
	for i := 1; i<=n;i++ {
		data[i-1] = co.CheckString(i)
	}
	ctx.Response.AppendBody(lua.S2B(strings.Join(data , "")))
	return 0
}

func fsExit(co *lua.LState) int {
	code := co.CheckInt(1)
	ctx := checkRequestCtx(co)
	ctx.Response.SetStatusCode(code)
	ctx.SetUserValue(eof_uv_key , true)
	return 0
}

func fsEof(co *lua.LState) int {
	ctx := checkRequestCtx(co)
	ctx.SetUserValue(eof_uv_key , true)
	return 0
}

func fsHeader(co *lua.LState) int {
	n := co.GetTop()
	if n == 0 {
		return 0
	}

	if n % 2 != 0 {
		co.RaiseError("#args % 2 != 0")
		return 0
	}

	ctx := checkRequestCtx(co)

	for i := 0 ; i < n ; {
		key := co.CheckString(i + 1)
		val := co.CheckString(i + 2)
		i += 2
		ctx.Response.Header.Set(key , val)
	}

	return 0

}

func fsGet(ctx *RequestCtx , key string) lua.LValue {
	switch key {
	//主机头
	case "host":
		return lua.LString(ctx.Request.Host())

	//浏览器标识
	case "ua":
		return lua.LString(ctx.Request.Header.UserAgent())

	//客户端信息
	case "remote_addr":
		return lua.LString(ctx.RemoteIP().String())
	case "remote_port":
		return lua.LNumber(xPort(ctx.RemoteAddr()))

	//服务器信息
	case "server_addr":
		return lua.LString(ctx.LocalIP().String())
	case "server_port":
		return lua.LNumber(xPort(ctx.LocalAddr()))

	//请求信息
	case "uri":
		return lua.LString(ctx.URI().Path())
	case "full_uri":
		return lua.LString(ctx.URI().String())

	case "query":
		return lua.LString(ctx.URI().QueryString())
	case "referer":
		return lua.LString(ctx.Request.Header.Peek("referer"))

	case "content_length":
		return lua.LNumber(ctx.Request.Header.ContentLength())
	case "content_type":
		return lua.LString(ctx.Request.Header.ContentType())

	//位置信息
	case "region_city":
		return lua.LNumber(regionCityId(ctx))

	//返回结果
	case "stauts":
		return lua.LNumber(ctx.Response.StatusCode())
	case "sent":
		return lua.LNumber(ctx.Response.Header.ContentLength())

	//返回完整的数据
	case "region_raw":
		return lua.LString(regionRaw(ctx))
	case "header_raw":
		return lua.LString(ctx.Request.Header.String())
	case "cookie_raw":
		return lua.LString(lua.B2S(ctx.Request.Header.Peek("cookie")))
	case "body_raw":
		return lua.LString(lua.B2S(ctx.Request.Body()))

	default:
		switch {
		case strings.HasPrefix(key, "arg_"):
			return lua.LString(ctx.QueryArgs().Peek(key[4:]))

		case strings.HasPrefix(key, "post_"):
			return lua.LString(ctx.PostArgs().Peek(key[5:]))

		case strings.HasPrefix(key, "http_"):
			return lua.LString(ctx.Request.Header.Peek(key[5:]))

		case strings.HasPrefix(key, "cookie_"):
			return lua.LString(ctx.Request.Header.Cookie(key[7:]))

		case strings.HasPrefix(key, "param_"):
			uv := ctx.UserValue(key[6:])
			switch s := uv.(type) {
			case lua.LValue:
				return s
			case string:
				return lua.LString(s)
			case int:
				return lua.LNumber(s)
			case interface{ String() string }:
				return lua.LString(s.String())
			case interface{ Byte() []byte }:
				return lua.LString(s.Byte())
			default:
				return lua.LNil
			}
		}
	}

	return lua.LNil
}

func (fc *fsContext) newFunc(co *lua.LState , key string , gn lua.LGFunction) *lua.LFunction {
	//缓存池
	item := fc.meta.Get(key)
	if item != nil {
		return item.val.(*lua.LFunction)
	}

	//新建
	fn := co.NewFunction(gn)
	fc.meta.insert(key , fn)
	return fn
}

func (fc *fsContext) Index(co *lua.LState , key string) lua.LValue {
	ctx := checkRequestCtx(co)
	if ctx == nil {
		co.RaiseError("invalid request context")
		return lua.LNil
	}

	switch key {

	//输出
	case "say":
		return fc.newFunc(co , key , fsSay)

	//添加
	case "append":
		return fc.newFunc(co , key , fsAppend)

	//退出
	case "exit":
		return fc.newFunc(co , key , fsExit)

	//退出
	case "eof":
		return fc.newFunc(co , key , fsEof)

	//请求头
	case "set_header":
		return fc.newFunc(co , key , fsHeader)

	//默认
	default:
		return fsGet(ctx , key)
	}

}