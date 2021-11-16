package fasthttp

import (
	"github.com/rock-go/rock/lua"
	"reflect"
	"sync"
	"time"
)

var (
	once       sync.Once
	handlePool *pool
	routerPool *pool
	Context    *lua.AnyData

	fasthttpTypeOf = reflect.TypeOf((*server)(nil)).String()
)

const (
	thread_uv_key = "__thread_co__"
	eof_uv_key    = "__handle_eof__"
	debug_uv_key  = "__debug__"
)

func init() {
	once.Do(func() {
		Context = newContext()
		handlePool = newPool()
		routerPool = newPool()
		go func() {
			tk := time.NewTicker(800 * time.Millisecond)
			for range tk.C {
				routerPool.sync(compileRouter)
				handlePool.sync(compileHandle)
			}
		}()
	})
}
