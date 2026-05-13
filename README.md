# Doc Svr

一个基于 Go Fiber v3 的本地静态页面入口服务。它会在启动时扫描 public 目录下的子页面入口，把 public/{route}/index.html 自动挂载成可访问路由，并通过首页统一展示所有已注册页面。

当前项目不是 Markdown 博客渲染器，而是一个轻量的静态页面聚合壳层：模板、样式和 public 下的页面资源都会通过 embed.FS 打进单个二进制，便于本地运行、容器部署和内网分发。

## 当前能力

- 自动发现 public/**/index.html 并注册为路由
- 首页统一展示所有已注册页面入口，并可生成到 public/index.html
- 子页面通过共享 workspace 壳层接入树形导航、breadcrumb 和 iframe 视图区
- 提供 healthz、robots.txt、favicon 和共享静态资源缓存配置
- 使用 vendor 模式构建，降低外部依赖波动带来的影响

## 当前目录结构

```text
.
├── .dockerignore           # Docker 构建上下文过滤
├── README.md               # 项目总说明
├── build.sh                # 构建/运行/校验/镜像入口
├── deploy/                 # 预留部署目录，当前为空
├── dockerfile              # 多阶段镜像构建文件
├── docs/                   # 维护文档与 Fiber 使用说明
├── go.mod
├── go.sum
├── internal/
│   └── site/               # 站点路由、页面发现、模板渲染
├── main.go                 # Fiber 应用入口
├── makefile                # 常用开发命令
├── public/                 # 实际被扫描并注册的静态页面目录
│   ├── benchmarks/
│   ├── index.html          # 由 generate-home 落盘生成的首页
│   ├── notify/
│   └── storage/
├── scripts/
│   ├── install.sh          # 环境安装脚本
│   └── run.sh              # 本地后台运行脚本
├── vendor/                 # vendored 依赖
└── web/
    ├── assets/             # 首页、favicon 和共享样式
    └── templates/          # 服务端模板
```

## 运行方式

默认端口是 4000。可以通过 PORT 环境变量覆盖，也可以直接使用命令行参数 --port。

```bash
make run
```

指定端口：

```bash
make run PORT=4105
```

或：

```bash
./build.sh run
go run . --port 4105
```

后台启动：

```bash
./scripts/run.sh
```

访问地址： <http://localhost:4000>

## 构建与校验

```bash
make build
make check
make smoke
```

或：

```bash
./build.sh build
./build.sh check
```

生成的二进制位于 bin/doc-svr。

如果只想刷新首页静态文件，可以直接执行：

```bash
make generate-home
```

它会把当前首页模板渲染结果输出到 public/index.html，方便后续打包或静态预览。

## Docker

构建镜像：

```bash
make docker-build
```

或：

```bash
./build.sh docker
```

运行容器：

```bash
docker run --rm -p 4000:4000 -e PORT=4000 doc-svr:dev
```

说明：镜像构建依赖 Docker 能拉取基础镜像；如果构建失败且报 registry 超时，优先检查本机外网访问或镜像代理。

## Fiber 使用说明

本项目对 Fiber 的使用方式已经固定成一条简单链路：

- 在 main.go 中创建 fiber.App，并统一挂载 requestid、logger、etag、compress 等中间件
- 通过 /assets 暴露 web/assets 内的共享静态资源
- 通过 /favicon.ico 和 /favicon.svg 提供浏览器图标入口
- 把模板系统和 public 文件系统交给 internal/site.SiteApp 管理
- 由 SiteApp 负责注册首页、健康检查、workspace 壳层页和所有原始静态页面路由

详细说明见 docs/fiber-usage.md。

## 页面接入规范

新增一个页面时，只需要满足下面的目录约定：

```text
public/
└── your-page/
    ├── index.html
    └── other-assets...
```

服务启动后会自动挂载：

- public/your-page/index.html -> /your-page
- public/your-page/index.html -> /_pages/your-page/
- public/your-page/xxx.js -> /_pages/your-page/xxx.js

如果页面资源里使用相对路径，服务会自动补 base href，避免静态资源引用错位。

## Workspace 壳层说明

- 所有业务静态页都会挂到一个共享 workspace 容器中
- 外层负责树形导航、breadcrumb、移动端抽屉和 iframe 可视区
- 原始页面继续由 /_pages/{route}/ 提供，不会被壳层样式直接污染
- 首页和运行脚本会在构建前自动生成 public/index.html，保持入口页可落盘复用

## 当前建议

- 新页面优先继续走 public/{route}/index.html 约定，避免手工加路由
- 共享样式尽量落到 web/assets/site.css，避免每个页面各自重复维护
- 大体积调试页会被 embed 进二进制，发布前要关注二进制体积和冷启动时间
- 部署前建议替换 main.go 中 Profile 里的 owner、email、location 等信息
- 当前项目的结构性观察和后续改进建议记录在 docs/project-notes.md