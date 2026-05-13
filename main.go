package main

import (
	"embed"
	"errors"
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

type runtimeOptions struct {
	command string
	port    string
	output  string
}

//go:embed web/templates web/assets public
var embeddedFiles embed.FS

func main() {
	options, err := parseRuntimeOptions(os.Args[1:], getEnv("PORT", "4000"))
	if err != nil {
		log.Fatalf("parse runtime options: %v", err)
	}

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
		Description: "自动转发本地服务 · 统一入口 · 便捷访问",
		Owner:       "tenyunw",
		Email:       "hujianghong@tenyunw.com",
		Location:    "Shenzhen / xili",
		Year:        time.Now().Year(),
		Version:     version,
	}

	port := options.port
	siteApp, err := site.NewSiteApp(site.Config{
		Templates: templates,
		PublicFS:  publicFS,
		Profile:   profile,
		Version:   version,
	})
	if err != nil {
		log.Fatalf("init site app: %v", err)
	}

	if options.command == "generate-home" {
		if err := siteApp.WriteStaticHome(options.output); err != nil {
			log.Fatalf("generate home: %v", err)
		}
		log.Printf("level=info generated_home=%s", options.output)
		return
	}

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
	app.Get("/favicon.ico", func(c fiber.Ctx) error {
		return c.Redirect().To("/assets/favicon.svg")
	})
	app.Get("/favicon.svg", func(c fiber.Ctx) error {
		return c.Redirect().To("/assets/favicon.svg")
	})
	app.Use("/assets", fiberstatic.New("", fiberstatic.Config{
		FS:            assetsFS,
		Compress:      true,
		ByteRange:     true,
		CacheDuration: 24 * time.Hour,
		MaxAge:        86400,
	}))

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

func parseRuntimeOptions(args []string, defaultPort string) (runtimeOptions, error) {
	options := runtimeOptions{
		command: "serve",
		port:    defaultPort,
		output:  "public/index.html",
	}

	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		options.command = args[0]
		args = args[1:]
	}

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--port":
			index++
			if index >= len(args) {
				return runtimeOptions{}, errors.New("--port requires a value")
			}
			options.port = strings.TrimSpace(args[index])
		case "--output":
			index++
			if index >= len(args) {
				return runtimeOptions{}, errors.New("--output requires a value")
			}
			options.output = strings.TrimSpace(args[index])
		default:
			return runtimeOptions{}, errors.New("unsupported argument: " + args[index])
		}
	}

	if options.command != "serve" && options.command != "generate-home" {
		return runtimeOptions{}, errors.New("unsupported command: " + options.command)
	}
	if options.port == "" {
		options.port = defaultPort
	}
	if options.output == "" {
		options.output = "public/index.html"
	}
	return options, nil
}
