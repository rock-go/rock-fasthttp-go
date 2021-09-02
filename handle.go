package fasthttp

import (
	"errors"
	"github.com/rock-go/rock/logger"
	"github.com/rock-go/rock/lua"
	"github.com/rock-go/rock/xcall"
	"github.com/valyala/fasthttp"
	"sync/atomic"
)

const (
	VHANDLER handleType = iota + 1 //表示当前数据类型
	VHSTRING
	VHFUNC
)

var (
	emptyHandle        = errors.New("empty handle object")
	rockFasthttpHeader = "rock-fasthttp-go v1.0"
)

type handleType int

type handle struct {
	//必须字段
	name   string
	mtime  int64

	//业务字段
	count  uint32
	filter *filter

	//返回包处理
	code   int
	header *header
	hook   *lua.LFunction
	close  *lua.LFunction
	//返回结果
	body []byte

	//结束匹配
	eof bool
}

func newHandle(name string) *handle {
	return &handle{name: name, eof: false}
}

func (hd *handle) DisableReflect() {}

func (hd *handle) Close() error {
	if hd.close == nil {
		return nil
	}
	co := lua.State()
	defer lua.FreeState(co)

	return xcall.CallByEnv(co, hd.close, xcall.Rock)
}

func (hd *handle) MTime() int64 {
	return hd.mtime
}

func (hd *handle) Option() interface{} {
	return nil
}

func (hd *handle) do(co *lua.LState, ctx *RequestCtx , eof *bool) error {
	atomic.AddUint32(&hd.count, 1)

	if hd.filter == nil {
		goto set
	}

	if hd.filter.do(ctx) {
		goto set
	}

	//如果没有命中 eof 掉

	*eof = false
	return  nil

set:
	//设置header
	ctx.Response.Header.Set("server", rockFasthttpHeader)
	if hd.header != nil {
		hd.header.ForEach(func(key string, val string) {
			ctx.Response.Header.Set(key, val)
		})
	}

	if hd.code == 0 && hd.hook == nil && hd.body == nil {
		return emptyHandle
	}

	//设置状态
	if hd.code != 0 {
		//设置状态码
		ctx.SetStatusCode(hd.code)
	}

	//设置响应体
	if hd.body != nil {
		ctx.SetBody(hd.body)
	}

	//运行hook
	if hd.hook != nil {
		return xcall.CallByEnv(co, hd.hook, xcall.Rock)
	}

	*eof = true
	return nil
}

func (hd *handle) Set(L *lua.LState , key string , val lua.LValue) {
	switch key {
	case "code":
		if val.Type() != lua.LTNumber {
			L.RaiseError("invalid handle code , must be int")
			return
		}
		hd.code = int(val.(lua.LNumber))

	case "filter":
		hd.filter = toFilter(L , val)

	case "header":
		hd.header = toHeader(L , val)

	case "hook":
		if val.Type() != lua.LTFunction {
			L.RaiseError("invalid handle hook , must be function")
			return
		}
		hd.hook = val.(*lua.LFunction)
	case "close":
		if val.Type() != lua.LTFunction {
			return
		}
		hd.close = val.(*lua.LFunction)

	case "eof":
		if val.Type() != lua.LTBool {
			L.RaiseError("invalid handle eof , must be bool")
			return
		}
		hd.eof = bool(val.(lua.LBool))

	case "body":
		if val.Type() != lua.LTString {
			L.RaiseError("invalid handle body")
			return
		}
		hd.body = lua.S2B(val.String())

	case "file":

	}
}

func newLuaHandle(L *lua.LState) int {
	tab := L.CheckTable(1)
	hd := newHandle("")

	tab.ForEach(func(key lua.LValue, val lua.LValue) {
		if key.Type() != lua.LTString {
			L.RaiseError("invalid fasthttp handle option")
			return
		}

		hd.Set(L , key.String() , val)
	})

	L.Push(L.NewAnyData(hd))
	return 1
}

type HandleChains struct {
	data []interface{}
	mask []handleType
	cap  int
}

func newHandleChains(cap int) *HandleChains {
	return &HandleChains{
		data: make([]interface{}, cap),
		mask: make([]handleType, cap),
		cap:  cap,
	}
}

func (hc *HandleChains) Store(v interface{}, mask handleType, offset int) {
	if offset > hc.cap {
		logger.Errorf("vHandle overflower , cap:%d , got: %d", hc.cap, offset)
		return
	}

	hc.data[offset] = v
	hc.mask[offset] = mask
}

//没有匹配的Handle代码
var notFoundBody = []byte("not found handle")

func (hc *HandleChains) notFound(ctx *RequestCtx) {
	ctx.Response.SetStatusCode(fasthttp.StatusNotFound)
	ctx.Response.SetBody(notFoundBody)
}

func (hc *HandleChains) invalid(ctx *RequestCtx, body string) {
	ctx.Response.SetStatusCode(fasthttp.StatusInternalServerError)
	ctx.Response.SetBodyString(body)
}

func (hc *HandleChains) do(ctx *RequestCtx, path string) { //path handle 查找路径
	if hc.cap == 0 {
		hc.notFound(ctx)
		return
	}

	var item *handle
	var err error
	var eof bool

	co := newLuaThread(ctx)
	for i := 0; i < hc.cap; i++ {
		switch hc.mask[i] {

		//字符串
		case VHSTRING:
			item, err = requireHandle(path, hc.data[i].(string))
			if err != nil {
				hc.invalid(ctx, err.Error())
				return
			}

			err = item.do(co, ctx , &eof)
			if err != nil {
				hc.invalid(ctx, err.Error())
				return
			}

		//处理对象
		case VHANDLER:
			item = hc.data[i].(*handle)
			err = item.do(co, ctx , &eof)
			if err != nil {
				hc.invalid(ctx, err.Error())
				return
			}

		case VHFUNC:
			if e := xcall.CallByEnv(co,
				hc.data[i].(*lua.LFunction), xcall.Rock); e != nil {
				hc.invalid(ctx, e.Error())
				return
			}

		//异常
		default:
			hc.invalid(ctx, "invalid handle type")
			return
		}

		if eof || checkLuaEof(ctx) {
			return
		}

	}
}
