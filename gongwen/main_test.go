package main

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestExampleOutputGoldenHash(t *testing.T) {
	fm, body := parseFrontMatter(exampleMD)
	output := convert(fm, body)
	got := fmt.Sprintf("%x", sha256.Sum256([]byte(output)))
	const want = "6a9f12f5be9c8ca6d1b81d9459e14aa7723a4fc35e699999c169716f74bc8678"
	if got != want {
		t.Fatalf("example output changed: got %s, want %s", got, want)
	}
}

func TestSingleImagesUseFloatingPlacement(t *testing.T) {
	output := convertBody(`![流程图](assets/process.png)
`)

	if !strings.Contains(output, "placement: auto") {
		t.Fatal("expected single images to use Typst floating placement")
	}
	if !strings.Contains(output, `image("assets/process.png"`) {
		t.Fatal("expected image path to be rendered")
	}
}

func TestShortTablesStayInFlowCenteredAndUnsplit(t *testing.T) {
	output := convertBody(`| 阶段 | 时间 | 负责部门 |
|------|------|----------|
| 自查自纠 | 3月1日-15日 | 各部门 |
| 集中检查 | 3月16日-31日 | 安全管理处 |

: 检查工作进度安排
`)

	if strings.Contains(output, "#place(auto, float: true)[") {
		t.Fatal("expected short tables to stay in document flow")
	}
	if !strings.Contains(output, "#block(breakable: false, width: 100%)[") {
		t.Fatal("expected short tables to stay unsplit in a full-width in-flow block")
	}
	if !strings.Contains(output, "table.cell(colspan: 3, align: center, stroke: none, inset: 0pt)[#align(center)[#pad(bottom: (((297mm - 37mm - 35mm) / 22) - zh(3)))[#text(font: FONT_FS, size: zh(3))[表1#h(1em)检查工作进度安排]]]],") {
		t.Fatal("expected short table captions to use the shared table-row caption spacing")
	}
	if strings.Contains(output, "table.header(repeat: true,") {
		t.Fatal("expected short non-paginated tables to use a plain bold header row")
	}
	if !strings.Contains(output, "#align(center)[\n#table(") {
		t.Fatal("expected table to stay centered")
	}
	if !strings.Contains(output, "表1#h(1em)检查工作进度安排") {
		t.Fatal("expected table caption to stay with the table")
	}
	if strings.Contains(output, "（续）") {
		t.Fatal("expected short tables not to generate continuation captions")
	}
	if count := strings.Count(output, "#table("); count != 1 {
		t.Fatalf("expected short table not to split, got %d table chunks", count)
	}
}

func TestSignatureFlushesFloatsAndSticksToLastParagraph(t *testing.T) {
	output := convert(frontMatter{
		Title:     "测试通知",
		Author:    "测试处",
		Date:      "2026-05-06",
		Signature: true,
	}, `## 工作安排

| 阶段 | 时间 |
|------|------|
| 自查 | 3月 |
| 检查 | 4月 |

特此通知。
`)

	tableIdx := strings.Index(output, "#align(center)[\n#table(")
	flushIdx := strings.Index(output, "#place.flush()")
	stickyIdx := strings.Index(output, "#block(sticky: true, width: 100%)[")
	signatureIdx := strings.Index(output, "#block(width: 100%)[\n#align(right)[")

	if tableIdx < 0 {
		t.Fatal("expected short table to render")
	}
	if !(tableIdx < flushIdx && flushIdx < stickyIdx && stickyIdx < signatureIdx) {
		t.Fatalf("expected pending floats to flush before sticky signature, got table=%d flush=%d sticky=%d signature=%d", tableIdx, flushIdx, stickyIdx, signatureIdx)
	}
	if !strings.Contains(output[stickyIdx:signatureIdx], "特此通知。") {
		t.Fatal("expected the final paragraph to stick with the signature")
	}
	if !strings.Contains(output, "#box[\n  #set align(center)") {
		t.Fatal("expected signature content to keep centered author/date inside the right-aligned box")
	}
}

