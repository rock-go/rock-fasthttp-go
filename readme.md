# rock-fasthttp-go
主要是用的fasthttp和lua的开发框架 可以灵活的处理业务 利用lua的热更新和加载的原理

## SETUP启动配置
启动配置一般包含启动配置
```lua
    local http = fasthttp.server{
    
        -- proc 服务名称 
        name = "demo_fasthttp", 
    
        -- network
        network = "tcp", --监听的协议
    
        -- 监听端口
        listen  = "0.0.0.0:9090", --监听的端口
   
        -- 日志格式 , 关闭：off 
        access_format = "time,server_addr,server_port,remote_addr,host,path,query,ua,referer,status,content-lenght,region_info,http_risk",
    
        -- 日志的格式 
        access_encode = "json",
    
        -- 记录IP的位置信息
        access_region = "x-real-ip",
    
        -- 路由查找路径,如:www.a.com.lua , 通过主机头查找
        routers = "resource/fasthttp/server.d",
    
        -- uri handle处理逻辑 可以匿名或者公共库查找, 这里就是公共库
        handler = "resource/fasthttp/handle.d",
    
        -- 默认没有发现的放回结果 
        not_found = "not_found",
        
        -- 位置处理SDK
        -- region = rock.region{},

        -- 日志文件输出SDK
        -- output = rock.file{},
    }

    proc.start(http)
```

## router配置
利用的fasthttp的router路由逻辑，完成默认路由的注入 ， 利用fasthttp的快速匹配模式完成路由查找
下面是www.a.com的主机的配置,文件路径:resource/fasthttp/server.d/www.a.com.lua
注意: SETUP 中的routers配置目录下
### fasthttp.router{}
新建一个router对象
```lua
    local r = fasthttp.router{
    -- 日志格式 , 关闭：off 
    access_format = "time,server_addr,server_port,remote_addr,host,path,query,ua,referer,status,content-lenght,region_info,http_risk",

    -- 日志的格式 
    access_encode = "json",

    -- 记录IP的位置信息
    access_region = "x-real-ip",
    
    --
    output = rock.file{},
    
    --regionsdk
    region = rock.region{}


}
```

完整的例子 
```lua
local ctx = fasthttp.ctx    -- 用户请求周期变量
local json = rock.json
local function auth() 
    local u = json.decode(ctx.body_raw)
    if u.name == "admin" and u.pass == "123654" then
        ctx.say(json.encode({code=200 , message= u.name .. "login success"}))
        ctx.exit(200) 
        return
    end
    ctx.say(json.encode({code=200 , message= "login fail"}))
    ctx.exit(200)
end

r.POST("/login" , auth) --注册路由

r.GET("/info" , 
    fasthttp.handle{
        code = 200,
        filter = fasthttp.filter{
           "$host == www.a.com,www.b.com,www.c.com",
            "$uri == /info"
        },
        
        header = fasthttp.header{
            ['server'] = "rock-fasthttp-test-v1.0", 
        },
        
        body = "helo",
        hook = function()
            ctx.say("helo by hook")
        end,
        eof = true 
    },
    "defaut_handle" --这个是查找公共库下的handle处理逻辑 ,一般为default_handle.lua
)
```
### 函数说明 
#### 1.路由配置
- router.GET
- router.HEAD
- router.POST
- router.PUT
- router.PATCH
- router.DELETE
- router.CONNECT
- router.OPTIONS
- router.TRACE
- router.POST
- router.ANY 忽略发方法名 , 注意: router.ANY("*" , fasthttp.handle...)
  
- 语法:  r.METHOD(path string , fasthttp.handle ... )
- 参数 path： 代表路径的 完全兼容 fasthttp.router的路径语法 如:/api/{name}/{val:*}
- 参数 handle: 就是用fasthttp.handle构造的对象

