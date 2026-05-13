# Fiber 使用说明

## 启动链路

项目入口在 main.go，整体启动顺序如下：

1. 通过 embed.FS 打包 web/templates、web/assets 和 public
2. 初始化 template.FuncMap，用于首页模板里的路由名和占位卡片计算
3. 创建 fiber.App，并统一配置超时、错误处理和 ServerHeader
4. 注册通用中间件：requestid、logger、etag、compress
5. 注册 favicon 路由，避免浏览器默认请求 /favicon.ico 时出现 404
6. 通过 /assets 暴露 web/assets 下的共享静态资源
7. 初始化 internal/site.SiteApp
8. 由 SiteApp 统一注册首页、静态子页、healthz 和 robots.txt

## 中间件说明

当前中间件顺序如下：

- requestid：为每个请求生成请求 ID，便于日志串联
- logger：打印请求状态、耗时、方法、路径和来源 IP
- etag：为静态响应添加缓存标识
- compress：启用压缩，减小首页和静态资源传输体积
- static(/assets)：只负责共享样式和模板用到的静态资源

这个顺序的目的很简单：先打上请求标识，再记录日志，再处理缓存和压缩，最后暴露共享资源。

## 路由组织

项目没有把所有路由直接堆在 main.go，而是做了两层分离：

- main.go：只负责应用生命周期和基础设施
- internal/site/app.go：负责站点级路由和静态页面发现

当前公开路由包括：

- /：首页
- /healthz：健康检查
- /robots.txt：机器人协议
- /favicon.ico、/favicon.svg：图标入口
- /<page>：带 workspace 壳层的页面入口
- /_pages/<page>/：对应页面的原始 index.html
- /_pages/<page>/*：对应页面目录里的静态资源

这样做的目的是把共享导航、breadcrumb、移动端抽屉等 UI 固定在壳层，而把原始业务页面继续隔离在 iframe 里，避免互相污染样式或脚本。

## 如何新增页面

如果你只是新增一个调试页或内部工具页，不需要改 Fiber 路由代码，只需要：

1. 在 public 下新增目录
2. 放入 index.html 和相关资源
3. 重启服务

例如：

```text
public/
└── notify/
    ├── index.html
    ├── app.js
    └── style.css
```

服务会自动把它挂成 /notify，同时原始页面会从 /_pages/notify/ 提供给 iframe 壳层使用。

## 如何新增服务端能力

如果需要新增真正的服务端接口，建议按下面的边界处理：

- 纯页面壳层能力，放在 internal/site
- 单独的 API 能力，拆到新的 internal 子包
- 仍由 main.go 统一完成依赖注入和 Register 调用

不要把新的业务逻辑直接堆进 main.go；main.go 应该继续只保留启动和装配责任。

## 当前使用约束

- 项目默认使用 vendor 构建，命令里会显式带 -mod=vendor
- public 内容会进入 embed.FS，因此所有子页面都会被编译进二进制
- 首页模板可以通过 `go run . generate-home --output public/index.html` 单独落盘
- 当前没有自动化测试，主要依赖 go build ./... 做最小校验

## 推荐的后续演进

- 给 internal/site 增加路由发现和首页渲染的测试
- 把静态页共享导航的样式进一步抽离到共享 CSS，减少服务端字符串拼接体积
- 如果后续 API 变多，可以在 Fiber 上按模块拆分子 router