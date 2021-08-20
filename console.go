package fasthttp

import (
	"github.com/rock-go/rock/lua"
)

func (srv *server) Header(out lua.Printer) {
	out.Printf("type: %s", srv.Type())
	out.Printf("uptime: %s", srv.U.Format("2006-01-02 15:04:06"))
	out.Printf("version: v1.0.5")
	out.Println("")
}

func (srv *server) Show(out lua.Printer) {
	srv.Header(out)
	out.Printf("name  = %s", srv.Name())
	out.Printf("network = %s", srv.cfg.network)
	out.Printf("listen = %s", srv.cfg.listen)
	out.Printf("routers = %s", srv.cfg.router)
	out.Printf("handler = %s", srv.cfg.handler)
	out.Printf("not_found = %s", srv.cfg.notFound)
	out.Printf("reuseport = %s" , srv.cfg.reuseport)
	out.Printf("keepalive = %s", srv.cfg.keepalive)
	out.Printf("read_timetout = %d", srv.cfg.readTimeout)
	out.Printf("access_format = %s", srv.cfg.accessFormat)
	out.Printf("access_encode = %s", srv.cfg.accessEncode)
	out.Printf("access_region = %s", srv.cfg.accessRegion)
	out.Printf("region = %s", srv.cfg.accessRegionSdk.Name())
	out.Printf("output = %s", srv.cfg.accessOutputSdk.Name())
}

func (srv *server) Help(out lua.Printer) {
	srv.Header(out)
}
