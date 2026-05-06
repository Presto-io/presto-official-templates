package main

import (
	_ "embed"
	"strings"

	"github.com/Presto-io/presto-official-templates/internal/cli"
	"github.com/Presto-io/presto-official-templates/internal/typst"
)

//go:embed manifest.json
var manifestJSON string

//go:embed example.md
var exampleMD string

func convert(input string) string {
	content := strings.TrimSpace(input)
	if content == "" {
		content = "gongwen-beta canary"
	}

	return `#set page(width: 210mm, height: 297mm, margin: 24mm)
#set text(size: 12pt)
#heading(level: 1)[gongwen-beta canary]
#par[` + typst.EscapeContent(content) + `]
`
}

func main() {
	cli.Run(manifestJSON, exampleMD, convert)
}
