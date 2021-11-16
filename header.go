package fasthttp

import (
	"errors"
	"github.com/rock-go/rock/lua"
	"strings"
)

type headerKV struct {
	key string
	val string
}

type header struct {
	lua.Super
	data []headerKV
}

func (h *header) Type() string {
	return "fasthttp.header"
}

func (h *header) Name() string {
	return h.Type()
}

func (h *header) Len() int {
	return len(h.data)
}

func (h *header) Set(key string, val string) {
	n := h.Len()
	for i := 0; i < n; i++ {
		item := &h.data[i]
		if strings.EqualFold(item.key, key) {
			item.key = key
			item.val = val
			return
		}
	}

	h.data = append(h.data, headerKV{key, val})
}

func (h *header) ForEach(fn func(string, string)) {
	n := h.Len()
	for i := 0; i < n; i++ {
		item := &h.data[i]
		fn(item.key, item.val)
	}
}

func newHeader() *header {
	return &header{}
}

func toHeader(L *lua.LState , val lua.LValue) *header {
	if val.Type() != lua.LTTable {
		L.RaiseError("header must be table")
		return nil
	}
	tab := val.(*lua.LTable)
	h := newHeader()

	tab.ForEach(func(key lua.LValue, val lua.LValue) {
		if key.Type() != lua.LTString {
			L.RaiseError("fasthttp header must be table , got array")
			return
		}

		switch val.Type() {

		case lua.LTString:
			h.Set(key.String(), val.String())

		case lua.LTNumber:
			h.Set(key.String(), val.String())

		default:
			L.RaiseError("invalid header , must be string , got %s", val.Type().String())
		}
	})

	return h
}

func newLuaHeader(L *lua.LState) int {
	tab := L.CheckTable(1)
	h := newHeader()
	tab.ForEach(func(key lua.LValue, val lua.LValue) {
		if key.Type() != lua.LTString {
			L.RaiseError("fasthttp header must be table , got array")
			return
		}

		switch val.Type() {

		case lua.LTString:
			h.Set(key.String(), val.String())

		case lua.LTNumber:
			h.Set(key.String(), val.String())

		default:
			L.RaiseError("invalid header , must be string , got %s", val.Type().String())
		}
	})

	L.Push(L.NewAnyData(h))
	return 1
}

var (
	invalidHeaderType  = errors.New("invalid header type , must be userdata")
	invalidHeaderValue = errors.New("invalid header value")
)

func checkHeader(val lua.LValue) (*header, error) {
	if val.Type() != lua.LTANYDATA {
		return nil, invalidHeaderType
	}

	h, ok := val.(*lua.AnyData).Value.(*header)
	if !ok {
		return nil, invalidHeaderValue
	}
	return h, nil
}
