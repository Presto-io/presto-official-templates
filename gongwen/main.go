package main

import (
	_ "embed"
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Presto-io/presto-official-templates/internal/cli"
	"github.com/Presto-io/presto-official-templates/internal/typst"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

//go:embed template_head.typ
var templateHead string

//go:embed manifest.json
var manifestJSON string

//go:embed example.md
var exampleMD string

// ---------- YAML front-matter ----------

type frontMatter struct {
	Title     string
	Author    string // joined with "、"
	Date      string // raw string from YAML
	Signature bool
}

func todayDate() string {
	return time.Now().Format("2006-01-02")
}

// parseFrontMatter splits "---" delimited YAML from body and returns metadata + body.
func parseFrontMatter(input string) (frontMatter, string) {
	var fm frontMatter
	fm.Title = "请输入文字"
	fm.Author = "请输入文字"
	fm.Date = todayDate()

	// Normalise line endings
	input = strings.ReplaceAll(input, "\r\n", "\n")

	if !strings.HasPrefix(input, "---") {
		return fm, input
	}

	// Find closing ---
	rest := input[3:]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fm, input
	}
	yamlBlock := rest[:idx]
	body := rest[idx+4:] // skip "\n---"
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	}

	// Parse YAML into a generic map
	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlBlock), &raw); err != nil {
		return fm, body
	}

	// title
	if v, ok := raw["title"]; ok {
		fm.Title = fmt.Sprintf("%v", v)
	}

	// author: string or list of strings → join with "、"
	if v, ok := raw["author"]; ok {
		switch a := v.(type) {
		case string:
			fm.Author = a
		case []interface{}:
			parts := make([]string, 0, len(a))
			for _, item := range a {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
			fm.Author = strings.Join(parts, "、")
		}
	}

	// date. Accept data as a compatibility alias so a typo cannot break preview.
	if v, ok := raw["date"]; ok {
		fm.Date = fmt.Sprintf("%v", v)
	} else if v, ok := raw["data"]; ok {
		fm.Date = fmt.Sprintf("%v", v)
	}

	// signature: bool or string
	if v, ok := raw["signature"]; ok {
		switch s := v.(type) {
		case bool:
			fm.Signature = s
		case string:
			lower := strings.ToLower(s)
			fm.Signature = lower == "true" || lower == "yes"
		}
	}

	return fm, body
}

var dateRe = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`)

// formatDate converts "YYYY-MM-DD" to datetime(year: N, month: N, day: N),
// otherwise returns a quoted string.
func formatDate(date string) string {
	if date == "" {
		return `""`
	}
	m := dateRe.FindStringSubmatch(date)
	if m != nil {
		// Strip leading zeros for month/day
		year := m[1]
		month := strings.TrimLeft(m[2], "0")
		day := strings.TrimLeft(m[3], "0")
		return fmt.Sprintf("datetime(\n  year: %s,\n  month: %s,\n  day: %s,\n)", year, month, day)
	}
	return fmt.Sprintf(`"%s"`, typst.EscapeString(date))
}

// ---------- Punctuation conversion ----------

// urlPattern matches common URL schemes to skip
var urlPattern = regexp.MustCompile(`https?://[^\s]+|ftp://[^\s]+|mailto:[^\s]+`)

// markerPattern matches {…} markers to skip
var markerPattern = regexp.MustCompile(`\{[^}]*\}`)

// convertPunctuation converts half-width punctuation to full-width for Chinese text.
func convertPunctuation(text string) string {
	// Find all regions to skip (URLs and markers)
	type span struct{ start, end int }
	var skipSpans []span

	for _, loc := range urlPattern.FindAllStringIndex(text, -1) {
		skipSpans = append(skipSpans, span{loc[0], loc[1]})
	}
	for _, loc := range markerPattern.FindAllStringIndex(text, -1) {
		skipSpans = append(skipSpans, span{loc[0], loc[1]})
	}
	sort.Slice(skipSpans, func(i, j int) bool {
		if skipSpans[i].start == skipSpans[j].start {
			return skipSpans[i].end < skipSpans[j].end
		}
		return skipSpans[i].start < skipSpans[j].start
	})
	if len(skipSpans) > 1 {
		merged := skipSpans[:0]
		for _, s := range skipSpans {
			lastIdx := len(merged) - 1
			if lastIdx >= 0 && s.start <= merged[lastIdx].end {
				if s.end > merged[lastIdx].end {
					merged[lastIdx].end = s.end
				}
				continue
			}
			merged = append(merged, s)
		}
		skipSpans = merged
	}

	var buf strings.Builder
	buf.Grow(len(text))

	runes := []rune(text)
	spanIdx := 0
	bytePos := 0
	for i, r := range runes {
		for spanIdx < len(skipSpans) && bytePos >= skipSpans[spanIdx].end {
			spanIdx++
		}
		if spanIdx < len(skipSpans) && bytePos >= skipSpans[spanIdx].start && bytePos < skipSpans[spanIdx].end {
			buf.WriteRune(r)
			bytePos += utf8.RuneLen(r)
			continue
		}

		switch r {
		case ',':
			buf.WriteRune('，')
		case ';':
			buf.WriteRune('；')
		case '?':
			buf.WriteRune('？')
		case '(':
			buf.WriteRune('（')
		case ')':
			buf.WriteRune('）')
		case ':':
			// Keep colon between digits (e.g. 12:30)
			if i > 0 && i < len(runes)-1 && unicode.IsDigit(runes[i-1]) && unicode.IsDigit(runes[i+1]) {
				buf.WriteRune(':')
			} else {
				buf.WriteRune('：')
			}
		default:
			buf.WriteRune(r)
		}
		bytePos += utf8.RuneLen(r)
	}
	return buf.String()
}

