package site

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	rendererhtml "github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

type Post struct {
	Slug        string
	Title       string
	Summary     string
	Tags        []string
	PublishedAt time.Time
	ReadingTime string
	HTML        template.HTML
	Featured    bool
	SourcePath  string
}

type frontMatter struct {
	Title    string   `yaml:"title"`
	Summary  string   `yaml:"summary"`
	Date     string   `yaml:"date"`
	Tags     []string `yaml:"tags"`
	Featured bool     `yaml:"featured"`
}

func LoadPosts(contentFS fs.FS, root string) ([]Post, error) {
	parser := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.DefinitionList,
		),
		goldmark.WithRendererOptions(rendererhtml.WithUnsafe()),
	)

	posts := make([]Post, 0, 8)
	err := fs.WalkDir(contentFS, root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || path.Ext(filePath) != ".md" || strings.HasPrefix(path.Base(filePath), "_") {
			return nil
		}

		raw, err := fs.ReadFile(contentFS, filePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", filePath, err)
		}

		meta, body, err := parseFrontMatter(raw)
		if err != nil {
			return fmt.Errorf("parse front matter %s: %w", filePath, err)
		}

		var htmlBuf bytes.Buffer
		if err := parser.Convert(body, &htmlBuf); err != nil {
			return fmt.Errorf("render markdown %s: %w", filePath, err)
		}

		publishedAt, err := parseDate(meta.Date)
		if err != nil {
			return fmt.Errorf("parse date %s: %w", filePath, err)
		}

		summary := strings.TrimSpace(meta.Summary)
		if summary == "" {
			summary = excerpt(string(body), 108)
		}

		posts = append(posts, Post{
			Slug:        strings.TrimSuffix(path.Base(filePath), path.Ext(filePath)),
			Title:       fallbackTitle(meta.Title, filePath),
			Summary:     summary,
			Tags:        meta.Tags,
			PublishedAt: publishedAt,
			ReadingTime: estimateReadingTime(string(body)),
			HTML:        template.HTML(htmlBuf.String()),
			Featured:    meta.Featured,
			SourcePath:  filePath,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].PublishedAt.After(posts[j].PublishedAt)
	})

	return posts, nil
}

func FindPost(posts []Post, slug string) (*Post, bool) {
	for index := range posts {
		if posts[index].Slug == slug {
			return &posts[index], true
		}
	}
	return nil, false
}

func parseFrontMatter(raw []byte) (frontMatter, []byte, error) {
	var meta frontMatter
	if !bytes.HasPrefix(raw, []byte("---\n")) {
		return meta, bytes.TrimSpace(raw), nil
	}

	section := raw[len("---\n"):]
	endIndex := bytes.Index(section, []byte("\n---\n"))
	if endIndex < 0 {
		return meta, nil, fmt.Errorf("missing closing front matter delimiter")
	}

	if err := yaml.Unmarshal(section[:endIndex], &meta); err != nil {
		return meta, nil, err
	}

	body := bytes.TrimSpace(section[endIndex+len("\n---\n"):])
	return meta, body, nil
}

func parseDate(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}

	layouts := []string{time.RFC3339, "2006-01-02", "2006/01/02"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported date format: %s", trimmed)
}

func estimateReadingTime(markdown string) string {
	runeCount := utf8.RuneCountInString(markdown)
	minutes := runeCount / 420
	if runeCount%420 != 0 {
		minutes++
	}
	if minutes < 1 {
		minutes = 1
	}
	return fmt.Sprintf("%d min read", minutes)
}

func excerpt(markdown string, limit int) string {
	replacer := strings.NewReplacer(
		"#", " ",
		"*", " ",
		"`", " ",
		">", " ",
		"[", " ",
		"]", " ",
		"(", " ",
		")", " ",
		"-", " ",
	)
	clean := strings.Join(strings.Fields(replacer.Replace(markdown)), " ")
	if utf8.RuneCountInString(clean) <= limit {
		return clean
	}

	runes := []rune(clean)
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func fallbackTitle(title string, filePath string) string {
	trimmed := strings.TrimSpace(title)
	if trimmed != "" {
		return trimmed
	}

	name := strings.TrimSuffix(path.Base(filePath), path.Ext(filePath))
	return strings.ReplaceAll(name, "-", " ")
}
