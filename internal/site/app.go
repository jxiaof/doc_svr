package site

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	fiberstatic "github.com/gofiber/fiber/v3/middleware/static"
)

type Profile struct {
	Name        string
	Tagline     string
	Description string
	Owner       string
	Email       string
	Location    string
	Year        int
	Version     string
}

type StatCard struct {
	Value string
	Label string
	Note  string
}

type PageEntry struct {
	Title       string
	Description string
	Path        string
	Kind        string
	Source      string
	Featured    bool
}

type NavItem struct {
	Label  string
	Path   string
	Active bool
}

type BreadcrumbItem struct {
	Label   string
	Path    string
	Current bool
}

type PageData struct {
	MetaTitle       string
	MetaDescription string
	CurrentPath     string
	Site            Profile
	Stats           []StatCard
	GeneratedAt     string
	ServerAddr      string
	WorkflowSteps   []string
	DesignTokens    []string
	AppPages        []PageEntry
	AppPageCount    int
}

type Config struct {
	Templates *template.Template
	PublicFS  fs.FS
	Profile   Profile
	Port      string
	Version   string
}

type SiteApp struct {
	templates    *template.Template
	publicFS     fs.FS
	profile      Profile
	staticPages  []staticPage
	generatedAt  string
	serverAddr   string
	healthStatus fiber.Map
}

type staticPage struct {
	PageEntry
	Dir string
}

func NewSiteApp(config Config) (*SiteApp, error) {
	staticPages, err := discoverStaticPages(config.PublicFS)
	if err != nil {
		return nil, err
	}

	return &SiteApp{
		templates:   config.Templates,
		publicFS:    config.PublicFS,
		profile:     config.Profile,
		staticPages: staticPages,
		generatedAt: time.Now().Format("2006-01-02 15:04 MST"),
		serverAddr:  fmt.Sprintf("http://localhost:%s", config.Port),
		healthStatus: fiber.Map{
			"status":           "ok",
			"service":          config.Profile.Name,
			"version":          config.Version,
			"registered_pages": len(staticPages),
		},
	}, nil
}

func (app *SiteApp) Register(router fiber.Router) error {
	router.Get("/", app.handleHome)

	for _, page := range app.staticPages {
		if err := app.registerStaticPage(router, page); err != nil {
			return err
		}
	}

	router.Get("/healthz", app.handleHealthz)
	router.Get("/robots.txt", app.handleRobots)
	return nil
}

func (app *SiteApp) handleHome(c fiber.Ctx) error {
	data := app.baseData("/")
	data.MetaTitle = app.profile.Name + " | 已注册页面入口"
	data.MetaDescription = app.profile.Description
	data.Stats = []StatCard{
		{Value: fmt.Sprintf("%d", data.AppPageCount), Label: "已注册页面", Note: "自动识别 public 目录下的 index 页面并生成首页入口。"},
		{Value: "Auto Route", Label: "接入方式", Note: "新增 public/<route>/index.html 后，服务启动时自动挂载对应路由。"},
		{Value: "Blue / White", Label: "展示风格", Note: "统一采用专业蓝白配色、稳定卡片栅格与清晰层级。"},
	}
	data.WorkflowSteps = []string{
		"在任意业务目录下生成独立页面后，将入口落到 public/<route>/index.html。",
		"服务启动时自动识别并注册路由，首页卡片入口同步更新。",
		"页面内部资源继续跟随各自目录分发，不需要额外手写路由。",
	}
	data.DesignTokens = []string{
		"网站只保留单首页作为入口层，避免额外信息架构分散注意力。",
		"整体采用专业蓝白配色、清晰层级和稳定卡片布局。",
		"每个已注册页面都保留统一返回主页入口，方便来回切换。",
	}
	return renderTemplate(c, app.templates, "home", data)
}

func (app *SiteApp) handleHealthz(c fiber.Ctx) error {
	status := fiber.Map{}
	for key, value := range app.healthStatus {
		status[key] = value
	}
	status["generated_at"] = app.generatedAt
	return c.JSON(status)
}

func (app *SiteApp) handleRobots(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	return c.SendString("User-agent: *\nAllow: /\n")
}

func (app *SiteApp) registerStaticPage(router fiber.Router, page staticPage) error {
	subFS, err := fs.Sub(app.publicFS, page.Dir)
	if err != nil {
		return fmt.Errorf("static subfs %s: %w", page.Dir, err)
	}

	handler := func(c fiber.Ctx) error {
		return sendStaticIndex(c, subFS, page.Path+"/")
	}

	router.Get(page.Path, handler)
	router.Get(page.Path+"/", handler)
	router.Use(page.Path+"/*", fiberstatic.New("", fiberstatic.Config{
		FS:            subFS,
		Compress:      true,
		ByteRange:     true,
		CacheDuration: 24 * time.Hour,
		MaxAge:        86400,
	}))
	return nil
}

