package fasthttp

import (
	"errors"
	"github.com/rock-go/rock/logger"
	"github.com/rock-go/rock/lua"
	"reflect"
	"time"
)

var vhostTypeof = reflect.TypeOf((*vhost)(nil)).String()

type vhost struct {
	lua.Super

	code string
	name string

	r    *vRouter
	fss  *server
}

func newfsHost(L *lua.LState) *vhost {
	tab := L.CheckTable(1)

	app := &vhost{}

	tab.ForEach(func(key lua.LValue, val lua.LValue) {
		switch key.String() {
		case "name":
			app.name = val.String()
		case "server":
			if val.Type() != lua.LTLightUserData {
				L.RaiseError(" vhost server must  web server  , got %s" , val.Type().String())
				return
			}

			var ok bool
			app.fss , ok = val.(*lua.LightUserData).Value.(*server)
			if !ok {
				L.RaiseError(" invalid vhost serve")
				return
			}
		}
	})

	if e := app.verify(L); e != nil {
		L.RaiseError("vhost %s" , e)
		return nil
	}

	app.r = newRouter(L , tab)
	app.code = L.CodeVM()
	app.V(lua.INIT , vhostTypeof)
	return app

}

func newLuaFasthttpHost(L *lua.LState) int {
	app := newfsHost(L)

	proc := L.NewProc(app.name , vhostTypeof)
	if proc.IsNil() {
		proc.Set(app)
		L.Push(proc)
		return 1
	}

	obj := proc.Value.(*vhost)

	//如果切换web服务中心
	if obj.fss.Name() != app.fss.Name() {
		obj.fss.vhost.clear(obj.Name())
		logger.Errorf("%s web %s vhost clear from %s" , obj.fss.Name(), obj.Name())
	}

	obj.fss = app.fss
	obj.r = app.r

	L.Push(proc)
	return 1
}

func (v *vhost) Index(L *lua.LState , key string) lua.LValue {
	return v.r.Get(L , key)
}

func (v *vhost) Name() string {
	return v.name
}

func (v *vhost) Start() error {
	v.fss.vhost.insert(v.name , v.r)
	return nil
}

func (v *vhost) Close() error {
	v.fss.vhost.clear(v.name)
	v.V(lua.CLOSE , time.Now())
	return nil
}

func (v *vhost) verify(L *lua.LState) error {
	if L.CodeVM() == "" {
		return errors.New("vhost not allow thread")
	}

	if v.fss == nil {
		return errors.New("vhost not found fss")
	}

	if v.name == "" {
		return errors.New("vhost invalid hostname")
	}

	return nil
}