package fasthttp

import (
	"os"
	"net"
	"time"
	"github.com/rock-go/rock/lua"
	"github.com/valyala/fasthttp"
	"github.com/rock-go/rock/logger"
	"github.com/valyala/fasthttp/reuseport"
)

type server struct {
	lua.Super

	//基础配置
	cfg *config

	//监听
	ln  net.Listener

	//中间对象
	fs  *fasthttp.Server

	//
	accessFn func(*RequestCtx) []byte

	//基础状态
	uptime time.Time

	//状态
	stat   lua.LightUserDataStatus
}

func newServer(cfg *config) *server {
	return &server{ cfg:cfg, stat: lua.INIT }
}

func (ser *server) Name() string {
	return ser.cfg.name
}

func (ser *server) Type() string {
	return "fasthttp.server"
}

func (ser *server) State() lua.LightUserDataStatus {
	return ser.stat
}

func (ser *server) Close() error {
	if ser.stat == lua.CLOSE {
		return nil
	}

	ser.fs.Shutdown()
	routerPool.clear(ser.cfg.router)
	handlePool.clear(ser.cfg.handler)
	ser.stat = lua.CLOSE
	return nil
}

func (ser *server) Listen() (net.Listener , error) {
	if ser.cfg.reuseport == "on" {
		return reuseport.Listen(ser.cfg.network , ser.cfg.listen)
	}
	return net.Listen(ser.cfg.network , ser.cfg.listen)
}

func (ser *server) keepalive() bool {
	if ser.cfg.keepalive == "on" {
		return  true
	}
	return false
}

func (ser *server) notFoundBody(ctx *RequestCtx) {
	ctx.Response.SetStatusCode(fasthttp.StatusNotFound)
	ctx.Response.SetBodyString("not found")
}

func (ser *server) notFound(ctx *RequestCtx) {
	if ser.cfg.notFound == "" {
		ser.notFoundBody(ctx)
		return
	}

	r , err := requireRouter(ser.cfg.router , ser.cfg.handler , ser.cfg.notFound)
	if err != nil {
		if os.IsNotExist(err) {
			ser.notFoundBody(ctx)
			return
		}
		ser.invalid(ctx , err)
		return
	}
	r.r.Handler(ctx)

}

func (ser *server) invalid(ctx *RequestCtx , err error) {
	ctx.Response.SetStatusCode(fasthttp.StatusInternalServerError)
	ctx.Response.SetBodyString(err.Error())
}

func (ser *server) Region(r *vRouter , ctx *RequestCtx ) {
	region := ser.cfg.accessRegion
	sdk := ser.cfg.accessRegionSdk
	if r == nil {
		goto done
	}

	if r.accessRegion != "" {
		region = r.accessRegion
	}

	if r.accessRegionSdk != nil {
		sdk = r.accessRegionSdk
	}

done:
	if region == "" || sdk == nil {
		return
	}

	ip := fsGet(ctx , region).String()
	if len(ip) < 7 {
		return
	}

	info, err := sdk.Search(ip)
	if err != nil {
		logger.Errorf("%v" , err)
		return
	}

	ctx.SetUserValue("region" , info)
	return

}

func (ser *server) Log( r *vRouter , ctx *RequestCtx ) {
	fn  := ser.accessFn
	sdk := ser.cfg.accessOutputSdk

	if r == nil {
		goto done
	}

	if r.accessFn != nil {
		fn = r.accessFn
	}

	//获取每个域名的请求
	if r.accessOutputSdk != nil {
		sdk = r.accessOutputSdk
	}

done:
	//判断全局是否正常
	if sdk == nil || fn == nil {
		return
	}
	sdk.Write(fn(ctx))
}

//编译
func (ser *server) compile() {
	ser.accessFn = compileAccessFormat(ser.cfg.accessFormat , ser.cfg.accessEncode)
}

func (ser *server) Handler( ctx *RequestCtx ) {
	r , err := requireRouter(ser.cfg.router , ser.cfg.handler , lua.B2S(ctx.Host()))

	//是否获取IP地址位置信息
	ser.Region(r , ctx)

	if err != nil {
		if os.IsNotExist(err) {
			ser.notFound(ctx)
			goto done
		}

		ser.invalid(ctx , err)
	}

	r.r.Handler( ctx )
	//运行处理逻辑

done:
	ser.Log(r , ctx)

	//释放co
	freeLuaThread(ctx)
}

func (ser *server) Start() error {
	logger.Errorf("%s fasthttp start ..." , ser.Name())

	ln , err := ser.Listen()
	if err != nil {
		return err
	}

	ser.fs = &fasthttp.Server{
		Handler: ser.Handler,
		TCPKeepalive: ser.keepalive(),
	}
	ser.ln = ln
	ser.compile()

	//延时捕获异常
	tk := time.NewTicker(2 * time.Second)
	go func() {
		err = ser.fs.Serve(ln)
	}()
	<-tk.C

	//处理异常
	if err != nil {
		ser.stat = lua.PANIC
		return err
	}

	ser.stat = lua.RUNNING
	return nil
}

