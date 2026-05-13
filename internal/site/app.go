package site

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
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

type PageEntry struct {
	Title       string
	Description string
	Path        string
	Kind        string
	Source      string
	Featured    bool
}

type BreadcrumbItem struct {
	Label   string
	Path    string
	Current bool
}

type WorkspaceNavItem struct {
	Label        string
	Path         string
	Active       bool
	CompactLabel string
}

type WorkspaceNavGroup struct {
	Label        string
	Active       bool
	Expanded     bool
	Standalone   bool
	CompactLabel string
	Item         WorkspaceNavItem
	Items        []WorkspaceNavItem
}

type PageData struct {
	MetaTitle       string
	MetaDescription string
	AssetVersion    string
	CurrentPath     string
	Site            Profile
	AppPages        []PageEntry
	AppPageCount    int
	CurrentPage     PageEntry
	RawPagePath     string
	Breadcrumbs     []BreadcrumbItem
	WorkspaceNav    []WorkspaceNavGroup
	BodyClass       string
	ShellClass      string
	HideHeader      bool
	HideFooter      bool
}

type Config struct {
	Templates *template.Template
	PublicFS  fs.FS
	Profile   Profile
	Version   string
}

type SiteApp struct {
	templates    *template.Template
	publicFS     fs.FS
	profile      Profile
	staticPages  []staticPage
	assetVersion string
	generatedAt  string
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
		templates:    config.Templates,
		publicFS:     config.PublicFS,
		profile:      config.Profile,
		staticPages:  staticPages,
		assetVersion: fmt.Sprintf("%d", time.Now().Unix()),
		generatedAt:  time.Now().Format("2006-01-02 15:04 MST"),
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
	return renderTemplate(c, app.templates, "home", app.homeData())
}

