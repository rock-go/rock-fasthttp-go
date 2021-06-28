package fasthttp

import (
	"sync"
	"github.com/rock-go/rock/lua"
)

var (
	once       sync.Once
	handlePool *pool
	routerPool *pool
	Context    *lua.AnyData
)

func init() {
	once.Do(func() {
		Context = newContext()
		handlePool = newPool()
		routerPool = newPool()
		go routerPoolSync()
		go handlePoolSync()
	})
}