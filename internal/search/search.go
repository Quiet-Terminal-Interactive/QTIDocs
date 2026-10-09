package search

import (
	"encoding/json"
	"strings"

	"golang.org/x/net/html"
)

type Doc struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content"`
}

func BuildIndex(docs []Doc) ([]byte, error) {
	if docs == nil {
		docs = []Doc{}
	}
	return json.Marshal(docs)
}

func PlainText(renderedHTML string) string {
	var b strings.Builder
	z := html.NewTokenizer(strings.NewReader(renderedHTML))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return strings.Join(strings.Fields(b.String()), " ")
		case html.TextToken:
			b.Write(z.Text())
			b.WriteByte(' ')
		}
	}
}
