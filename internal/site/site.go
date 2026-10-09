package site

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/quiet-terminal-interactive/qtidocs/internal/frontmatter"
	"github.com/quiet-terminal-interactive/qtidocs/internal/markdown"
	"github.com/quiet-terminal-interactive/qtidocs/internal/nav"
	"github.com/quiet-terminal-interactive/qtidocs/internal/sanitize"
	"github.com/quiet-terminal-interactive/qtidocs/internal/search"
	"github.com/quiet-terminal-interactive/qtidocs/internal/theme"
)

const (
	assetsDirName     = "assets"
	overrideCSSName   = "qtidocs.css"
	navConfigName     = "nav.yaml"
	sidebarConfigName = "sidebar.yaml"
	themeOutputDir    = "_qtidocs"
	themeOutputFile   = "theme.css"
	searchJSFile      = "search.js"
	themeJSFile       = "theme.js"
	analyticsJSFile   = "analytics.js"
	statsJSFile       = "stats.js"
	searchIndexFile   = "search-index.json"
)

var renderer = markdown.New()

type VersionInfo struct {
	Current string
	Names   []string
}

func (v VersionInfo) basePath() string {
	if v.Current == "" {
		return ""
	}
	return "/" + v.Current
}

func (v VersionInfo) links(pageURLPath string) []theme.VersionLink {
	if len(v.Names) == 0 {
		return nil
	}
	out := make([]theme.VersionLink, len(v.Names))
	for i, name := range v.Names {
		p := "/" + name
		if pageURLPath != "" {
			p += "/" + pageURLPath
		}
		out[i] = theme.VersionLink{Name: name, Path: p}
	}
	return out
}

type page struct {
	urlPath     string
	title       string
	description string
	order       int
	navVisible  bool
	html        template.HTML
	plainText   string
}

func Build(srcDir, outDir, siteTitle string) error {
	return BuildVersioned(srcDir, outDir, siteTitle, VersionInfo{})
}

func BuildVersioned(srcDir, outDir, siteTitle string, versionInfo VersionInfo) error {
	pages, err := loadPages(srcDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("site: creating output dir: %w", err)
	}
	if err := writeThemeAssets(outDir); err != nil {
		return err
	}

	hasOverrideCSS, err := copyAssets(srcDir, outDir)
	if err != nil {
		return err
	}

	if err := writeSearchIndex(outDir, pages); err != nil {
		return err
	}

	if len(pages) == 0 {
		return writePlaceholder(outDir, siteTitle, hasOverrideCSS, versionInfo)
	}

	navEntries, err := buildNav(srcDir, pages)
	if err != nil {
		return err
	}
	if basePath := versionInfo.basePath(); basePath != "" {
		prefixNavPaths(navEntries, basePath)
	}

	for _, p := range pages {
		if err := writePage(outDir, p, siteTitle, navEntries, hasOverrideCSS, versionInfo); err != nil {
			return err
		}
	}
	return nil
}

func prefixNavPaths(entries []*nav.Entry, basePath string) {
	prefix := strings.TrimPrefix(basePath, "/") + "/"
	for _, e := range entries {
		if e.Path != "" {
			e.Path = prefix + e.Path
		}
		prefixNavPaths(e.Children, basePath)
	}
}

func writeSearchIndex(outDir string, pages []page) error {
	docs := make([]search.Doc, len(pages))
	for i, p := range pages {
		docs[i] = search.Doc{
			Path:        p.urlPath,
			Title:       p.title,
			Description: p.description,
			Content:     p.plainText,
		}
	}
	data, err := search.BuildIndex(docs)
	if err != nil {
		return fmt.Errorf("site: building search index: %w", err)
	}
	return os.WriteFile(filepath.Join(outDir, themeOutputDir, searchIndexFile), data, 0o644)
}

