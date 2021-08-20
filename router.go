package fasthttp

import (
	"github.com/fasthttp/router"
	"github.com/rock-go/rock/logger"
	"github.com/rock-go/rock/lua"
	"github.com/rock-go/rock/region"
	"github.com/rock-go/rock/xcall"
	"github.com/valyala/fasthttp"
)

type RequestCtx = fasthttp.RequestCtx

type vRouter struct {
	lua.Super

	//获取名称
	name string

	//上次修改时间
	mtime int64 //时间

	//router中的命令
	accessFormat string
	accessEncode string
	accessRegion string
	accessFn     func(ctx *RequestCtx) []byte

	accessOutputSdk lua.Writer
	accessRegionSdk *region.Region

	//handler处理脚本路径
	handler string

	close       *lua.LFunction
	interceptor *lua.LFunction

	//缓存路由
	r *router.Router
}

func newRouter(co *lua.LState) *vRouter {
	tab := co.CheckAny(1)
	r := router.New()
	r.PanicHandler = panicHandler

	v := &vRouter{
		r:            r,
		accessFormat: "",
		accessRegion: "",
		accessEncode: "line",
	}

	if tab.Type() != lua.LTTable {
		return v
	}

	tab.(*lua.LTable).ForEach(func(key lua.LValue, val lua.LValue) {
		switch key.String() {
		case "access_format":
			v.accessFormat = val.String()

		case "access_encode":
			v.accessEncode = val.String()

		case "access_region":
			v.accessRegion = val.String()

		case "region":
			v.accessRegionSdk = checkRegionSdk(co, val)

		case "output":
			v.accessOutputSdk = checkOutputSdk(co, val)

		case "close":
			if val.Type() != lua.LTFunction {
				return
			}
			v.close = val.(*lua.LFunction)
		case "interceptor":
			if val.Type() == lua.LTFunction {
				v.interceptor = val.(*lua.LFunction)
			}
		}
	})

	v.accessFn = compileAccessFormat(v.accessFormat, v.accessEncode)
	return v
}

func newLuaRouter(co *lua.LState) int {
	r := newRouter(co)
	co.D = r
	co.Push(co.NewLightUserData(r))
	return 1
}

func (r *vRouter) Close() error {
	if r.close == nil {
		return nil
	}
	co := lua.State()
	defer lua.FreeState(co)

	return xcall.CallByEnv(co, r.close, xcall.Rock)
}

func (r *vRouter) MTime() int64 {
	return r.mtime
}

func (r *vRouter) Option() interface{} {
	return r.handler
}

func (r *vRouter) Name() string {
	return r.name
}

func (r *vRouter) Type() string {
	return "fasthttp.router"
}

func (r *vRouter) handleIndexFn(L *lua.LState, method string) *lua.LFunction {
	fn := func(co *lua.LState) int {
		path := co.CheckString(1)
		chains := checkHandleChains(co)
		r.r.Handle(method, path, func(ctx *RequestCtx) { chains.do(ctx, r.handler) })
		return 0
	}
	return L.NewFunction(fn)
}

func (r *vRouter) anyIndexFn(L *lua.LState) *lua.LFunction {
	fn := func(co *lua.LState) int {
		path := co.CheckString(1)
		chains := checkHandleChains(co)
		r.r.ANY(path, func(ctx *fasthttp.RequestCtx) { chains.do(ctx, r.handler) })
		return 0
	}

	return L.NewFunction(fn)
}

func (r *vRouter) notFoundIndexFn(L *lua.LState) *lua.LFunction {
	fn := func(co *lua.LState) int {
		chains := checkHandleChains(co)
		r.r.NotFound = func(ctx *fasthttp.RequestCtx) { chains.do(ctx, r.handler) }
		return 0
	}
	return L.NewFunction(fn)

}

func (r *vRouter) fileIndexFn(L *lua.LState) *lua.LFunction {
	fn := func(vm *lua.LState) int {
		n := vm.GetTop()
		path := vm.CheckString(1)
		root := vm.CheckString(2)
		fs := &fasthttp.FS{
			Root:               root,
			IndexNames:         []string{"index.html"},
			GenerateIndexPages: true,
			AcceptByteRange:    true,
		}

		if n == 3 {
			fn := vm.CheckFunction(3)
			fs.PathRewrite = func(ctx *fasthttp.RequestCtx) []byte {
				co := newLuaThread(ctx)
				err := xcall.CallByEnv(co, fn, xcall.Rock)
				if err != nil {
					logger.Errorf("%v" , err)
					goto done
				}
			done:
				freeLuaThread(ctx)
				return ctx.Path()
			}
		}

		r.r.ServeFilesCustom(path, fs)

		return 0
	}

	return L.NewFunction(fn)
}

func (r *vRouter) call(co *lua.LState, hook *lua.LFunction) {
	if hook == nil {
		return
	}
	err := xcall.CallByEnv(co, hook, xcall.Rock)
	if err != nil {
		logger.Errorf("http hook call error: %v", err)
	}
}

func (r *vRouter) Index(L *lua.LState, key string) lua.LValue {
	switch key {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS", "TRACE":
		return r.handleIndexFn(L, key)

	case "ANY":
		return r.anyIndexFn(L)

	case "not_found":
		return r.notFoundIndexFn(L)

	case "file":
		return r.fileIndexFn(L)

	}
	return lua.LNil
}

func (r *vRouter) do(ctx *RequestCtx) {
	r.r.Handler(ctx)

	if r.interceptor == nil {
		return
	}

	co := newLuaThread(ctx)
	p := lua.P{
		Fn:      r.interceptor,
		NRet:    0,
		Protect: true,
	}

	err := xcall.CallByParam(co, p, xcall.Rock)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(err.Error())
		return
	}
}
