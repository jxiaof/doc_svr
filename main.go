package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"doc/internal/site"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	fiberstatic "github.com/gofiber/fiber/v3/middleware/static"
)

var version = "dev"

//go:embed web/templates web/assets public
var embeddedFiles embed.FS

func main() {
	templates, err := template.New("site").Funcs(template.FuncMap{
		"formatDate": func(value time.Time) string {
			if value.IsZero() {
				return "待补充"
			}
			return value.Format("2006.01.02")
		},
		"join": strings.Join,
		"routeTail": func(value string) string {
			trimmed := strings.Trim(value, "/")
			if trimmed == "" {
				return "home"
			}
			return path.Base(trimmed)
		},
		"shelfPadding": func(total int) []int {
			padding := 12 - total
			if padding < 0 {
				padding = 0
			}
			return make([]int, padding)
		},
	}).ParseFS(embeddedFiles, "web/templates/*.html")
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}

	assetsFS, err := fs.Sub(embeddedFiles, "web/assets")
	if err != nil {
		log.Fatalf("init assets fs: %v", err)
	}

	publicFS, err := fs.Sub(embeddedFiles, "public")
	if err != nil {
		log.Fatalf("init public fs: %v", err)
	}

	profile := site.Profile{
		Name:        "Docs",
		Tagline:     "本地文档入口服务",
		Description: "统一承载多个已注册本地页面入口，并自动分发各页面的本地静态资源。",
		Owner:       "tenyunw",
		Email:       "hujianghong@tenyunw.com",
		Location:    "Shenzhen / xili",
		Year:        time.Now().Year(),
		Version:     version,
	}

	port := getEnv("PORT", "3000")
	app := fiber.New(fiber.Config{
		AppName:       profile.Name,
		CaseSensitive: false,
		ReadTimeout:   10 * time.Second,
		WriteTimeout:  15 * time.Second,
		IdleTimeout:   60 * time.Second,
		ServerHeader:  "doc-svr",
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := http.StatusInternalServerError
			if fiberErr, ok := err.(*fiber.Error); ok {
				code = fiberErr.Code
			}

			log.Printf("level=error request_id=%s status=%d path=%s err=%v", c.Get(fiber.HeaderXRequestID), code, c.Path(), err)
			return c.Status(code).SendString(http.StatusText(code))
		},
	})

	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} ${latency} ${method} ${path} ip=${ip}\n",
		TimeFormat: time.RFC3339,
	}))
	app.Use(etag.New())
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestCompression,
	}))
	app.Use("/assets", fiberstatic.New("", fiberstatic.Config{
		FS:            assetsFS,
		Compress:      true,
		ByteRange:     true,
		CacheDuration: 24 * time.Hour,
		MaxAge:        86400,
	}))

	siteApp, err := site.NewSiteApp(site.Config{
		Templates: templates,
		PublicFS:  publicFS,
		Profile:   profile,
		Port:      port,
		Version:   version,
	})
	if err != nil {
		log.Fatalf("init site app: %v", err)
	}

	if err := siteApp.Register(app); err != nil {
		log.Fatalf("register site routes: %v", err)
	}

	log.Printf("level=info service=%s version=%s addr=:%s", profile.Name, version, port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func getEnv(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
