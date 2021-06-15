package fasthttp

import (
	"sync"
	"github.com/rock-go/rock/lua"
)

var (
	once       sync.Once
	handlePool *pool
	routerPool *pool
	instance   *lua.LightUserData
)

func init() {
	once.Do(func() {
		instance = newContext()
		handlePool = newPool()
		routerPool = newPool()
		go routerPoolSync()
		go handlePoolSync()
	})
}