func loadPages(srcDir string) ([]page, error) {
	var pages []page
	err := filepath.WalkDir(srcDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel == assetsDirName {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".md") {
			return nil
		}

		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		data, body, err := frontmatter.Parse(src)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		html, err := renderer.Render(body)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}

		pages = append(pages, page{
			urlPath:     urlPathFor(rel),
			title:       data.Title,
			description: data.Description,
			order:       data.Order,
			navVisible:  data.Nav,
			html:        html,
			plainText:   search.PlainText(string(html)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pages, nil
}

func urlPathFor(rel string) string {
	trimmed := strings.TrimSuffix(rel, ".md")
	dir, base := path.Split(trimmed)
	if base == "index" {
		return strings.TrimSuffix(dir, "/")
	}
	return trimmed
}

func buildNav(srcDir string, pages []page) ([]*nav.Entry, error) {
	navPages := make([]nav.Page, len(pages))
	for i, p := range pages {
		navPages[i] = nav.Page{URLPath: p.urlPath, Title: p.title, Order: p.order, Nav: p.navVisible}
	}

	for _, name := range []string{navConfigName, sidebarConfigName} {
		src, err := os.ReadFile(filepath.Join(srcDir, name))
		switch {
		case err == nil:
			cfg, err := nav.ParseConfig(src)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			return nav.FromConfig(cfg, navPages), nil
		case errors.Is(err, fs.ErrNotExist):
			continue
		default:
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}

	return nav.Build(navPages), nil
}

func writePage(outDir string, p page, siteTitle string, navEntries []*nav.Entry, hasOverrideCSS bool, versionInfo VersionInfo) (err error) {
	dir := filepath.Join(outDir, filepath.FromSlash(p.urlPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return err
	}
	defer closeAndReportErr(f, &err)

	return theme.RenderPage(f, theme.PageData{
		SiteTitle:      siteTitle,
		Title:          p.title,
		Description:    p.description,
		Content:        p.html,
		Nav:            navEntries,
		HasOverrideCSS: hasOverrideCSS,
		BasePath:       versionInfo.basePath(),
		Versions:       versionInfo.links(p.urlPath),
		CurrentVersion: versionInfo.Current,
	})
}

func writePlaceholder(outDir, siteTitle string, hasOverrideCSS bool, versionInfo VersionInfo) (err error) {
	f, err := os.Create(filepath.Join(outDir, "index.html"))
	if err != nil {
		return err
	}
	defer closeAndReportErr(f, &err)

	return theme.RenderPage(f, theme.PageData{
		SiteTitle:      siteTitle,
		Title:          "No pages yet",
		Content:        template.HTML("<p>This site doesn't have any Markdown pages in its <code>qtidocs/</code> folder yet.</p>"),
		HasOverrideCSS: hasOverrideCSS,
		BasePath:       versionInfo.basePath(),
		Versions:       versionInfo.links(""),
		CurrentVersion: versionInfo.Current,
	})
}

func writeThemeAssets(outDir string) error {
	dir := filepath.Join(outDir, themeOutputDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, themeOutputFile), theme.DefaultCSS, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, searchJSFile), theme.SearchJS, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, themeJSFile), theme.ThemeJS, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, analyticsJSFile), theme.AnalyticsJS, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, statsJSFile), theme.StatsJS, 0o644)
}

func copyAssets(srcDir, outDir string) (hasOverrideCSS bool, err error) {
	assetsSrc := filepath.Join(srcDir, assetsDirName)
	info, err := os.Stat(assetsSrc)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("site: stat assets: %w", err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("site: %s is not a directory", assetsSrc)
	}

	overrideCSS, err := os.ReadFile(filepath.Join(assetsSrc, overrideCSSName))
	switch {
	case err == nil:
		if err := sanitize.CSS(overrideCSS); err != nil {
			return false, fmt.Errorf("site: %s: %w", overrideCSSName, err)
		}
		hasOverrideCSS = true
	case errors.Is(err, fs.ErrNotExist):
		// No override
	default:
		return false, fmt.Errorf("site: reading %s: %w", overrideCSSName, err)
	}

	if err := copyDir(assetsSrc, filepath.Join(outDir, assetsDirName)); err != nil {
		return false, err
	}

	return hasOverrideCSS, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer closeAndReportErr(out, &err)

	_, err = io.Copy(out, in)
	return err
}

func closeAndReportErr(f *os.File, err *error) {
	if cerr := f.Close(); cerr != nil && *err == nil {
		*err = fmt.Errorf("site: closing %s: %w", f.Name(), cerr)
	}
}
