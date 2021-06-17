package fasthttp

import (
	"sync"
	"github.com/rock-go/rock/lua"
)

var (
	once       sync.Once
	handlePool *pool
	routerPool *pool
	ctxMeta    lua.UserKV
	fsCtx      *lua.LightUserData
)

func init() {
	once.Do(func() {
		fsCtx = newContext()
		ctxMeta = newCtxMeta()
		handlePool = newPool()
		routerPool = newPool()
		go routerPoolSync()
		go handlePoolSync()
	})
}