// ---------- Markdown pre-processing ----------

var reNoindentOpen = regexp.MustCompile(`(?m)^::: \{\.noindent\}\s*$`)
var reNoindentClose = regexp.MustCompile(`(?m)^:::\s*$`)

func preprocessBody(body string) string {
	body = reNoindentOpen.ReplaceAllString(body, "<!-- noindent-start -->")
	body = reNoindentClose.ReplaceAllString(body, "<!-- noindent-end -->")
	return body
}

// ---------- Goldmark AST → Typst converter ----------

type converter struct {
	source        []byte
	figureCounter int
	tableCounter  int
	hasSeenHeader bool
}

// nodeText extracts raw text from an inline node and its children.
func (c *converter) nodeText(n ast.Node) string {
	var buf strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindText {
			t := child.(*ast.Text)
			buf.Write(t.Segment.Value(c.source))
			if t.SoftLineBreak() {
				buf.WriteByte('\n')
			}
		} else {
			buf.WriteString(c.nodeText(child))
		}
	}
	if n.Kind() == ast.KindText {
		t := n.(*ast.Text)
		buf.Write(t.Segment.Value(c.source))
	}
	return buf.String()
}

// plainText extracts all text from a node tree (for marker detection).
func (c *converter) plainText(n ast.Node) string {
	var buf strings.Builder
	_ = ast.Walk(n, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if node.Kind() == ast.KindText {
			t := node.(*ast.Text)
			buf.Write(t.Segment.Value(c.source))
			if t.SoftLineBreak() {
				buf.WriteByte(' ')
			}
		} else if node.Kind() == ast.KindCodeSpan {
			// include code span text
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if child.Kind() == ast.KindText {
					t := child.(*ast.Text)
					buf.Write(t.Segment.Value(c.source))
				}
			}
			return ast.WalkSkipChildren, nil
		} else if node.Kind() == ast.KindString {
			buf.WriteString(html.UnescapeString(string(node.(*ast.String).Value)))
		}
		return ast.WalkContinue, nil
	})
	return buf.String()
}

// renderInlines renders inline children of a node to Typst.
func (c *converter) renderInlines(n ast.Node) string {
	var buf strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		buf.WriteString(c.renderInline(child))
	}
	return buf.String()
}

// renderInline renders a single inline node to Typst.
func (c *converter) renderInline(n ast.Node) string {
	switch n.Kind() {
	case ast.KindText:
		t := n.(*ast.Text)
		raw := string(t.Segment.Value(c.source))
		result := typst.EscapeContent(convertPunctuation(raw))
		if t.SoftLineBreak() {
			result += "\n"
		}
		if t.HardLineBreak() {
			result += " \\\n"
		}
		return result

	case ast.KindString:
		raw := html.UnescapeString(string(n.(*ast.String).Value))
		return typst.EscapeContent(convertPunctuation(raw))

	case ast.KindCodeSpan:
		var code strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Kind() == ast.KindText {
				code.Write(child.(*ast.Text).Segment.Value(c.source))
			}
		}
		return "`" + code.String() + "`"

	case ast.KindEmphasis:
		em := n.(*ast.Emphasis)
		inner := c.renderInlines(n)
		if em.Level == 2 {
			return "#strong[" + inner + "]"
		}
		return "#emph[" + inner + "]"

	case ast.KindLink:
		link := n.(*ast.Link)
		inner := c.renderInlines(n)
		return fmt.Sprintf(`#link("%s")[%s]`, typst.EscapeString(string(link.Destination)), inner)

	case ast.KindAutoLink:
		al := n.(*ast.AutoLink)
		url := string(al.URL(c.source))
		return fmt.Sprintf(`#link("%s")`, typst.EscapeString(url))

	case ast.KindImage:
		return ""

	case ast.KindRawHTML:
		raw := strings.TrimSpace(string(n.Text(c.source)))
		if strings.EqualFold(raw, "<br>") || strings.EqualFold(raw, "<br/>") || strings.EqualFold(raw, "<br />") {
			return "#linebreak()"
		}
		return ""

	default:
		return c.renderInlines(n)
	}
}

// collectImages collects all Image nodes from a paragraph's children.
func (c *converter) collectImages(para ast.Node) []*ast.Image {
	var images []*ast.Image
	for child := para.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindImage {
			images = append(images, child.(*ast.Image))
		}
	}
	return images
}

