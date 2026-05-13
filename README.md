# Doc Svr

一个基于 Go Fiber v3 的静态网页渲染服务器，用来承载数据展示页面、博客更新和轻量知识库内容。站点内容使用 Markdown 编写，模板和静态资源打包进二进制，适合单文件部署和容器化交付。

## 适合场景

- 团队周报、月报和运营数据简报
- 产品博客、更新日志和专题文章
- 内部知识库、技术说明页、项目首页

## 技术选型

- Go Fiber v3：负责 HTTP 路由、中间件和服务启动
- compress + etag + requestid + logger：负责压缩、缓存标识和基础日志
- Goldmark：渲染 Markdown 为 HTML
- html/template：服务端模板渲染，避免引入额外前端构建链
- embed.FS：把模板、样式和内容打包进二进制

## 当前目录结构

```text
.
├── build.sh                # 脚本入口，构建/运行/校验/镜像
├── content/
│   └── posts/              # Markdown 内容
├── dockerfile              # 多阶段容器构建
├── internal/
│   └── site/               # 内容加载与 Markdown 渲染
├── main.go                 # Fiber 应用入口与路由
├── makefile                # 常用开发命令
├── web/
│   ├── assets/             # 样式资源
│   └── templates/          # HTML 模板
└── vendor/                 # vendored 依赖，默认走 -mod=vendor
```

## 页面与设计规范

- 首页：Hero + 指标卡片 + 工作流说明 + 精选文章
- 列表页：文章归档 + 侧边规范说明
- 详情页：统一长文阅读布局，适合博客与数据解读
- 配色：深青绿作为主色，铜橙做强调色，暖米色背景强化专业内容站点氛围
- 响应式：桌面双列，移动端自动回落为单列

## Markdown 规范

文章存放在 content/posts，文件名即 slug，例如 content/posts/weekly-ops.md 对应 /blog/weekly-ops。

每篇文章建议包含 Front Matter：

```yaml
---
title: 本周数据复盘
summary: 首页摘要与 SEO 描述
date: 2026-05-12
tags:
  - weekly
  - dashboard
featured: true
---
```

字段说明：

- title：文章标题
- summary：摘要，建议 60 到 110 字
- date：发布时间，支持 2006-01-02 / RFC3339
- tags：标签数组
- featured：首页精选位

项目额外提供了 content/posts/_template.md 作为起稿模板。以 _ 开头的 Markdown 文件不会被渲染成文章。

## 本地运行

```bash
make run
```

或：

```bash
./build.sh run
```

默认地址： http://localhost:3000

## 构建与校验

```bash
make build
make check
```

或：

```bash
./build.sh build
./build.sh check
```

生成的二进制位于 bin/doc-svr。

## Docker

构建镜像：

```bash
make docker-build
```

运行容器：

```bash
docker run --rm -p 3000:3000 doc-svr:dev
```

## 可维护性建议

- 继续新增内容时，优先保持 Markdown 驱动，避免过早引入重前端工程
- 如果后续需要图表，优先嵌入 SVG 或外部 iframe，再考虑 JS 图表组件
- 若要扩展栏目，可在 content 下增加专题目录，并在 internal/site 中补分类逻辑
- 正式上线前建议替换 main.go 中的站点 owner / email / location 为真实信息