func TestLongTablesRepeatContinuationCaptionInHeader(t *testing.T) {
	var input strings.Builder
	input.WriteString("| 序号 | 内容 |\n")
	input.WriteString("|------|------|\n")
	for i := 0; i < 28; i++ {
		input.WriteString("| ")
		input.WriteString("1")
		input.WriteString(" | 事项 |\n")
	}
	input.WriteString("\n: 事项清单\n")

	output := convertBody(input.String())

	if !strings.Contains(output, "table-start-page = here().page()") {
		t.Fatal("expected long tables to track their starting page")
	}
	if !strings.Contains(output, "columns: (auto, auto),") {
		t.Fatal("expected long tables to keep auto-sized columns")
	}
	if strings.Contains(output, "1fr") {
		t.Fatal("expected long tables not to stretch the last column")
	}
	if !strings.Contains(output, "let table-caption-width = calc.max(") {
		t.Fatal("expected long tables to measure dynamic continuation caption width")
	}
	if !strings.Contains(output, "measure(text(font: FONT_FS, size: zh(3))[表1#h(1em)事项清单（续）]).width") {
		t.Fatal("expected continuation caption width to be measured from the actual caption")
	}
	if !strings.Contains(output, "(((297mm - 37mm - 35mm) / 22) - zh(3))") {
		t.Fatal("expected table caption spacing to be derived from the 22-line page grid")
	}
	if !strings.Contains(output, "table.cell(colspan: 2, align: center, stroke: none, inset: 0pt)[#context if here().page() == table-start-page") {
		t.Fatal("expected repeated table captions to use the same zero-inset table caption row")
	}
	if !strings.Contains(output, "box(width: table-caption-width)[#align(center)[#pad(bottom: (((297mm - 37mm - 35mm) / 22) - zh(3)))[#text(font: FONT_FS, size: zh(3))[表1#h(1em)事项清单（续）]]]]") {
		t.Fatal("expected continuation caption to use the same centered text style and grid-derived spacing as standalone captions")
	}
	if strings.Contains(output, "pad(bottom: 8pt)") {
		t.Fatal("expected continuation captions not to use a hand-tuned fixed gap")
	}
	if strings.Contains(output, "box(width: 12em") || strings.Contains(output, "box(width: 10em") {
		t.Fatal("expected continuation captions not to use fixed em widths")
	}
	if !strings.Contains(output, "表1#h(1em)事项清单") {
		t.Fatal("expected first page table header to keep the original caption")
	}
	if !strings.Contains(output, "表1#h(1em)事项清单（续）") {
		t.Fatal("expected repeated table headers to use continuation captions")
	}
	if strings.Contains(output, "续表1") {
		t.Fatal("expected continuation captions to use 表N 表题（续）, not 续表N")
	}
	if !strings.Contains(output, "table.header(repeat: true,") {
		t.Fatal("expected long tables to repeat their header")
	}
	if count := strings.Count(output, "#table("); count != 1 {
		t.Fatalf("expected long table to remain one naturally breaking table, got %d table chunks", count)
	}
	if strings.Contains(output, "breakable: false") {
		t.Fatal("expected long tables to use normal table flow")
	}
	if strings.Contains(output, "#pagebreak(weak: true)") {
		t.Fatal("expected long tables not to force page breaks")
	}
	if strings.Contains(output, "#place(auto, float: true)[") {
		t.Fatal("expected over-one-page tables not to float")
	}
}

func TestShortTableCaptionSpacingUsesPageGrid(t *testing.T) {
	output := convertBody(`| 阶段 | 时间 |
|------|------|
| 自查 | 3月 |

: 工作安排
`)

	if !strings.Contains(output, "table.cell(colspan: 2, align: center, stroke: none, inset: 0pt)[#align(center)[#pad(bottom: (((297mm - 37mm - 35mm) / 22) - zh(3)))[#text(font: FONT_FS, size: zh(3))[表1#h(1em)工作安排]]]],") {
		t.Fatal("expected standalone table captions to use the shared table-row spacing")
	}
}