// renderSingleImage generates Typst figure code for a single image.
func (c *converter) renderSingleImage(img *ast.Image) string {
	c.figureCounter++
	path := string(img.Destination)
	filename := filepath.Base(path)
	caption := strings.TrimSuffix(filename, filepath.Ext(filename))
	escapedPath := typst.EscapeString(path)
	escapedCaption := typst.EscapeContent(caption)

	return fmt.Sprintf(`#figure(
  context {
    let img = image("%s")
    let img-size = measure(img)
    let x = img-size.width
    let y = img-size.height
    let max-size = 13.4cm

    let new-x = x
    let new-y = y

    if x > max-size {
      let scale = max-size / x
      new-x = max-size
      new-y = y * scale
    }

    if new-y > max-size {
      let scale = max-size / new-y
      new-x = new-x * scale
      new-y = max-size
    }

    image("%s", width: new-x, height: new-y)
  },
  placement: auto,
  caption: [%s],
) <fig-%d>
`, escapedPath, escapedPath, escapedCaption, c.figureCounter)
}

// renderMultiImage generates Typst code for multiple images in one paragraph.
func (c *converter) renderMultiImage(images []*ast.Image) string {
	type imgInfo struct {
		path, caption, alt string
		figNum             int
	}

	var infos []imgInfo
	isSubfigure := false

	for _, img := range images {
		alt := c.plainText(img)
		if alt != "" {
			isSubfigure = true
			break
		}
	}

	if isSubfigure {
		c.figureCounter++
	}

	for _, img := range images {
		path := string(img.Destination)
		filename := filepath.Base(path)
		caption := strings.TrimSuffix(filename, filepath.Ext(filename))
		alt := c.plainText(img)
		figNum := 0
		if !isSubfigure {
			c.figureCounter++
			figNum = c.figureCounter
		}
		infos = append(infos, imgInfo{path, caption, alt, figNum})
	}

	var pathsStr, captionsStr, altsStr []string
	mainCaption := ""
	for _, info := range infos {
		pathsStr = append(pathsStr, fmt.Sprintf(`"%s"`, typst.EscapeString(info.path)))
		captionsStr = append(captionsStr, fmt.Sprintf(`"%s"`, typst.EscapeString(info.caption)))
		altsStr = append(altsStr, fmt.Sprintf(`"%s"`, typst.EscapeString(info.alt)))
	}
	if isSubfigure && len(infos) > 0 {
		mainCaption = infos[0].alt
	}

	return fmt.Sprintf(`
#context {
  let paths = (%s)
  let captions = (%s)
  let alts = (%s)

  let is_subfigure = %s
  let main_caption = "%s"

  let gap = 0.3cm
  let max-width = 13.4cm
  let min-height = 6cm

  let sizes = paths.zip(captions).zip(alts).map(item => {
    let p = item.at(0).at(0)
    let c = item.at(0).at(1)
    let alt = item.at(1)
    let img = image(p)
    let s = measure(img)
    (width: s.width, height: s.height, path: p, caption: c, alt: alt, ratio: s.width / s.height)
  })

  let calc-row-height(imgs, total-width) = {
    let ratio-sum = imgs.map(i => i.ratio).sum()
    total-width / ratio-sum
  }

  let rows = ()

  if is_subfigure {
    rows.push(sizes)
  } else {
    let remaining = sizes

    while remaining.len() > 0 {
      let row = ()
      let found = false

      for n in range(1, remaining.len() + 1) {
        let candidate = remaining.slice(0, n)
        let gaps = (n - 1) * gap
        let available-width = max-width - gaps
        let row-h = calc-row-height(candidate, available-width)

        if row-h < min-height and n > 1 {
          row = remaining.slice(0, n - 1)
          remaining = remaining.slice(n - 1)
          found = true
          break
        }
      }

      if not found {
        row = remaining
        remaining = ()
      }

      rows.push(row)
    }
  }

  let render-rows(rows) = {
    for row in rows {
      let n = row.len()
      let gaps = (n - 1) * gap
      let available-width = max-width - gaps
      let row-height = calc-row-height(row, available-width)

      if row-height > max-width {
        row-height = max-width
      }

      align(center, grid(
        columns: n,
        gutter: gap,
        ..row.enumerate().map(item => {
          let i = item.at(0)
          let img-data = item.at(1)
          let w = row-height * img-data.ratio

          if is_subfigure {
             let sub-label = numbering("a", i + 1)
             let sub-text = [ (#sub-label) #img-data.caption ]

             v(0.5em)
             align(center, block({
               image(img-data.path, width: w, height: row-height)
               align(center, text(font: FONT_FS, size: zh(3))[#sub-text])
             }))
          } else {
             figure(
               image(img-data.path, width: w, height: row-height),
               caption: [ #img-data.caption ]
             )
          }
        })
      ))
      if is_subfigure { v(0.5em) } else { v(0.3em) }
    }
  }

  if is_subfigure {
    figure(
      context { render-rows(rows) },
      placement: auto,
      caption: [ #main_caption ]
    )
  } else {
    render-rows(rows)
  }
}

`, strings.Join(pathsStr, ", "), strings.Join(captionsStr, ", "),
		strings.Join(altsStr, ", "), strconv.FormatBool(isSubfigure), typst.EscapeString(mainCaption))
}

// Legacy: {v} / {v:N}. Preferred: {.blank} / {.blank:N} or {.br} / {.br:N}.
var legacyVMarkerRe = regexp.MustCompile(`^\{v(?::(\d+))?\}$`)
var customBlockMarkerRe = regexp.MustCompile(`^\{\.([a-z][a-z0-9-]*)(?::([a-z0-9-]+))?\}$`)

func renderLinebreaks(count int) string {
	if count < 1 {
		count = 1
	}
	var lines []string
	for i := 0; i < count; i++ {
		lines = append(lines, "#linebreak(justify: false)")
	}
	return strings.Join(lines, "\n") + "\n"
}

