package fasthttp

import (
	"errors"
	"github.com/rock-go/rock/lua"
)

var (
	AccessFormat = "http_time,server_addr,server_port,remote_addr,host,path"
)

type region interface {
	Search( string ) (int64 , []byte , error)
}

type config struct {
	//基础配置
	name           string
	listen         string
	network        string
	router         string
	handler        string
	keepalive      string
	reuseport      string
	notFound       string
	daemon         string

	//设置access日志
	access         string
	accessFormat   string
	accessEncode   string
	accessRegion   string

	//下面对象配置
	accessRegionSdk   region
	accessOutputSdk   lua.Writer
}

func newConfig(L *lua.LState) *config {
	tab := L.CheckTable(1)
	cfg := &config{}

	tab.ForEach(func(key lua.LValue, val lua.LValue) {
		if key.Type() != lua.LTString {
			L.RaiseError("invalid config table , got arr")
			return
		}

		switch key.String() {
		case "name": cfg.name = val.String()
		case "daemon": cfg.daemon = val.String()
		case "listen": cfg.listen = val.String()
		case "network": cfg.network = val.String()
		case "routers": cfg.router = val.String()
		case "handler": cfg.handler = val.String()
		case "not_found": cfg.notFound = val.String()
		case "reuseport": cfg.reuseport = val.String()
		case "keepalive": cfg.keepalive = val.String()

		case "access_log": cfg.access = val.String()
		case "access_format": cfg.accessFormat = val.String()
		case "access_encode": cfg.accessEncode = val.String()
		case "access_region": cfg.accessRegion = val.String()

		case "region": cfg.accessRegionSdk = checkRegionSdk(L , val)
		case "output": cfg.accessOutputSdk = checkOutputSdk(L , val)

		default:
			L.RaiseError("invalid fasthttp config %s field" , key.String() )
			return
		}
	})
	return cfg
}

func (cfg *config) verify() error {
	if cfg.name == "" {
		return errors.New("invalid name")
	}

	return nil
}