package markdown

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	highlighting "github.com/yuin/goldmark-highlighting/v2"

	"github.com/quiet-terminal-interactive/qtidocs/internal/sanitize"
)

type Renderer struct {
	md goldmark.Markdown
}

func New() *Renderer {
	return &Renderer{
		md: goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				highlighting.NewHighlighting(
					highlighting.WithStyle(lightChromaStyle),
					highlighting.WithFormatOptions(chromaFormatterOptions()...),
				),
			),
			goldmark.WithParserOptions(
				parser.WithASTTransformers(
					util.Prioritized(assetLinkTransformer{}, 100),
				),
			),
			goldmark.WithRendererOptions(
				goldmarkhtml.WithUnsafe(),
			),
		),
	}
}

func (r *Renderer) Render(src []byte) (template.HTML, error) {
	var buf bytes.Buffer
	if err := r.md.Convert(src, &buf); err != nil {
		return "", err
	}
	return template.HTML(sanitize.HTML(buf.Bytes())), nil
}

type assetLinkTransformer struct{}

var assetsPrefix = []byte("assets/")

func (assetLinkTransformer) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	if err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Image:
			n.Destination = resolveAssetDestination(n.Destination)
		case *ast.Link:
			n.Destination = resolveAssetDestination(n.Destination)
		}
		return ast.WalkContinue, nil
	}); err != nil {
		panic(fmt.Sprintf("markdown: walking AST for asset links: %v", err))
	}
}

func resolveAssetDestination(dest []byte) []byte {
	if !bytes.HasPrefix(dest, assetsPrefix) {
		return dest
	}
	resolved := make([]byte, 0, len(dest)+1)
	resolved = append(resolved, '/')
	resolved = append(resolved, dest...)
	return resolved
}