func markerCount(value string) (int, bool) {
	if value == "" {
		return 1, true
	}
	count, err := strconv.Atoi(value)
	if err != nil || count < 1 {
		return 0, false
	}
	return count, true
}

// processMarker checks if text is a standalone marker and returns Typst code.
func processMarker(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if m := legacyVMarkerRe.FindStringSubmatch(text); m != nil {
		count, _ := markerCount(m[1])
		return renderLinebreaks(count), true
	}
	if m := customBlockMarkerRe.FindStringSubmatch(text); m != nil {
		name := m[1]
		arg := m[2]
		switch name {
		case "br", "blank":
			count, ok := markerCount(arg)
			if !ok {
				return "", false
			}
			return renderLinebreaks(count), true
		case "pagebreak":
			if arg == "" {
				return "#pagebreak()\n", true
			}
			if arg == "weak" {
				return "#pagebreak(weak: true)\n", true
			}
		}
	}
	if text == "{pagebreak}" {
		return "#pagebreak()\n", true
	}
	if text == "{pagebreak:weak}" {
		return "#pagebreak(weak: true)\n", true
	}
	return "", false
}

type trailingStyleMarkers struct {
	noindent bool
	indent   bool
	bold     bool
	raw      []string
}

var trailingStyleMarkerRe = regexp.MustCompile(`\s*(\{\.[a-z][a-z0-9-]*\}|\{indent\})\s*$`)

// stripTrailingStyleMarkers checks for supported trailing style markers.
func stripTrailingStyleMarkers(text string) (string, trailingStyleMarkers) {
	var markers trailingStyleMarkers
	for {
		m := trailingStyleMarkerRe.FindStringSubmatchIndex(text)
		if m == nil {
			break
		}

		raw := text[m[2]:m[3]]
		switch raw {
		case "{.noindent}":
			markers.noindent = true
		case "{.indent}", "{indent}":
			markers.indent = true
		case "{.bold}":
			markers.bold = true
		default:
			return text, markers
		}

		markers.raw = append(markers.raw, raw)
		text = strings.TrimRight(text[:m[0]], " ")
	}
	return text, markers
}

func trimRenderedTrailingMarkers(content string, markers trailingStyleMarkers) string {
	for _, marker := range markers.raw {
		content = strings.TrimRight(content, " \n")
		content = strings.TrimSuffix(content, marker)
	}
	return strings.TrimRight(content, " \n ")
}

func isBodyHeadingLevel(level int) bool {
	return level >= 2 && level <= 5
}

// renderParagraph renders a paragraph node to Typst.
func (c *converter) renderParagraph(para *ast.Paragraph) string {
	images := c.collectImages(para)
	if len(images) == 1 {
		return c.renderSingleImage(images[0])
	}
	if len(images) > 1 {
		return c.renderMultiImage(images)
	}

	plain := c.plainText(para)
	trimmed := strings.TrimSpace(plain)

	if result, ok := processMarker(trimmed); ok {
		return result
	}

	content := c.renderInlines(para)

	_, markers := stripTrailingStyleMarkers(trimmed)
	if markers.noindent {
		content = trimRenderedTrailingMarkers(content, markers)
		return "#block[#set par(first-line-indent: 0pt)\n#block[\n" + content + "\n\n]\n]\n"
	}
	if markers.indent {
		content = trimRenderedTrailingMarkers(content, markers)
		return content + "\n\n"
	}

	if !c.hasSeenHeader {
		t := strings.TrimSpace(content)
		if strings.HasSuffix(t, "：") || strings.HasSuffix(t, ":") {
			return "#block[#set par(first-line-indent: 0pt)\n#block[\n" + content + "\n\n]\n]\n"
		}
	}

	return content + "\n\n"
}

// renderHeading renders a heading node to Typst.
func (c *converter) renderHeading(h *ast.Heading) string {
	c.hasSeenHeader = true

	if h.Level == 1 {
		return ""
	}

	content := c.renderInlines(h)

	_, markers := stripTrailingStyleMarkers(strings.TrimSpace(c.plainText(h)))
	if markers.noindent {
		content = trimRenderedTrailingMarkers(content, markers)
		prefix := strings.Repeat("=", h.Level)
		return "#block[#set par(first-line-indent: 0pt)\n" + prefix + " " + content + "\n]\n\n"
	}
	if markers.bold && isBodyHeadingLevel(h.Level) {
		content = trimRenderedTrailingMarkers(content, markers)
		return fmt.Sprintf("#custom-heading-block(%d, [%s], bold: true)\n\n", h.Level, content)
	}

	prefix := strings.Repeat("=", h.Level)
	return prefix + " " + content + "\n\n"
}

func (c *converter) renderRunInHeadingParagraph(h *ast.Heading, para *ast.Paragraph) string {
	c.hasSeenHeader = true

	content := c.renderInlines(h)
	_, markers := stripTrailingStyleMarkers(strings.TrimSpace(c.plainText(h)))
	if len(markers.raw) > 0 {
		content = trimRenderedTrailingMarkers(content, markers)
	}

	body := strings.TrimLeft(c.renderInlines(para), " \n")
	return fmt.Sprintf("#custom-heading(%d, [%s], bold: %s)%s\n\n", h.Level, content, strconv.FormatBool(markers.bold), body)
}