func (app *SiteApp) WriteStaticHome(outputPath string) error {
	markup, err := executeTemplate(app.templates, "home", app.homeData())
	if err != nil {
		return fmt.Errorf("render home template: %w", err)
	}

	dir := filepath.Dir(outputPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	if err := os.WriteFile(outputPath, []byte(markup), 0o644); err != nil {
		return fmt.Errorf("write static home: %w", err)
	}
	return nil
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

	rawPagePath := rawStaticPagePath(page.Path)

	shellHandler := func(c fiber.Ctx) error {
		data := app.baseData(page.Path)
		data.MetaTitle = page.Title + " | " + app.profile.Name
		data.MetaDescription = page.Description
		data.CurrentPage = page.PageEntry
		data.RawPagePath = rawPagePath + "/"
		data.WorkspaceNav = buildWorkspaceNav(data.AppPages, page.Path)
		data.BodyClass = "workspace-mode"
		data.ShellClass = "page-shell-workspace"
		data.HideHeader = true
		data.HideFooter = true
		data.Breadcrumbs = []BreadcrumbItem{
			{Label: "首页", Path: "/"},
			{Label: "页面", Path: "/#registered-pages"},
			{Label: page.Title, Path: page.Path, Current: true},
		}
		return renderTemplate(c, app.templates, "page_shell", data)
	}

	rawHandler := func(c fiber.Ctx) error {
		setNoCacheHeaders(c)
		return sendStaticIndex(c, subFS, rawPagePath+"/")
	}

	rawAssetNoCache := func(c fiber.Ctx) error {
		setNoCacheHeaders(c)
		return c.Next()
	}

	router.Get(page.Path, shellHandler)
	router.Get(page.Path+"/", shellHandler)
	router.Get(rawPagePath, rawHandler)
	router.Get(rawPagePath+"/", rawHandler)
	router.Use(rawPagePath+"/*", rawAssetNoCache, fiberstatic.New("", fiberstatic.Config{
		FS:            subFS,
		Compress:      true,
		ByteRange:     true,
		CacheDuration: 0,
		MaxAge:        0,
	}))
	return nil
}

func (app *SiteApp) baseData(currentPath string) PageData {
	appPages := make([]PageEntry, 0, len(app.staticPages))
	for _, page := range app.staticPages {
		appPages = append(appPages, page.PageEntry)
	}

	return PageData{
		AssetVersion: app.assetVersion,
		CurrentPath:  currentPath,
		Site:         app.profile,
		AppPages:     appPages,
		AppPageCount: len(appPages),
	}
}

func setNoCacheHeaders(c fiber.Ctx) {
	c.Set(fiber.HeaderCacheControl, "no-store, no-cache, must-revalidate, max-age=0")
	c.Set(fiber.HeaderPragma, "no-cache")
	c.Set(fiber.HeaderExpires, "0")
}

func (app *SiteApp) homeData() PageData {
	data := app.baseData("/")
	data.MetaTitle = app.profile.Name + " | 已注册页面入口"
	data.MetaDescription = app.profile.Description
	return data
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
	markup, err := executeTemplate(templates, name, data)
	if err != nil {
		log.Printf("level=error template=%s path=%s err=%v", name, c.Path(), err)
		return fiber.NewError(http.StatusInternalServerError, "template render failed")
	}

	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return c.SendString(markup)
}

func executeTemplate(templates *template.Template, name string, data PageData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
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
	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return c.SendString(html)
}

func rawStaticPagePath(pagePath string) string {
	trimmed := strings.Trim(pagePath, "/")
	if trimmed == "" {
		return "/_pages/home"
	}
	return "/_pages/" + trimmed
}

func buildWorkspaceNav(appPages []PageEntry, currentPath string) []WorkspaceNavGroup {
	type navGroupBuilder struct {
		segment  string
		label    string
		root     *WorkspaceNavItem
		children []WorkspaceNavItem
		active   bool
	}

	order := make([]string, 0, len(appPages))
	builders := make(map[string]*navGroupBuilder, len(appPages))

	for _, page := range appPages {
		trimmed := strings.Trim(page.Path, "/")
		if trimmed == "" {
			continue
		}

		segments := strings.Split(trimmed, "/")
		segment := segments[0]
		builder, ok := builders[segment]
		if !ok {
			builder = &navGroupBuilder{segment: segment, label: humanizeSegment(segment)}
			builders[segment] = builder
			order = append(order, segment)
		}

		item := WorkspaceNavItem{
			Label:        page.Title,
			Path:         page.Path,
			Active:       page.Path == currentPath,
			CompactLabel: compactNavLabel(page.Path, page.Title),
		}

		if item.Active {
			builder.active = true
		}

		if len(segments) == 1 {
			builder.root = &item
			if strings.TrimSpace(page.Title) != "" {
				builder.label = page.Title
			}
			continue
		}

		builder.children = append(builder.children, item)
	}

	navGroups := make([]WorkspaceNavGroup, 0, len(order))
	for _, segment := range order {
		builder := builders[segment]
		if builder == nil {
			continue
		}

		if len(builder.children) == 0 && builder.root != nil {
			navGroups = append(navGroups, WorkspaceNavGroup{
				Standalone:   true,
				Label:        builder.root.Label,
				Active:       builder.root.Active,
				CompactLabel: builder.root.CompactLabel,
				Item:         *builder.root,
			})
			continue
		}

		items := make([]WorkspaceNavItem, 0, len(builder.children)+1)
		if builder.root != nil {
			rootItem := *builder.root
			if rootItem.Label == builder.label {
				rootItem.Label = "概览"
			}
			items = append(items, rootItem)
		}
		items = append(items, builder.children...)

		navGroups = append(navGroups, WorkspaceNavGroup{
			Label:        builder.label,
			Active:       builder.active,
			Expanded:     builder.active,
			CompactLabel: compactNavLabel(segment, builder.label),
			Items:        items,
		})
	}

	return navGroups
}

func compactNavLabel(value string, fallback string) string {
	source := strings.TrimSpace(value)
	if strings.Contains(source, "/") {
		source = path.Base(strings.Trim(source, "/"))
	}
	if source == "" {
		source = strings.TrimSpace(fallback)
	}
	if source == "" {
		source = "PAGE"
	}

	runes := []rune(strings.ToUpper(source))
	if len(runes) > 3 {
		runes = runes[:3]
	}
	return string(runes)
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