func TestTableCellsUsePageGridSpacing(t *testing.T) {
	output := convertBody(`| 项目 | 说明 |
|------|------|
| A | 第一行<br>第二行 |
| B | 第三行 |
`)

	cellInset := "table.cell(inset: (x: 2pt, y: ((((297mm - 37mm - 35mm) / 22) - zh(3)) / 2)))"
	if count := strings.Count(output, cellInset); count != 6 {
		t.Fatalf("expected all table cells to use grid-derived inset, got %d:\n%s", count, output)
	}
	if !strings.Contains(output, "#set par(leading: (((297mm - 37mm - 35mm) / 22) - zh(3)), spacing: 0pt, first-line-indent: 0pt)") {
		t.Fatalf("expected table cell paragraphs to use grid-derived leading:\n%s", output)
	}
	if !strings.Contains(output, "第一行#linebreak()第二行") {
		t.Fatalf("expected table <br> to render as a Typst linebreak:\n%s", output)
	}
}

func TestTableRowLineEstimateCountsExplicitBreaks(t *testing.T) {
	got := estimatedTableRowLines([]cellInfo{{content: "第一行#linebreak()第二行"}})
	if got != 2 {
		t.Fatalf("expected explicit table linebreak to count as two lines, got %d", got)
	}
}

func TestAdjacentHeadingAndParagraphRenderRunIn(t *testing.T) {
	cases := []struct {
		name string
		md   string
		want string
	}{
		{
			name: "level 2",
			md: `## 工作要求
各部门要严格落实责任。
`,
			want: "#custom-heading(2, [工作要求], bold: false)各部门要严格落实责任。\n\n",
		},
		{
			name: "level 3",
			md: `### 工作要求
各部门要严格落实责任。
`,
			want: "#custom-heading(3, [工作要求], bold: false)各部门要严格落实责任。\n\n",
		},
		{
			name: "level 4",
			md: `#### 工作要求
各部门要严格落实责任。
`,
			want: "#custom-heading(4, [工作要求], bold: false)各部门要严格落实责任。\n\n",
		},
		{
			name: "level 5",
			md: `##### 工作要求
各部门要严格落实责任。
`,
			want: "#custom-heading(5, [工作要求], bold: false)各部门要严格落实责任。\n\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := convertBody(tc.md)
			if output != tc.want {
				t.Fatalf("expected adjacent heading and paragraph to render on one line:\n%s", output)
			}
		})
	}
}

func TestSeparatedHeadingKeepsStandaloneSpacing(t *testing.T) {
	output := convertBody(`## 工作要求

各部门要严格落实责任。
`)

	if !strings.Contains(output, "== 工作要求\n\n各部门要严格落实责任。\n\n") {
		t.Fatalf("expected blank-line separated heading to keep standalone rendering:\n%s", output)
	}
}