// renderList renders a list node to Typst.
func (c *converter) renderList(list *ast.List) string {
	var buf strings.Builder
	marker := "- "
	if list.IsOrdered() {
		marker = "+ "
	}
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindListItem {
			buf.WriteString(marker)
			buf.WriteString(c.renderListItem(child))
			buf.WriteString("\n")
		}
	}
	buf.WriteString("\n")
	return buf.String()
}

// renderListItem renders a list item's content.
func (c *converter) renderListItem(item ast.Node) string {
	var parts []string
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case ast.KindParagraph:
			content := c.renderInlines(child)
			content = strings.TrimRight(content, "\n")
			parts = append(parts, content)
		case ast.KindList:
			parts = append(parts, c.renderList(child.(*ast.List)))
		default:
			content := c.renderInlines(child)
			if content == "" {
				for gc := child.FirstChild(); gc != nil; gc = gc.NextSibling() {
					content += c.renderInline(gc)
				}
			}
			content = strings.TrimRight(content, "\n")
			if content != "" {
				parts = append(parts, content)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// isHTMLComment checks if a node is an HTML block containing the given keyword.
func isHTMLComment(n ast.Node, source []byte, keyword string) bool {
	if n.Kind() != ast.KindHTMLBlock {
		return false
	}
	lines := n.Lines()
	if lines.Len() == 0 {
		return false
	}
	seg := lines.At(0)
	return strings.Contains(string(seg.Value(source)), keyword)
}

func (c *converter) shouldRunInHeadingParagraph(heading ast.Node, next ast.Node) bool {
	h, ok := heading.(*ast.Heading)
	if !ok || !isBodyHeadingLevel(h.Level) {
		return false
	}
	para, ok := next.(*ast.Paragraph)
	if !ok {
		return false
	}
	if len(c.collectImages(para)) > 0 {
		return false
	}
	plain := strings.TrimSpace(c.plainText(para))
	if plain == "" {
		return false
	}
	if _, ok := processMarker(plain); ok {
		return false
	}

	headingLines := heading.Lines()
	paraLines := para.Lines()
	if headingLines.Len() == 0 || paraLines.Len() == 0 {
		return false
	}

	betweenStart := headingLines.At(headingLines.Len() - 1).Stop
	betweenEnd := paraLines.At(0).Start
	if betweenStart > betweenEnd || betweenEnd > len(c.source) {
		return false
	}

	between := string(c.source[betweenStart:betweenEnd])
	return strings.Count(between, "\n") <= 1 && strings.TrimSpace(between) == ""
}

// renderDocumentBlocks renders top-level document blocks while preserving
// table captions and noindent regions that can consume multiple AST nodes.
func (c *converter) renderDocumentBlocks(doc ast.Node) []string {
	var blocks []string
	child := doc.FirstChild()

	for child != nil {
		if isHTMLComment(child, c.source, "noindent-start") {
			child = child.NextSibling()
			var innerBuf strings.Builder
			for child != nil && !isHTMLComment(child, c.source, "noindent-end") {
				innerBuf.WriteString(c.renderBlock(child, true))
				child = child.NextSibling()
			}
			if child != nil {
				child = child.NextSibling()
			}
			inner := innerBuf.String()
			blocks = append(blocks, "#block[#set par(first-line-indent: 0pt)\n#block[\n"+inner+"]\n]\n")
		} else if child.Kind() == east.KindTable {
			// Check for Pandoc-style table caption (": caption" after table)
			caption := ""
			next := child.NextSibling()
			if next != nil && next.Kind() == ast.KindParagraph {
				paraText := strings.TrimSpace(c.nodeText(next))
				if strings.HasPrefix(paraText, ": ") {
					caption = strings.TrimPrefix(paraText, ": ")
					next = next.NextSibling() // Skip the caption paragraph
				}
			}

			blocks = append(blocks, c.renderTableWithCaption(child, caption))

			child = next
		} else if next := child.NextSibling(); c.shouldRunInHeadingParagraph(child, next) {
			blocks = append(blocks, c.renderRunInHeadingParagraph(child.(*ast.Heading), next.(*ast.Paragraph)))
			child = next.NextSibling()
		} else {
			blocks = append(blocks, c.renderBlock(child, false))
			child = child.NextSibling()
		}
	}

	return blocks
}

// renderDocument renders the full document body.
func (c *converter) renderDocument(doc ast.Node) string {
	return strings.Join(c.renderDocumentBlocks(doc), "")
}

// renderBlock renders a single block-level node.
func (c *converter) renderBlock(n ast.Node, inNoindent bool) string {
	switch n.Kind() {
	case ast.KindParagraph:
		return c.renderParagraph(n.(*ast.Paragraph))
	case ast.KindHeading:
		return c.renderHeading(n.(*ast.Heading))
	case ast.KindList:
		content := c.renderList(n.(*ast.List))
		if inNoindent {
			return "#block[#set par(first-line-indent: 0pt)\n" + content + "]\n"
		}
		return content
	case ast.KindFencedCodeBlock, ast.KindCodeBlock:
		return c.renderCodeBlock(n)
	case ast.KindThematicBreak:
		return "#line(length: 100%)\n\n"
	case ast.KindBlockquote:
		return c.renderBlockquote(n)
	case east.KindTable:
		return c.renderTable(n)
	case ast.KindHTMLBlock:
		return ""
	default:
		var buf strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			buf.WriteString(c.renderBlock(child, inNoindent))
		}
		return buf.String()
	}
}

// renderCodeBlock renders a fenced or indented code block.
func (c *converter) renderCodeBlock(n ast.Node) string {
	var buf strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(c.source))
	}
	code := buf.String()

	lang := ""
	if fcb, ok := n.(*ast.FencedCodeBlock); ok {
		if fcb.Info != nil {
			lang = string(fcb.Info.Segment.Value(c.source))
			lang = strings.TrimSpace(strings.SplitN(lang, " ", 2)[0])
		}
	}

	if lang != "" {
		return "#code-block[```" + lang + "\n" + code + "```]\n\n"
	}
	return "#code-block[```\n" + code + "```]\n\n"
}

