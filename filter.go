package fasthttp

import (
	"errors"
	"github.com/rock-go/rock/lua"
	"strings"
)

type rule struct {
	raw string
	fn  func(*RequestCtx) bool
}

func newRule( raw string ) rule {
	return rule{raw: raw }
}

var (
	SPACE = " "
	COMMA = ","

	invalidRuleH = errors.New("invalid filter rule var")
	invalidRuleF = errors.New("invalid filter rule format ")
	invalidRuleM = errors.New("invalid filter rule method")
)
func (r *rule) compile() error {
	if r.raw[0] != '$' {
		return invalidRuleH
	}

	rs := strings.SplitN(r.raw[1:] , SPACE , 3)
	if len(rs) != 3 {
		return invalidRuleF
	}

	//filter 变量检测
	for _ , ch := range rs[0] {
		if (ch >= 'a' && ch <= 'z') || ch == '_' { continue }
		return invalidRuleH
	}

	//检测匹配方法
	v := strings.Split(rs[2] , COMMA)
	switch rs[1] {

	//比较等于
	case "==":
		r.fn = func(ctx *RequestCtx) bool {
			u := fsGet(ctx , rs[0]).String()
			if u == "" {
				return false
			}

			for _ , item := range v {
				if strings.EqualFold(item , u) {
					return true
				}
			}
			return false
		}

	case "!=":
		r.fn = func(ctx *RequestCtx) bool {
			u := fsGet(ctx , rs[0]).String()
			if u == "" {
				return true
			}

			for _ , item := range v {
				if strings.EqualFold(item , u) {
					return false
				}
			}
			return true
		}

	//异常的方法名
	default:
		return invalidRuleM

	}

	return nil
}

type filter struct {
	lua.Super
	data []rule
}

func newFilter() *filter {
	return &filter{data: make([]rule , 0)}
}

func (f *filter) append(raw string) error {
	r := newRule(raw)

	if e := r.compile(); e != nil {
		return e
	}

	f.data = append(f.data , r)
	return nil
}

func (f *filter) do( ctx *RequestCtx ) bool {
	n := len(f.data)
	if n == 0 {
		return true
	}

	for i := 0 ; i < n ; i++ {
		if !f.data[i].fn(ctx) {
			return  false
		}
	}

	return true
}

func newLuaFilter(L *lua.LState) int {
	tab := L.CheckTable(1)

	f := newFilter()
	tab.ForEach(func(key lua.LValue, val lua.LValue) {
		if key.Type() != lua.LTNumber {
			L.RaiseError("fasthttp filter must be arr , got table")
			return
		}

		if val.Type() != lua.LTString {
			L.RaiseError("invalid fasthttp filter rule")
			return
		}
		if e := f.append(val.String()); e != nil {
			L.RaiseError("invalid filter rule error %v" , e)
			return
		}
	})

	L.Push(L.NewLightUserData(f))
	return 1
}

var (
	invalidFilterType  = errors.New("invalid filter type , must be userdata")
	invalidFilterValue = errors.New("invalid filter value")
)
func checkFilter(val lua.LValue) (*filter , error){
	if val.Type() != lua.LTLightUserData {
		return nil , invalidFilterType
	}

	f , ok := val.(*lua.LightUserData).Value.(*filter)
	if !ok {
		return nil , invalidFilterValue
	}

	return f , nil
}