func TestCustomBlockMarkersUseUnifiedDotSyntax(t *testing.T) {
	cases := []struct {
		name string
		md   string
		want string
	}{
		{name: "blank defaults to one line", md: "{.blank}\n", want: "#linebreak(justify: false)\n"},
		{name: "blank count", md: "{.blank:3}\n", want: "#linebreak(justify: false)\n#linebreak(justify: false)\n#linebreak(justify: false)\n"},
		{name: "br alias", md: "{.br:2}\n", want: "#linebreak(justify: false)\n#linebreak(justify: false)\n"},
		{name: "pagebreak", md: "{.pagebreak}\n", want: "#pagebreak()\n"},
		{name: "weak pagebreak", md: "{.pagebreak:weak}\n", want: "#pagebreak(weak: true)\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := convertBody(tc.md); got != tc.want {
				t.Fatalf("unexpected marker output:\ngot:  %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestLegacyCustomBlockMarkersRemainSupported(t *testing.T) {
	cases := []struct {
		name string
		md   string
		want string
	}{
		{name: "legacy blank", md: "{v:2}\n", want: "#linebreak(justify: false)\n#linebreak(justify: false)\n"},
		{name: "legacy pagebreak", md: "{pagebreak}\n", want: "#pagebreak()\n"},
		{name: "legacy weak pagebreak", md: "{pagebreak:weak}\n", want: "#pagebreak(weak: true)\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := convertBody(tc.md); got != tc.want {
				t.Fatalf("unexpected legacy marker output:\ngot:  %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestParagraphIndentMarkersUseUnifiedDotSyntax(t *testing.T) {
	noindent := convertBody("这是一段顶格文字。 {.noindent}\n")
	if !strings.Contains(noindent, "#set par(first-line-indent: 0pt)") {
		t.Fatalf("expected noindent marker to disable first-line indent:\n%s", noindent)
	}
	if strings.Contains(noindent, "{.noindent}") {
		t.Fatalf("expected noindent marker to be stripped:\n%s", noindent)
	}

	indent := convertBody("这是一段恢复缩进文字。 {.indent}\n")
	if indent != "这是一段恢复缩进文字。\n\n" {
		t.Fatalf("expected indent marker to be stripped while preserving paragraph:\n%s", indent)
	}
}

func TestHeadingPartialBoldKeepsStrongInHeadingBody(t *testing.T) {
	output := convertBody(`## 工作**要求**
各部门要严格落实责任。
`)

	want := "#custom-heading(2, [工作#strong[要求]], bold: false)各部门要严格落实责任。\n\n"
	if output != want {
		t.Fatalf("expected partial heading bold to stay in heading body:\n%s", output)
	}
}

func TestHeadingBoldMarkerBoldensNumberAndHeading(t *testing.T) {
	output := convertBody(`## 工作要求 {.bold}
各部门要严格落实责任。
`)

	want := "#custom-heading(2, [工作要求], bold: true)各部门要严格落实责任。\n\n"
	if output != want {
		t.Fatalf("expected bold marker to request whole heading bold:\n%s", output)
	}

	standalone := convertBody(`## 工作要求 {.bold}

各部门要严格落实责任。
`)

	if !strings.Contains(standalone, "#custom-heading-block(2, [工作要求], bold: true)") {
		t.Fatalf("expected standalone bold marker to request whole heading block bold:\n%s", standalone)
	}
	if strings.Contains(standalone, "{.bold}") {
		t.Fatalf("expected bold marker to be stripped from output:\n%s", standalone)
	}
}

func TestTemplateUsesTextWeightForWholeHeadingBold(t *testing.T) {
	if strings.Contains(templateHead, "strong(body)") || strings.Contains(templateHead, "maybe-strong") {
		t.Fatal("expected whole-heading bold to use explicit text weight, not strong()")
	}
	if !strings.Contains(templateHead, `#let maybe-bold(enabled, body) = if enabled {`) {
		t.Fatal("expected template to define maybe-bold helper")
	}
	if !strings.Contains(templateHead, `text(weight: "bold", stroke: 0.2pt + black)[#body]`) {
		t.Fatal("expected bold headings to set explicit text weight and stroke")
	}
	for _, want := range []string{
		`#maybe-bold(bold)[#context h2-counter.display("一、")#body]`,
		`#maybe-bold(bold)[#context h3-counter.display("（一）")#body]`,
		`#maybe-bold(bold)[#context h4-counter.display("1.")#body]`,
		`#maybe-bold(bold)[#context h5-counter.display("（1）")#body]`,
	} {
		if !strings.Contains(templateHead, want) {
			t.Fatalf("expected bold wrapper around heading number and body:\n%s", want)
		}
	}
}

func TestCodeBlocksUseUnifiedCodeFont(t *testing.T) {
	output := convertBody("```go\nfunc main() { println(\"中文测试\") }\n```\n")

	if !strings.Contains(output, "#code-block[```go\nfunc main() { println(\"中文测试\") }\n```]\n\n") {
		t.Fatalf("expected fenced code block to use the code-block wrapper:\n%s", output)
	}
	if !strings.Contains(templateHead, `#let FONT_CODE = "Noto Sans Mono CJK SC"`) {
		t.Fatal("expected template to define a stable CJK monospace code font")
	}
	if !strings.Contains(templateHead, "#let code-block(body) = block(width: 100%)[") {
		t.Fatal("expected template to define code-block wrapper")
	}
}