// renderBlockquote renders a blockquote.
func (c *converter) renderBlockquote(n ast.Node) string {
	var buf strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		content := c.renderBlock(child, false)
		for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
			buf.WriteString("#quote[" + line + "]\n")
		}
	}
	buf.WriteString("\n")
	return buf.String()
}

// renderTable renders a GFM table to Typst table syntax.
// Uses a three-line style (三线表) consistent with GB/T 9704 document standards.
func (c *converter) renderTable(n ast.Node) string {
	rows, colAligns := c.extractTable(n)
	if len(rows) == 0 {
		return ""
	}
	return c.renderTableSeries(rows, colAligns, "", 0)
}

func (c *converter) extractTable(n ast.Node) ([][]cellInfo, []east.Alignment) {
	var rows [][]cellInfo
	var colAligns []east.Alignment

	// Collect all rows and cells
	for row := n.FirstChild(); row != nil; row = row.NextSibling() {
		var cells []cellInfo
		var isHeader bool

		switch row.Kind() {
		case east.KindTableHeader:
			isHeader = true
			cells = c.collectTableCells(row)
		case east.KindTableRow:
			tr := row.(*east.TableRow)
			if len(colAligns) == 0 {
				colAligns = tr.Alignments
			}
			isHeader = false
			cells = c.collectTableCells(row)
		default:
			continue
		}

		if len(cells) > 0 {
			for i := range cells {
				cells[i].isHeader = isHeader
			}
			rows = append(rows, cells)
		}
	}

	return rows, colAligns
}

const tablePageLineBudget = 22
const tableGridLineHeight = "((297mm - 37mm - 35mm) / 22)"
const tableCellVerticalInset = "((" + tableGridLineHeight + " - zh(3)) / 2)"
const tableCaptionBottomGap = "(((297mm - 37mm - 35mm) / 22) - zh(3))"

func estimatedTableRowLines(row []cellInfo) int {
	maxCellLines := 1
	for _, cell := range row {
		cellLines := 0
		for _, segment := range strings.Split(strings.TrimSpace(cell.content), "#linebreak()") {
			lines := (len([]rune(segment)) + 21) / 22
			if lines < 1 {
				lines = 1
			}
			cellLines += lines
		}
		if cellLines > maxCellLines {
			maxCellLines = cellLines
		}
	}
	return maxCellLines
}

func estimatedTableLines(rows [][]cellInfo, hasCaption bool) int {
	lines := 0
	if hasCaption {
		lines += 2
	}
	if len(rows) > 0 {
		lines += 2
	}
	for _, row := range rows[1:] {
		lines += estimatedTableRowLines(row)
	}
	return lines
}

func (c *converter) renderTableSeries(rows [][]cellInfo, colAligns []east.Alignment, caption string, captionNumber int) string {
	if estimatedTableLines(rows, caption != "") <= tablePageLineBudget {
		return c.renderTableBlock(rows, colAligns, caption, captionNumber, true, false)
	}

	return c.renderTableBlock(rows, colAligns, caption, captionNumber, false, caption != "")
}

func writeTableColumns(buf *strings.Builder, maxCols int) {
	buf.WriteString("  columns: (")
	for i := 0; i < maxCols; i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("auto")
	}
	buf.WriteString("),\n")
}

func writeTableAlignments(buf *strings.Builder, maxCols int, colAligns []east.Alignment) {
	buf.WriteString("  align: (")
	for i := 0; i < maxCols; i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		align := alignLeft
		if i < len(colAligns) {
			align = typstAlign(colAligns[i])
		}
		buf.WriteString(string(align))
	}
	buf.WriteString("),\n")
}

func tableCellContent(content string, strong bool) string {
	trimmed := strings.TrimRight(content, "\n")
	if strong {
		trimmed = "#strong[" + trimmed + "]"
	}
	return "table.cell(inset: (x: 2pt, y: " + tableCellVerticalInset + "))[#set par(leading: " + tableCaptionBottomGap + ", spacing: 0pt, first-line-indent: 0pt)\n" + trimmed + "]"
}