func (app *SiteApp) baseData(currentPath string) PageData {
	appPages := make([]PageEntry, 0, len(app.staticPages))
	for _, page := range app.staticPages {
		appPages = append(appPages, page.PageEntry)
	}

	return PageData{
		CurrentPath:  currentPath,
		Site:         app.profile,
		GeneratedAt:  app.generatedAt,
		ServerAddr:   app.serverAddr,
		AppPages:     appPages,
		AppPageCount: len(appPages),
	}
}

func discoverStaticPages(publicFS fs.FS) ([]staticPage, error) {
	pages := make([]staticPage, 0, 4)
	reserved := map[string]struct{}{
		"/":           {},
		"/healthz":    {},
		"/robots.txt": {},
	}

	err := fs.WalkDir(publicFS, ".", func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || path.Base(filePath) != "index.html" || filePath == "index.html" {
			return nil
		}

		dir := path.Dir(filePath)
		routePath := "/" + strings.TrimPrefix(dir, ".")
		routePath = strings.TrimSuffix(routePath, "/")
		if _, ok := reserved[routePath]; ok {
			return nil
		}

		meta := staticPageMeta(dir)
		pages = append(pages, staticPage{
			PageEntry: PageEntry{
				Title:       meta.title,
				Description: meta.description,
				Path:        routePath,
				Kind:        "本地页面",
				Source:      path.Join("public", filePath),
				Featured:    meta.featured,
			},
			Dir: dir,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Path < pages[j].Path
	})
	return pages, nil
}

func renderTemplate(c fiber.Ctx, templates *template.Template, name string, data PageData) error {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("level=error template=%s path=%s err=%v", name, c.Path(), err)
		return fiber.NewError(http.StatusInternalServerError, "template render failed")
	}

	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return c.SendString(buf.String())
}

func sendStaticIndex(c fiber.Ctx, contentFS fs.FS, baseHref string) error {
	body, err := fs.ReadFile(contentFS, "index.html")
	if err != nil {
		return fiber.NewError(http.StatusNotFound, "static page not found")
	}
	html := string(body)
	if baseHref != "" && !strings.Contains(html, "<base ") {
		html = strings.Replace(html, "<head>", "<head>\n  <base href=\""+baseHref+"\" />", 1)
	}
	html = injectStaticHomeBadge(html)
	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return c.SendString(html)
}

func injectStaticHomeBadge(html string) string {
	if strings.Contains(html, "data-site-home-badge") {
		return html
	}

	style := `<style>
  .site-home-badge{position:fixed;top:16px;left:16px;z-index:9999;display:inline-flex;align-items:center;gap:10px;padding:10px 14px;border-radius:999px;border:1px solid rgba(21,94,239,.14);background:rgba(255,255,255,.95);box-shadow:0 12px 28px rgba(15,23,42,.12);backdrop-filter:blur(10px);color:#0f4dd8;text-decoration:none;font:700 13px/1 "Avenir Next","PingFang SC","Microsoft YaHei",sans-serif}
  .site-home-badge-mark{width:12px;height:12px;border-radius:999px;background:linear-gradient(135deg,#155eef,#60a5fa);box-shadow:0 0 0 4px rgba(21,94,239,.12)}
  @media (max-width:760px){.site-home-badge{top:auto;bottom:104px;left:12px;padding:10px 12px}}
</style>`
	html = strings.Replace(html, "</head>", style+"\n</head>", 1)

	bodyMarker := strings.Index(strings.ToLower(html), "<body")
	if bodyMarker == -1 {
		return html
	}
	bodyOpenEnd := strings.Index(html[bodyMarker:], ">")
	if bodyOpenEnd == -1 {
		return html
	}
	bodyOpenEnd += bodyMarker
	badge := `
			<a class="site-home-badge" data-site-home-badge href="/" aria-label="返回网站主页">
				<span style="display:inline-flex;align-items:center;gap:4px">
					<svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg" style="display:inline-block;vertical-align:middle"><path d="M8.75 3.5L5.25 7L8.75 10.5" stroke="#155eef" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
					返回主页
				</span>
			</a>`
	return html[:bodyOpenEnd+1] + badge + html[bodyOpenEnd+1:]
}

type staticMeta struct {
	title       string
	description string
	featured    bool
}

func staticPageMeta(dir string) staticMeta {
	known := map[string]staticMeta{
		"benchmarks": {
			title:       "Benchmarks",
			description: "本地性能评测看板，直接渲染 public/benchmarks/index.html。",
			featured:    true,
		},
	}

	if meta, ok := known[dir]; ok {
		return meta
	}

	name := path.Base(dir)
	return staticMeta{
		title:       humanizeSegment(name),
		description: fmt.Sprintf("本地静态页面入口，来源于 public/%s/index.html。", dir),
		featured:    false,
	}
}

func humanizeSegment(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	for index, part := range parts {
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	if len(parts) == 0 {
		return value
	}
	return strings.Join(parts, " ")
}