#### 2. ctx周期变量的使用
- fasthttp.ctx.say(string...)
```lua
    local ctx = fasthttp.ctx
    local r = fasthttp.router()

    r.GET("/" , function()
        ctx.say("helo")
    end)
```
- fasthttp.ctx.append(string...) 
```lua
    local ctx = fasthttp.ctx
    local r = fasthttp.router()
    r.GET("/" , function()
        ctx.say("helo")
        ctx.append(" rock-go")
    end)
```
- fasthttp.ctx.exit(int) 
```lua
    local ctx = fasthttp.ctx
    local r = fasthttp.router()
    r.GET("/" , function()
        ctx.say("helo")
        ctx.append(" rock-go")
        ctx.exit(200)
    end)
```
- fasthttp.ctx.set_header(name1 , value1 , name2, value2,name3 , value3)
```lua
  local ctx = fasthttp.ctx
  local r = fasthttp.router()
  r.GET("/" , function()
      ctx.set_healder("content-length",1 , "uuid","x-x")
      ctx.say("helo")
  end)
```
- fasthttp.ctx.eof()
```lua
    local ctx = fasthttp.ctx
    local r = fasthttp.router()

    r.GET("/" , function()
        ctx.say("helo")
        ctx.eof() --关闭继续运行
    end)
```

- fasthttp.ctx.arg_*
- 作用: 读取用户请求参数的值
- 语法： fasthttp.ctx.arg_name , fasthttp.ctx.arg_a
```lua
    -- http://www.a.com/a?name=edx&a=123
    local ctx = fasthttp.ctx
    local name = ctx.arg_name or "" --name=edx
    local a = ctx.arg_a or "" -- a=123
```

- fasthttp.ctx.post_*
- 作用： 读取用户POST参数
- 语法： fasthttp.ctx.post_value
```lua
    -- POST /api HTTP/1.1
    -- Host: www.a.com
    -- Rule: sqli-deny
    --
    -- name=edunx&value=123
    
    local ctx = fasthttp.ctx
    local say = http.response.say

    say(ctx.post_name , " " , ctx.post_value)
```

- fasthttp.ctx.param_*
- 作用： 读取路由中的param
- 语法： fasthttp.ctx.param_name , fasthttp.ctx.param_val
```lua
 -- http://www.a.com/api/admin/123456
 -- 路由 r.GET("/api/{name}/{val:*}

    local ctx = fasthttp.ctx

    local name = ctx.param_name 
    local val = ctx.param_val

    ctx.say(name , " " , val)
```

- fasthttp.ctx.http_*
- 作用：获取请求header头里面的参数
- 语法： fasthttp.ctx.http_rule , fasthttp.ctx.http_user_agent
```lua
    -- GET / HTTP/1.1
    -- Host: www.a.com
    -- Rule: sqli-deny

    local ctx = fasthttp.ctx
    local rule = ctx.http_rule
```
- fasthttp.ctx.cookie_*
- 作用： 获取cookie里的某个具体字段
- 语法： fasthttp.ctx.cookie_session

```lua
    -- GET / HTTP/1.1
    -- Host: www.a.com
    -- cookie:session=123x

    local ctx = fasthttp.ctx
    local v = ctx.cookie_session
    ctx.say(v)
```

- fasthttp.ctx.remote_addr
- 作用: 获取connection的四层IP地址
- 语法: fasthttp.ctx.remote_addr
```lua
    local addr = fasthttp.ctx.remote_addr
``` 
- fasthttp.ctx.remote_port
- 作用: 获取connection的四层端口
- 语法: fasthttp.ctx.remote_port
```lua
    local port = fasthttp.ctx.remote_port
``` 
- fasthttp.ctx.server_addr
- 作用: 获取本地服务器的IP地址
- 语法: fasthttp.ctx.server_addr
```lua
    local addr = fasthttp.ctx.server_addr
``` 
- fasthttp.ctx.server_port
- 作用: 获取本地服务器的端口
- 语法: fasthttp.ctx.server_addr
```lua
    local port = fasthttp.ctx.server_port
``` 
- fasthttp.ctx.host
- 作用: 获取用户请求的主机名
- 语法: fasthttp.ctx.host
```lua
    local host = fasthttp.ctx.host
``` 
- fasthttp.ctx.uri
- 作用: 获取获用户请求的URI
- 语法: fasthttp.ctx.uri
```lua
    -- http://www.a.com/api/info
    local uri = fasthttp.ctx.uri -- /api/info
``` 
- fasthttp.ctx.args
- 作用: 获取用户请求的args字符串
- 语法: fasthttp.ctx.args
```lua
    -- http://www.a.com/api/info?name=admin&val=123
    local args = fasthttp.ctx.args -- name=admin&val=123
``` 
- fasthttp.ctx.request_uri
- 作用: 获取完整的请求URI
- 语法: fasthttp.ctx.request_uri
```lua
    -- http://www.a.com/api/info?name=admin&val=123
    local request = fasthttp.ctx.request_uri -- /api/info?name=admin&val=123
``` 

