package markdown

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

const (
	lightChromaStyle  = "github"
	darkChromaStyle   = "github-dark"
	chromaClassPrefix = "chroma-"
)

const wrapperClass = chromaClassPrefix + "chroma"

func chromaFormatterOptions() []chromahtml.Option {
	return []chromahtml.Option{
		chromahtml.WithClasses(true),
		chromahtml.ClassPrefix(chromaClassPrefix),
	}
}

func ChromaCSS() ([]byte, error) {
	f := chromahtml.New(chromaFormatterOptions()...)

	light, err := chromaStyleCSS(f, lightChromaStyle)
	if err != nil {
		return nil, err
	}
	dark, err := chromaStyleCSS(f, darkChromaStyle)
	if err != nil {
		return nil, err
	}
	if !wrapperColorDecl.Match(dark) {
		return nil, fmt.Errorf("markdown: chroma style %q: expected a %q color declaration to fall back to; chroma's output format may have changed", darkChromaStyle, "/* PreWrapper */")
	}
	dark = fillMissingTokenColors(light, dark)

	chromaClass := []byte("." + chromaClassPrefix)
	if !bytes.Contains(dark, chromaClass) {
		return nil, fmt.Errorf("markdown: chroma style %q: no %q rules found to scope for dark mode; chroma's class prefix may have changed", darkChromaStyle, chromaClass)
	}

	var buf bytes.Buffer
	buf.WriteString("/* Fenced-code syntax highlighting (goldmark-highlighting/chroma). */\n")
	buf.Write(light)
	buf.WriteString("\n@media (prefers-color-scheme: dark) {\n")
	buf.Write(scopeChromaCSS(dark, `:root:not([data-theme="light"])`))
	buf.WriteString("}\n\n")
	buf.Write(scopeChromaCSS(dark, `:root[data-theme="dark"]`))
	return buf.Bytes(), nil
}

func chromaStyleCSS(f *chromahtml.Formatter, styleName string) ([]byte, error) {
	style := styles.Get(styleName)
	if style == nil {
		return nil, fmt.Errorf("markdown: unknown chroma style %q", styleName)
	}
	var buf bytes.Buffer
	if err := f.WriteCSS(&buf, style); err != nil {
		return nil, fmt.Errorf("markdown: writing CSS for chroma style %q: %w", styleName, err)
	}
	raw := buf.Bytes()
	if !bytes.Contains(raw, []byte("/* Background */")) || !bytes.Contains(raw, []byte("/* PreWrapper */")) {
		return nil, fmt.Errorf("markdown: chroma style %q: expected %q and %q markers not found in generated CSS; chroma's output format may have changed", styleName, "/* Background */", "/* PreWrapper */")
	}
	return fixWrapperRule(raw), nil
}

var backgroundColorDecl = regexp.MustCompile(`background-color:\s*[^;]+;\s*`)

func fixWrapperRule(css []byte) []byte {
	lines := bytes.Split(css, []byte("\n"))
	kept := make([][]byte, 0, len(lines))
	for _, line := range lines {
		switch {
		case bytes.Contains(line, []byte("/* Background */")):
			continue
		case bytes.Contains(line, []byte("/* PreWrapper */")):
			kept = append(kept, backgroundColorDecl.ReplaceAll(line, nil))
		default:
			kept = append(kept, line)
		}
	}
	return bytes.Join(kept, []byte("\n"))
}

var tokenClassRule = regexp.MustCompile(`\.` + regexp.QuoteMeta(wrapperClass) + ` \.(` + regexp.QuoteMeta(chromaClassPrefix) + `[a-zA-Z0-9]+)\s*\{`)

var wrapperColorDecl = regexp.MustCompile(`/\* PreWrapper \*/.*?(color:\s*[^;]+;)`)

func tokenClasses(css []byte) map[string]bool {
	out := map[string]bool{}
	for _, m := range tokenClassRule.FindAllSubmatch(css, -1) {
		out[string(m[1])] = true
	}
	return out
}

func fillMissingTokenColors(light, dark []byte) []byte {
	m := wrapperColorDecl.FindSubmatch(dark)
	if m == nil {
		return dark
	}
	colorDecl := string(m[1])

	lightClasses, darkClasses := tokenClasses(light), tokenClasses(dark)
	var missing []string
	for class := range lightClasses {
		if !darkClasses[class] {
			missing = append(missing, class)
		}
	}
	if len(missing) == 0 {
		return dark
	}
	sort.Strings(missing)

	var buf bytes.Buffer
	buf.Write(dark)
	buf.WriteString("\n/* Fallback: token types github-dark leaves to inherit dark's base color, */\n/* but which github (light) defines its own color for, see fillMissingTokenColors. */\n")
	for _, class := range missing {
		fmt.Fprintf(&buf, ".%s .%s { %s }\n", wrapperClass, class, colorDecl)
	}
	return buf.Bytes()
}

func scopeChromaCSS(css []byte, prefix string) []byte {
	needle := []byte("." + chromaClassPrefix)
	replacement := []byte(prefix + " " + string(needle))

	lines := bytes.Split(css, []byte("\n"))
	for i, line := range lines {
		lines[i] = bytes.Replace(line, needle, replacement, 1)
	}
	return bytes.Join(lines, []byte("\n"))
}
