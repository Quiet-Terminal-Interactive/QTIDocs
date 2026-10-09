package sanitize

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

var htmlPolicy = newHTMLPolicy()

func newHTMLPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	p.RequireParseableURLs(true)
	p.AllowRelativeURLs(true)
	p.AllowURLSchemes("mailto", "http", "https")
	p.RequireNoFollowOnFullyQualifiedLinks(true)

	p.AllowElements(
		"p", "br", "hr",
		"b", "strong", "i", "em", "u", "s", "del", "ins",
		"sub", "sup", "mark", "small", "kbd",
		"details", "summary",
		"div", "span",
		"ul", "ol", "li", "dl", "dt", "dd",
		"table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"pre", "code",
	)

	p.AllowAttrs("class", "id").Globally()
	p.AllowAttrs("colspan", "rowspan", "align").OnElements("td", "th")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("src", "alt", "width", "height", "title").OnElements("img")
	p.AllowAttrs("start").Matching(regexp.MustCompile(`^[0-9]+$`)).OnElements("ol")
	p.AllowAttrs("type").Matching(regexp.MustCompile(`^(1|a|A|i|I)$`)).OnElements("ol")

	p.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	p.AllowAttrs("checked", "disabled").OnElements("input")

	p.AllowAttrs("tabindex").Matching(regexp.MustCompile(`^0$`)).OnElements("pre")
	p.AllowStyles("color", "background-color", "font-weight", "font-style", "text-decoration", "display").
		OnElements("pre", "code", "span")

	return p
}

func HTML(src []byte) []byte {
	return htmlPolicy.SanitizeBytes(src)
}

func CSS(src []byte) error {
	p := css.NewParser(parse.NewInput(bytes.NewReader(src)), false)
	for {
		gt, tt, data := p.Next()
		if gt == css.ErrorGrammar {
			if err := p.Err(); err != nil && err != io.EOF {
				return fmt.Errorf("sanitize: invalid CSS: %w", err)
			}
			return nil
		}

		if tt == css.AtKeywordToken && strings.EqualFold(string(data), "@import") {
			return fmt.Errorf("sanitize: @import is not allowed in qtidocs.css")
		}

		for _, tok := range p.Values() {
			if tok.TokenType == css.URLToken || tok.TokenType == css.BadURLToken {
				if err := checkCSSURL(tok.Data); err != nil {
					return err
				}
			}
		}
	}
}

func checkCSSURL(raw []byte) error {
	s := string(raw)
	open := strings.IndexByte(s, '(')
	closeParen := strings.LastIndexByte(s, ')')
	if open < 0 || closeParen < open {
		return fmt.Errorf("sanitize: malformed url() token %q in qtidocs.css", s)
	}

	inner := strings.TrimSpace(s[open+1 : closeParen])
	inner = strings.Trim(inner, `"'`)
	if inner == "" || strings.HasPrefix(inner, "data:") {
		return nil
	}
	if strings.HasPrefix(inner, "//") {
		return fmt.Errorf("sanitize: qtidocs.css url() must not reference an external origin: %q", inner)
	}

	u, err := url.Parse(inner)
	if err != nil {
		return fmt.Errorf("sanitize: qtidocs.css has an invalid url() reference %q: %w", inner, err)
	}
	if u.IsAbs() {
		return fmt.Errorf("sanitize: qtidocs.css url() must not reference an external origin: %q", inner)
	}
	return nil
}

var safeAssetContentTypes = map[string]string{
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".avif":  "image/avif",
	".ico":   "image/x-icon",
	".svg":   "image/svg+xml",
	".pdf":   "application/pdf",
	".css":   "text/css; charset=utf-8",
	".txt":   "text/plain; charset=utf-8",
	".json":  "application/json",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".otf":   "font/otf",
}

const unsafeAssetContentType = "application/octet-stream"

func AssetContentType(name string) string {
	if ct, ok := safeAssetContentTypes[strings.ToLower(filepath.Ext(name))]; ok {
		return ct
	}
	return unsafeAssetContentType
}

func ThemeAssetContentType(name string) string {
	if strings.EqualFold(filepath.Ext(name), ".js") {
		return "text/javascript; charset=utf-8"
	}
	return AssetContentType(name)
}