- fasthttp.ctx.http_time
- 作用: 获取请求时间
- 语法: fasthttp.ctx.http_time
```lua
    local ht = fasthttp.ctx.http_time --2020-01-01 01:02:03.00
``` 
- fasthttp.ctx.cookie_raw
- 作用: 获取cooie的子完整自字符串
- 语法: fasthttp.ctx.cookie_raw
```lua
    local cookie = fasthttp.ctx.cookie_raw
``` 

- fasthttp.ctx.header_raw
- 作用: 获取完整的header字符串
- 语法: fasthttp.ctx.header_raw
```lua
    local raw = fasthttp.ctx.header_raw
``` 
- fasthttp.ctx.content_length
- 作用: 获取获取用户请求的包大小
- 语法: fasthttp.ctx.content_length
```lua
    local len = fasthttp.ctx.content_length
``` 
- fasthttp.ctx.content_type
- 作用: 获取获取用户请求的content_type
- 语法: fasthttp.ctx.content_type
```lua
    local ct = fasthttp.ctx.content_type
``` 
- fasthttp.ctx.body
- 作用: 获取用户的请求的body请求体
- 语法: fasthttp.ctx.body
```lua
    local body = fasthttp.ctx.body
``` 
- fasthttp.ctx.region_cityid
- 作用: 获取用户所在城市的ID
- 语法: fasthttp.ctx.region_cityid
```lua
    local id = fasthttp.ctx.region_cityid
``` 
- fasthttp.ctx.region_info
- 作用: 获取获IP地址位置信息
- 语法: fasthttp.ctx.region_info
```lua
    local info = fasthttp.ctx.region_info
``` 
- fasthttp.ctx.ua
- 作用: 获取user_agent
- 语法: fasthttp.ctx.ua
```lua
    local ua = fasthttp.ctx.ua
``` 
- fasthttp.ctx.referer
- 作用: 获取referer
- 语法: fasthttp.ctx.referer
```lua
    local ref = fasthttp.ctx.referer
``` 
- fasthttp.ctx.status
- 作用: 获取返回状态码
- 语法: fasthttp.ctx.status
```lua
    local status = fasthttp.ctx.status
``` 
- fasthttp.ctx.sent
- 作用: 获取发送数据包的大小
- 语法: fasthttp.ctx.sent
```lua
    local sent = fasthttp.ctx.sent
```

#### 3.handle配置
主要的业务处理逻辑 绑定之前注册路由
```lua
    local ctx = fasthttp.ctx

    fasthttp.handle{
        -- 新建过滤条件
        filter = fasthttp.filter{ 
           "$host == www.a.com,www.b.com", 
           "$uri == /123"
        },
    
        -- 返回状态码
        code = 200, 
        header = fasthttp.header{
            ["content-length"] = 100,
            ["server"] = "hacker"
        },
    
        -- 返回值 
        body = "hello",
    
        -- hook 处理函数逻辑
        hook = function()
            ctx.say("hook helo")
        end,
        
        -- 关闭 路由些其他handle的处理
        eof = true, 
    }
```
#### 4.handle 说明
handle 下面调用方式
```lua
    -- 注册路由的时候直接创建
    r.GET("/123" , fasthttp.handle{ })
    
    -- 注册路由时候 告诉名称， 自动从公共库中加载
    r.GET("/get/info" , "sqli" , "xss" , "not_found")
    
    --直接注入匿名函数
    r.GET("/get/useinfo" , function() end)
```

#### 5.handle 库定义
handle库的定义，一般是文件名调用 查找路径 setup 中的 handler
例如下图中的handle
```lua
    return fasthttp.handle{ --记得一定要return 不然无法获取
        code = 200,
        header = fasthttp.header{},
        hook = function()
            --todo
        end
    }
```