func writeMeasuredTableWidth(buf *strings.Builder, rows [][]cellInfo, colAligns []east.Alignment, maxCols int, continuedCaptionText string) {
	buf.WriteString("  let table-caption-width = calc.max(\n")
	buf.WriteString("    measure(text(font: FONT_FS, size: zh(3))[" + continuedCaptionText + "]).width,\n")
	buf.WriteString("    measure(table(\n")
	writeTableColumns(buf, maxCols)
	writeTableAlignments(buf, maxCols, colAligns)
	buf.WriteString("      stroke: none,\n")
	if len(rows) > 0 {
		for _, cell := range rows[0] {
			buf.WriteString("      " + tableCellContent(cell.content, true) + ",\n")
		}
	}
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		for _, cell := range rows[rowIdx] {
			buf.WriteString("      " + tableCellContent(cell.content, false) + ",\n")
		}
	}
	buf.WriteString("    )).width,\n")
	buf.WriteString("  )\n")
}

func tableCaptionCellContent(captionText string) string {
	return "box(width: table-caption-width)[#align(center)[#pad(bottom: " + tableCaptionBottomGap + ")[#text(font: FONT_FS, size: zh(3))[" + captionText + "]]]]"
}

func tableCaptionCell(captionText string, maxCols int) string {
	return "  table.cell(colspan: " + strconv.Itoa(maxCols) + ", align: center, stroke: none, inset: 0pt)[#align(center)[#pad(bottom: " + tableCaptionBottomGap + ")[#text(font: FONT_FS, size: zh(3))[" + captionText + "]]]],\n"
}

func (c *converter) renderTableBlock(rows [][]cellInfo, colAligns []east.Alignment, caption string, captionNumber int, keepTogether bool, repeatCaption bool) string {
	// Determine column count (max across all rows)
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Count total rows
	totalRows := len(rows)

	// Build Typst table using table.hline for three-line style
	var buf strings.Builder
	if keepTogether {
		buf.WriteString("#block(breakable: false, width: 100%)[\n")
	}

	if repeatCaption {
		captionText := "表" + strconv.Itoa(captionNumber)
		continuedCaptionText := captionText
		escapedCaption := typst.EscapeContent(caption)
		if escapedCaption != "" {
			captionText += "#h(1em)" + escapedCaption
			continuedCaptionText += "#h(1em)" + escapedCaption + "（续）"
		} else {
			continuedCaptionText += "（续）"
		}
		buf.WriteString("#context {\n")
		buf.WriteString("  let table-start-page = here().page()\n")
		writeMeasuredTableWidth(&buf, rows, colAligns, maxCols, continuedCaptionText)
		buf.WriteString("  align(center)[\n")
		buf.WriteString("  #table(\n")
	} else {
		buf.WriteString("#align(center)[\n")
		buf.WriteString("#table(\n")
	}

	// Keep columns auto-sized so tables stay close to their natural content
	// width; continuation captions get their own minimum width below.
	writeTableColumns(&buf, maxCols)

	// Alignment per column
	writeTableAlignments(&buf, maxCols, colAligns)

	// No default strokes
	buf.WriteString("  stroke: none,\n")

	headerRows := 1
	if caption != "" {
		headerRows = 2
	}

	// Top line (before the column header row; captions are above the line)
	buf.WriteString("  table.hline(y: " + strconv.Itoa(headerRows-1) + ", stroke: 0.75pt),\n")

	// Header bottom line (after row 0 = before row 1)
	buf.WriteString("  table.hline(y: " + strconv.Itoa(headerRows) + ", stroke: 0.5pt),\n")

	// Bottom line (after last row = before row totalRows)
	buf.WriteString("  table.hline(y: " + strconv.Itoa(totalRows+headerRows-1) + ", stroke: 0.75pt),\n")

	// Caption + header rows. Table captions are rendered as the first table row
	// for both short and long tables so the caption-to-rule spacing is identical.
	if len(rows) > 0 {
		if repeatCaption {
			buf.WriteString("  table.header(repeat: true,\n")
			captionText := "表" + strconv.Itoa(captionNumber)
			continuedCaptionText := captionText
			escapedCaption := typst.EscapeContent(caption)
			if escapedCaption != "" {
				captionText += "#h(1em)" + escapedCaption
				continuedCaptionText += "#h(1em)" + escapedCaption + "（续）"
			} else {
				continuedCaptionText += "（续）"
			}
			buf.WriteString("  table.cell(colspan: " + strconv.Itoa(maxCols) + ", align: center, stroke: none, inset: 0pt)[#context if here().page() == table-start-page { " + tableCaptionCellContent(captionText) + " } else { " + tableCaptionCellContent(continuedCaptionText) + " }],\n")
			for _, cell := range rows[0] {
				buf.WriteString("  " + tableCellContent(cell.content, true) + ",\n")
			}
			buf.WriteString("  ),\n")
		} else {
			if caption != "" {
				captionText := "表" + strconv.Itoa(captionNumber) + "#h(1em)" + typst.EscapeContent(caption)
				buf.WriteString(tableCaptionCell(captionText, maxCols))
			}
			for _, cell := range rows[0] {
				buf.WriteString("  " + tableCellContent(cell.content, true) + ",\n")
			}
		}
	}

	// Body rows
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		for _, cell := range row {
			buf.WriteString("  " + tableCellContent(cell.content, false) + ",\n")
		}
	}

	buf.WriteString(")\n")
	buf.WriteString("]\n\n")
	if repeatCaption {
		buf.WriteString("}\n\n")
	}

	if keepTogether {
		buf.WriteString("]\n\n")
	}

	return buf.String()
}

