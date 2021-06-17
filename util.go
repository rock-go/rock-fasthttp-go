package fasthttp

import (
	"errors"
	"github.com/rock-go/rock/lua"
	"github.com/rock-go/rock/logger"
	"github.com/rock-go/rock/xcall"
	"fmt"
	"os"
	"time"
	"strconv"
	"strings"
	"bytes"
	"github.com/rock-go/rock/region"
)

func checkHandleChains(L *lua.LState) *HandleChains {
	n := L.GetTop()
	if n < 2 {
		logger.Errorf("invalid args , #1 must string , #n+1 must be http.handler")
		return nil
	}

	hc := newHandleChains(n - 1)
	var val lua.LValue
	for i := 2 ; i<=n ; i++ {

		val = L.Get(i)
		switch val.Type() {

		//判断是否为加载
		case lua.LTString:
			hc.Store(val.String() , VHSTRING, i - 2)

		case lua.LTLightUserData:
			hd , ok := val.(*lua.LightUserData).Value.(*handle)
			if !ok {
				L.RaiseError("%d invalid http.handler" , i)
				return nil
			}
			hc.Store(hd , VHANDLER, i - 2)

		case lua.LTFunction:
			hc.Store(val.(*lua.LFunction) , VHFUNC , i - 2)

		default:
			L.RaiseError("invalid handle value")
			return nil
		}
	}

	return hc
}


var (
	notFoundRouter = errors.New("not found router in co")
	invalidRouter  = errors.New("invalid router in co")
)
func checkRouter(L *lua.LState) (*vRouter, error) {
	if L.D == nil {
		return nil , notFoundRouter
	}

	r  , ok := L.D.(*vRouter)
	if !ok {
		return nil , invalidRouter
	}

	return r , nil
}

func checkRegionSdk(L *lua.LState , val lua.LValue) *region.Region {

	switch val.Type() {
	case lua.LTNil:
		return nil

	case lua.LTLightUserData:
		r , ok := val.(*lua.LightUserData).Value.(*region.Region)
		if !ok {
			L.RaiseError("invalid region sdk")
			return nil
		}
		return r

	default:
		//todo
	}

	L.RaiseError("invalid region object , got %s" , val.Type().String())
	return nil
}

func checkOutputSdk(L *lua.LState , val lua.LValue) lua.Writer {
	switch val.Type() {
	case lua.LTNil:
		return nil
	case lua.LTLightUserData:
		w, ok := val.(*lua.LightUserData).Value.(lua.Writer)
		if ok {
			return w
		}

	default:
		//todo
	}

	L.RaiseError("invalid output object , got %s" , val.Type().String())
	return nil
}

func compileAccessFormat(format string , encode string) func(*RequestCtx) []byte {
	if format == "" || format == "off" {
		return nil
	}
	format = strings.TrimSpace(format)

	a := strings.Split(format , ",")
	var fn func(ctx *RequestCtx) []byte
	switch encode {
	case "json":
		fn = func(ctx *RequestCtx) []byte {
			n := len(a)
			if n == 0 {
				return []byte("[]")
			}

			buff := lua.NewJsonBuffer("")
			for i := 0 ; i < n ;i++ {
				item := a[i]
				val := fsGet(ctx , item)
				if strings.HasPrefix(item , "http_") {
					item = item[5:]
				}

				//避免JSON 最后一个逗号
				if i == n - 1 {
					buff.EOF = true
				}

				switch val.Type() {
				case lua.LTNumber:
					buff.WriteKI(item , int(val.(lua.LNumber)))
				default:
					buff.WriteKV(item , val.String())
				}
			}
			buff.End()
			return buff.Bytes()
		}

	case "line":
		fn = func(ctx *RequestCtx) []byte {
			n := len(a)
			if n == 0 {
				return byteNull
			}
			var buff bytes.Buffer
			for i:=0;i<n;i++ {

				if i != 0 {
					buff.WriteByte(' ')
				}
				item := a[i]
				val := fsGet(ctx ,item)

				//去除前缀
				if strings.HasPrefix(item , "http_") {
					item = item[5:]
				}
				switch val.Type() {
				case lua.LTNumber:
					buff.WriteString(strconv.Itoa(int(val.(lua.LNumber))))
				default:
					buff.Write(lua.S2B(val.String()))
				}
			}
			return buff.Bytes()
		}

	default:
		return nil
	}

	return fn
}

