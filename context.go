package fasthttp

import (
	"github.com/rock-go/rock/lua"
	"net"
	"strings"
	"github.com/rock-go/rock/region"
	"github.com/rock-go/rock/json"
)

type fsContext struct {
	lua.NoReflect
	meta lua.UserKV
}


func newContext() *lua.AnyData {
	ctx := &fsContext{meta: lua.NewUserKV()}
	ctx.initMeta()
	return lua.NewAnyData( ctx )
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
	uv := ctx.UserValue("region")
	if uv == nil {
		return byteNull
	}

	v , ok := uv.(*region.Info)
	if ok {
		return v.Byte()
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

type ToJson interface {
	ToJson() ([]byte , error)
}

func fsSayJson(co *lua.LState) int {
	val := co.Get(1)
	ctx := checkRequestCtx(co)

	var v interface{ToJson() ([]byte , error)}
	switch obj := val.(type) {
	case *lua.LightUserData:
		v = obj.Value
	case *lua.AnyData:
		var ok bool
		v , ok = obj.Value.(ToJson)
		if !ok {
			co.RaiseError("invalid toJson")
			return 0
		}

	case *lua.LUserData:
		var ok bool
		v , ok = obj.Value.(ToJson)
		if !ok {
			co.RaiseError("invalid toJson")
			return 0
		}
	default:
		co.RaiseError("invalid type , must object , got %s" , val.Type().String())
		return 0
	}
	chunk , e := v.ToJson()
	if e != nil {
		ctx.Error(e.Error() , 500)
		return 0
	}
	ctx.SetBody(chunk)
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

func fsISERR(co *lua.LState) int {
	n := co.GetTop()
	if n == 0 {
		return 0
	}

	v := co.Get(1)
	if v.Type() == lua.LTNil {
		return 0
	}
	co.RaiseError("%v" , v)
	return 0
}

func fsERR(co *lua.LState) int {
	n := co.GetTop()
	if n == 0 {
		co.RaiseError("invalid")
		return 0
	}

	data := make([]interface{}, n)
	format := make([]string , n)
	for i := 1; i<=n;i++ {
		format[i-1] = "%v "
		data[i-1] = co.CheckAny(i)
	}
	co.RaiseError(strings.Join(format , " ") , data...)
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
		return lua.LString(lua.B2S(ctx.Request.Host()))

	//浏览器标识
	case "ua":
		return lua.LString(lua.B2S(ctx.Request.Header.UserAgent()))

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
		return lua.LString(lua.B2S(ctx.URI().Path()))
	case "full_uri":
		return lua.LString(ctx.URI().String())

	case "query":
		return lua.LString(lua.B2S(ctx.URI().QueryString()))
	case "referer":
		return lua.LString(lua.B2S(ctx.Request.Header.Peek("referer")))

	case "content_length":
		return lua.LNumber(ctx.Request.Header.ContentLength())
	case "content_type":
		return lua.LString(lua.B2S(ctx.Request.Header.ContentType()))

	//返回结果
	case "status":
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
			return lua.LString(lua.B2S(ctx.QueryArgs().Peek(key[4:])))

		case strings.HasPrefix(key, "post_"):
			return lua.LString(lua.B2S(ctx.PostArgs().Peek(key[5:])))

		case strings.HasPrefix(key, "http_"):
			item := lua.S2B(key[5:])
			for i:=0;i<len(item);i++ { if item[i] == '_' { item[i] = '-' } }
			return lua.LString(lua.B2S(ctx.Request.Header.Peek(lua.B2S(item))))

		case strings.HasPrefix(key, "cookie_"):
			return lua.LString(lua.B2S(ctx.Request.Header.Cookie(key[7:])))

		case strings.HasPrefix(key , "region_"):
			uv := ctx.UserValue("region")
			if uv == nil {
				return lua.LNil
			}

			info , ok := uv.(*region.Info)
			if !ok {
				return lua.LNil
			}

			switch key[7:] {
			case "city":
				return lua.LString(lua.B2S(info.City()))
			case "city_id":
				return lua.LNumber(info.CityID())
			case "province":
				return lua.LString(lua.B2S(info.Province()))
			case "region":
				return lua.LString(lua.B2S(info.Region()))
			case "isp":
				return lua.LString(lua.B2S(info.ISP()))
			default:
				return lua.LNil
			}


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
				return lua.LString(lua.B2S(s.Byte()))
			default:
				return lua.LNil
			}
		}
	}

	return lua.LNil
}

func luaFastJSON(L *lua.LState) int {
	ctx := checkRequestCtx(L)
	f , err := json.NewFastJson(ctx.Request.Body())
	if err != nil {
		L.RaiseError("json %v" , err)
		return 0
	}
	L.Push(L.NewAnyData( f ))
	return 1
}

func (fsc *fsContext) initMeta() {
	fsc.meta.Set("say_json" , lua.NewFunction(fsSayJson))
	fsc.meta.Set("say" , lua.NewFunction(fsSay))
	fsc.meta.Set("append" , lua.NewFunction(fsAppend))
	fsc.meta.Set("exit" , lua.NewFunction(fsExit))
	fsc.meta.Set("eof" ,        lua.NewFunction(fsEof))
	fsc.meta.Set("set_header" , lua.NewFunction(fsHeader))
	fsc.meta.Set("ERR" , lua.NewFunction(fsERR))
	fsc.meta.Set("bind_json" , lua.NewFunction(luaFastJSON))
	fsc.meta.Set("file" , lua.NewFunction(newLuaFormFile))
}

func (fsc *fsContext) Get(co *lua.LState , key string) lua.LValue {
	ctx := checkRequestCtx(co)
	if v := fsc.meta.Get(key); v != lua.LNil {
		return v
	}
	return fsGet(ctx , key)
}