// renderTableWithCaption renders a table with optional caption.
// If caption is provided, adds "表N caption" above the table.
func (c *converter) renderTableWithCaption(n ast.Node, caption string) string {
	rows, colAligns := c.extractTable(n)
	if len(rows) == 0 {
		return ""
	}
	captionNumber := 0
	if caption != "" {
		c.tableCounter++
		captionNumber = c.tableCounter
	}

	return c.renderTableSeries(rows, colAligns, caption, captionNumber)
}

// collectTableCells extracts cell content from a TableRow or TableHeader node.
func (c *converter) collectTableCells(row ast.Node) []cellInfo {
	var cells []cellInfo
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		if cell.Kind() != east.KindTableCell {
			continue
		}
		content := c.renderInlines(cell)
		cells = append(cells, cellInfo{
			content: content,
		})
	}
	return cells
}

type cellInfo struct {
	content  string
	isHeader bool
}

type cellAlignment string

const (
	alignLeft   cellAlignment = "left"
	alignCenter cellAlignment = "center"
	alignRight  cellAlignment = "right"
)

func typstAlign(a east.Alignment) cellAlignment {
	switch a {
	case east.AlignLeft:
		return alignLeft
	case east.AlignRight:
		return alignRight
	case east.AlignCenter:
		return alignCenter
	default:
		return alignLeft
	}
}

// convertBody parses markdown body and renders to Typst.
func convertBody(body string) string {
	return strings.Join(convertBodyBlocks(body), "")
}

func convertBodyBlocks(body string) []string {
	body = preprocessBody(body)
	source := []byte(body)

	md := goldmark.New(
		goldmark.WithExtensions(extension.Table),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	doc := md.Parser().Parse(text.NewReader(source))

	conv := &converter{source: source}
	return conv.renderDocumentBlocks(doc)
}

func shouldStickSignatureToBlock(block string) bool {
	trimmed := strings.TrimSpace(block)
	if trimmed == "" {
		return false
	}
	if strings.Contains(trimmed, "#pagebreak") || strings.Contains(trimmed, "#place(") || strings.Contains(trimmed, "#figure(") {
		return false
	}
	if strings.Contains(trimmed, "#table(") && !strings.Contains(trimmed, "#block(breakable: false)") {
		return false
	}
	return true
}

func renderSignatureBlock() string {
	return `#v(18pt)
#block(width: 100%)[
#align(right)[
#box[
  #set align(center)
  #autoAuthor \
  #autoDate.display(
    "[year]年[month padding:none]月[day padding:none]日",
  )
]
]
]
`
}

// convert takes parsed front-matter and markdown body, returns full .typ output.
func convert(fm frontMatter, body string) string {
	var out strings.Builder

	out.WriteString(templateHead)
	fmt.Fprintf(&out, "#let autoTitle = \"%s\"\n\n", typst.EscapeString(fm.Title))
	fmt.Fprintf(&out, "#let autoAuthor = \"%s\"\n\n", typst.EscapeString(fm.Author))
	fmt.Fprintf(&out, "#let autoDate = %s\n\n", formatDate(fm.Date))

	out.WriteString(`#set document(
  title: autoTitle.replace("|", " "),
  author: autoAuthor,
  keywords: "工作总结, 年终报告",
  date: autoDate,
)

= #autoTitle.split("|").map(s => s.trim()).join(linebreak())

`)

	if !fm.Signature {
		out.WriteString("#name(autoAuthor)\n")
	}
	out.WriteString("\n")

	if fm.Signature {
		blocks := convertBodyBlocks(body)
		if len(blocks) > 0 && shouldStickSignatureToBlock(blocks[len(blocks)-1]) {
			for _, block := range blocks[:len(blocks)-1] {
				out.WriteString(block)
			}
			out.WriteString("#place.flush()\n")
			out.WriteString("#block(sticky: true, width: 100%)[\n")
			out.WriteString(blocks[len(blocks)-1])
			out.WriteString(renderSignatureBlock())
			out.WriteString("]\n")
		} else {
			out.WriteString(strings.Join(blocks, ""))
			out.WriteString("#place.flush()\n")
			out.WriteString(renderSignatureBlock())
		}
	} else {
		out.WriteString(convertBody(body))
	}

	return out.String()
}

// ---------- CLI ----------

func main() {
	cli.Run(manifestJSON, exampleMD, func(input string) string {
		fm, body := parseFrontMatter(input)
		return convert(fm, body)
	}, func(input string) cli.OutputInfo {
		fm, _ := parseFrontMatter(input)
		title := strings.TrimSpace(strings.ReplaceAll(fm.Title, "|", " "))
		if title == "" {
			title = "output"
		}
		authors := []string{}
		if strings.TrimSpace(fm.Author) != "" && fm.Author != "请输入文字" {
			authors = []string{fm.Author}
		}
		return cli.OutputInfo{
			SchemaVersion:  1,
			OutputBaseName: cli.CleanFilenameBase(title),
			PreviewTitle:   title,
			Document: cli.DocumentInfo{
				Title:       title,
				Authors:     authors,
				Date:        fm.Date,
				Keywords:    []string{"公文", "通知", "报告", "GB/T 9704"},
				Subject:     "类公文文档",
				Description: "Presto 类公文模板生成的 PDF",
				Language:    "zh-CN",
			},
		}
	})
}
