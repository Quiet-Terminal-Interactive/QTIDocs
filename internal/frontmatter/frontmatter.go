package frontmatter

import (
	"errors"
	"fmt"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

var ErrMissingTitle = errors.New("frontmatter: title is required")

type Data struct {
	Title       string
	Order       int
	Nav         bool
	Description string
	Author      string
}

type raw struct {
	Title       string `yaml:"title"`
	Order       int    `yaml:"order"`
	Nav         *bool  `yaml:"nav"`
	Description string `yaml:"description"`
	Author      string `yaml:"author"`
}

func Parse(src []byte) (Data, []byte, error) {
	lines := strings.Split(string(src), "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return Data{}, nil, ErrMissingTitle
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return Data{}, nil, fmt.Errorf("frontmatter: unterminated --- block")
	}

	block := strings.Join(lines[1:end], "\n")
	var r raw
	if err := yaml.Unmarshal([]byte(block), &r); err != nil {
		return Data{}, nil, fmt.Errorf("frontmatter: %w", err)
	}
	if r.Title == "" {
		return Data{}, nil, ErrMissingTitle
	}

	nav := true
	if r.Nav != nil {
		nav = *r.Nav
	}

	body := []byte(strings.Join(lines[end+1:], "\n"))
	return Data{
		Title:       r.Title,
		Order:       r.Order,
		Nav:         nav,
		Description: r.Description,
		Author:      r.Author,
	}, body, nil
}