func compileHandle(filename string) (*handle , error) {
	//重新获取
	co := lua.State()
	defer lua.FreeState(co)
	stat , err := os.Stat(filename)
	if err != nil {
		return nil , err
	}

	fn , err := co.LoadFile(filename)
	if err != nil {
		return nil , err
	}

	p := lua.P{
		Fn: fn,
		NRet: 1,
		Protect: true,
	}

	err = xcall.CallByParam(co , p , xcall.Rock)
	if err != nil {
		return nil , err
	}

	n := co.GetTop()
	if n != 1 {
		return nil , errors.New("not found handle object , top: %d" + strconv.Itoa(n))
	}

	val := co.Get(n)
	if val.Type() != lua.LTLightUserData {
		return nil , errors.New("invalid handle light userdata")
	}

	hd , ok := val.(*lua.LightUserData).Value.(*handle)
	if !ok {
		return nil , errors.New("invalid handle object")
	}
	hd.mtime = stat.ModTime().Unix()
	hd.name = filename

	logger.Errorf("handle %s compile succeed" , filename)
	return hd , nil

}
func requireHandle( path , name string ) (*handle , error) {
	filename := fmt.Sprintf("%s/%s.lua" , path , name)

	//查看缓存
	item := handlePool.Get(filename)
	if item != nil {
		return item.val.(*handle) , nil
	}

	hd , err := compileHandle(filename)
	if err != nil {
		return nil , err
	}

	handlePool.insert(filename , hd)
	return hd , nil

}

func compileRouter(filename string , handler string) (*vRouter , error) {

	//重新获取
	stat , err := os.Stat(filename)
	if err != nil {
		return nil , err
	}
	var r *vRouter

	co := lua.State()
	defer lua.FreeState(co)

	//执行配置脚本
	err = xcall.DoFileByEnv(co , filename , xcall.Rock)
	if err != nil {
		return nil , err
	}

	r , err = checkRouter(co)
	if err != nil {
		return nil , err
	}

	r.handler = handler
	r.name = filename
	r.mtime = stat.ModTime().Unix()
	logger.Errorf("router %s compile succeed" , filename)
	return r , nil

}

func requireRouter(path , handler, host string) (*vRouter , error) {
	filename := fmt.Sprintf("%s/%s.lua" , path , host)

	//查看缓存
	item := routerPool.Get(filename)
	if item != nil {
		return item.val.(*vRouter) , nil
	}

	r , err := compileRouter(filename , handler)
	if err != nil {
		return nil , err
	}

	routerPool.insert(filename , r)
	return r , err
}

func routerPoolSync() {
	tk := time.NewTicker(1000 *time.Millisecond)
	for range tk.C {
		routerPool.sync(func(item *poolItem , del *int){
			stat , err := os.Stat(item.key)
			var r *vRouter
			if os.IsNotExist(err) {
				*del++
				logger.Errorf("router %s delete" , item.key)
				goto next
			}

			r = item.val.(*vRouter)
			if stat.ModTime().Unix() == r.mtime {
				goto next
			}

			if xr , e := compileRouter(item.key , r.handler); e != nil {
				logger.Errorf("router %s sync update fail , err: %v" , item.key , e)
			} else {
				logger.Errorf("router %s sync update succeed " , item.key)
				item.val = xr
			}
		next:
		})
	}
}

func handlePoolSync() {
	tk := time.NewTicker(1000 *time.Millisecond)
	for range tk.C {
		handlePool.sync(func(item *poolItem, del *int) {
			stat, err := os.Stat(item.key)
			var hd *handle
			if os.IsNotExist(err) {
				*del++
				logger.Errorf("handle %s delete" , item.key)
				goto next
			}

			hd = item.val.(*handle)
			if stat.ModTime().Unix() == hd.mtime {
				goto next
			}

			if xhd, e := compileHandle(item.key); e != nil {
				logger.Errorf("handle %s sync update fail , err: %v", item.key, e)
			} else {
				logger.Errorf("handle %s sync update succeed ", item.key)
				item.val = xhd
			}
		next:
		})
	}
}