package theme

import (
	"embed"
	"fmt"
	"html/template"
	"io"

	"github.com/quiet-terminal-interactive/qtidocs/internal/markdown"
	"github.com/quiet-terminal-interactive/qtidocs/internal/nav"
)

//go:embed templates/page.html.tmpl templates/redirect.html.tmpl templates/stats.html.tmpl
var templateFS embed.FS

//go:embed static/default.css
var defaultCSS []byte

var DefaultCSS = mustBuildDefaultCSS()

func mustBuildDefaultCSS() []byte {
	chromaCSS, err := markdown.ChromaCSS()
	if err != nil {
		panic(fmt.Sprintf("theme: building default CSS: %v", err))
	}
	return append(append([]byte{}, defaultCSS...), chromaCSS...)
}

//go:embed static/search.js
var SearchJS []byte

//go:embed static/theme.js
var ThemeJS []byte

//go:embed static/analytics.js
var AnalyticsJS []byte

//go:embed static/stats.js
var StatsJS []byte

var pageTemplate = template.Must(template.ParseFS(templateFS, "templates/page.html.tmpl"))
var redirectTemplate = template.Must(template.ParseFS(templateFS, "templates/redirect.html.tmpl"))
var statsTemplate = template.Must(template.ParseFS(templateFS, "templates/stats.html.tmpl"))

const brandName = "QTIDocs"

type VersionLink struct {
	Name string
	Path string
}

type PageData struct {
	SiteTitle      string
	Title          string
	Description    string
	Content        template.HTML
	Nav            []*nav.Entry
	HasOverrideCSS bool
	BasePath       string
	Versions       []VersionLink
	CurrentVersion string
}

func RenderPage(w io.Writer, data PageData) error {
	if data.SiteTitle == "" {
		data.SiteTitle = brandName
	} else {
		data.SiteTitle = data.SiteTitle + " | " + brandName
	}
	return pageTemplate.Execute(w, data)
}

type redirectData struct {
	SiteTitle   string
	DefaultPath string
}

func RenderVersionRedirect(w io.Writer, siteTitle, defaultPath string) error {
	if siteTitle == "" {
		siteTitle = brandName
	} else {
		siteTitle = siteTitle + " | " + brandName
	}
	return redirectTemplate.Execute(w, redirectData{SiteTitle: siteTitle, DefaultPath: defaultPath})
}

type StatsPageData struct {
	SiteTitle string
	BasePath  string
}

// RenderStats writes the public page-view stats page to w.
func RenderStats(w io.Writer, data StatsPageData) error {
	if data.SiteTitle == "" {
		data.SiteTitle = brandName
	}
	return statsTemplate.Execute(w, data)
}
