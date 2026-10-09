package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/quiet-terminal-interactive/qtidocs/internal/site"
)

func main() {
	src := flag.String("src", "", "path to the site's qtidocs/ source folder")
	out := flag.String("out", "", "output directory for the rendered site")
	title := flag.String("title", "", "site title shown in the nav header as \"<title> | QTIDocs\" (optional; normally comes from the registry's sites/<subdomain>.yaml \"title\" field)")
	flag.Parse()

	if *src == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "render: -src and -out are required")
		flag.Usage()
		os.Exit(2)
	}

	if err := site.Build(*src, *out, *title); err != nil {
		fmt.Fprintf(os.Stderr, "render: %v\n", err)
		os.Exit(1)
	}
}
