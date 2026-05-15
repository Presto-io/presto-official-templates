package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestDefinesJiaoanJihua(t *testing.T) {
	for _, want := range []string{`"name": "jiaoan-jihua"`, `"displayName": "授课进度计划表模板"`, `"author": "Presto-io"`, `"category": "教育"`, `"name": "STSong"`, `"displayName": "华文宋体"`, `"url": "https://www.foundertype.com/index.php/FontInfo/index/id/135"`, `"frontmatterSchema"`, `"school_year"`, `"first_teaching_day"`, `"calendar_json"`, `"template"`} {
		if !strings.Contains(manifestJSON, want) {
			t.Fatalf("manifest missing %q", want)
		}
	}
}

func TestExampleContainsMinimumContract(t *testing.T) {
	for _, want := range []string{`template: "jiaoan-jihua"`, `calendar_json: "presto/calendar.json"`, `## CA6140卧式车床电气控制线路安装与调试`, `## X62W万能铣床电气控制线路安装与调试`, `### 安技教育及旧知识回顾`, `安技教育-1`, `控制线路布线与通电调试-6`} {
		if !strings.Contains(exampleMD, want) {
			t.Fatalf("example missing %q", want)
		}
	}
}

func TestConvertMarkdownStartsWithTypstDirective(t *testing.T) {
	output := convertMarkdown(exampleMD)
	if !strings.HasPrefix(output, "// jiaoan-jihua official template") && !strings.HasPrefix(output, "#set page(") {
		t.Fatalf("unexpected output prefix: %q", output[:min(40, len(output))])
	}
}

func TestParseFrontMatterKeepsSourceText(t *testing.T) {
	fm, _ := parseFrontMatter(exampleMD)
	assertEqual(t, fm.SchoolYear, "2025-2026")
	assertEqual(t, fm.Semester, "第一学期")
	assertEqual(t, fm.WeekRange, "第1 - 2周")
	assertEqual(t, fm.MajorName, "电气自动化技术")
	assertEqual(t, fm.CourseName, "电气设备控制线路安装与调试")
	assertEqual(t, fm.TeacherName, "张老师")
	assertEqual(t, fm.ClassName, "29WG电气3")
	assertEqual(t, fm.PreparedBy, "张老师")
	assertEqual(t, fm.FirstTeachingDay, "2025-09-01")
	assertEqual(t, fm.DailyHours, 8)
	assertEqual(t, fm.CalendarJSON, "presto/calendar.json")
}

func TestParseMarkdownTasksActivitiesAndContentRows(t *testing.T) {
	_, body := parseFrontMatter(exampleMD)
	plan := parseLearningPlanMarkdown(body)
	if len(plan.Tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(plan.Tasks))
	}
	assertEqual(t, plan.Tasks[0].Name, "CA6140卧式车床电气控制线路安装与调试")
	assertEqual(t, plan.Tasks[0].Activities[0].Name, "安技教育及旧知识回顾")
	assertEqual(t, plan.Tasks[0].Activities[0].Rows[0].Text, "安技教育")
	assertEqual(t, plan.Tasks[0].Activities[0].Rows[0].Hours, 1)
	found := false
	for _, activity := range plan.Tasks[0].Activities {
		for _, row := range activity.Rows {
			if row.Text == "控制线路布线与通电调试" && row.Hours == 6 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("missing expected content row")
	}
}

func TestParseMultipleTasksPreservesOrder(t *testing.T) {
	_, body := parseFrontMatter(exampleMD)
	plan := parseLearningPlanMarkdown(body)
	assertEqual(t, plan.Tasks[0].Name, "CA6140卧式车床电气控制线路安装与调试")
	assertEqual(t, plan.Tasks[1].Name, "X62W万能铣床电气控制线路安装与调试")
}

func TestParseContentRowStartingWithTemplateIsPreserved(t *testing.T) {
	plan := parseLearningPlanMarkdown("## 任务\n\n### 环节\n\ntemplate: PLC控制模板讲解-2\n")
	row := plan.Tasks[0].Activities[0].Rows[0]
	assertEqual(t, row.Text, "template: PLC控制模板讲解")
	assertEqual(t, row.Hours, 2)
}

func TestMissingFieldsRenderPlaceholders(t *testing.T) {
	output := convertMarkdown("---\n---\n")
	for _, want := range []string{missingCourseName, missingTaskName, missingActivityName, missingContent, "请输入授课教师"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing placeholder %q", want)
		}
	}
}

func TestContentWithoutHourDefaultsToTwoHours(t *testing.T) {
	output := convertMarkdown(withBody("## 任务\n\n### 环节\n\n安全教育\n"))
	for _, want := range []string{"安全教育", "2H"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q", want)
		}
	}
}

func TestLegacyTableInputShowsHelpfulHint(t *testing.T) {
	output := convertMarkdown(withBody("| 教学内容 | 学时 |\n| 安技教育 | 1 |\n"))
	for _, want := range []string{missingTaskName, legacyInputHint} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q", want)
		}
	}
}

func TestLoadExplicitCalendarJSONPath(t *testing.T) {
	path := writeTempCalendar(t, `[{"date":"2025-09-06","workday":true}]`)
	fm := sampleFrontMatter()
	fm.CalendarJSON = path
	days, warning := loadCalendar(fm)
	if warning != "" {
		t.Fatalf("unexpected warning: %s", warning)
	}
	plan := oneRowPlan("周末训练", 1)
	got := schedulePlan(plan, fm, days).Tasks[0].Activities[0].Rows[0]
	assertEqual(t, got.WeekdayDisplay, "6")
}

func TestDefaultCalendarPathIsPrestoCalendarJSON(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.MkdirAll(filepath.Join(dir, "presto"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "presto", "calendar.json"), []byte(`[{"date":"2025-09-06","workday":true}]`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	fm := sampleFrontMatter()
	fm.CalendarJSON = ""
	days, _ := loadCalendar(fm)
	got := schedulePlan(oneRowPlan("周末训练", 1), fm, days).Tasks[0].Activities[0].Rows[0]
	assertEqual(t, got.WeekdayDisplay, "6")
}

func TestMissingCalendarFallsBackToFirstTeachingDay(t *testing.T) {
	fm := sampleFrontMatter()
	fm.CalendarJSON = filepath.Join(t.TempDir(), "missing.json")
	days, _ := loadCalendar(fm)
	got := schedulePlan(oneRowPlan("首日", 1), fm, days).Tasks[0].Activities[0].Rows[0]
	assertEqual(t, got.WeekDisplay, "1")
	assertEqual(t, got.WeekdayDisplay, "1")
}

func TestBadCalendarJSONRendersWarning(t *testing.T) {
	path := writeTempCalendar(t, `{bad`)
	input := strings.Replace(exampleMD, `calendar_json: "presto/calendar.json"`, `calendar_json: "`+path+`"`, 1)
	output := convertMarkdown(input)
	if !strings.Contains(output, badCalendarWarning) {
		t.Fatalf("missing bad calendar warning")
	}
}

func TestScheduleConsumesWorkdaysOnly(t *testing.T) {
	fm := sampleFrontMatter()
	fm.DailyHours = 4
	days := []calendarDay{
		{Date: "2025-09-01", Workday: true},
		{Date: "2025-09-02", Workday: false},
		{Date: "2025-09-03", Workday: true},
	}
	got := schedulePlan(oneRowPlan("跨日", 8), fm, days).Tasks[0].Activities[0].Rows[0]
	assertEqual(t, got.WeekdayDisplay, "1 3")
}

func TestInsufficientExternalCalendarExtendsFromLastDate(t *testing.T) {
	fm := sampleFrontMatter()
	fm.DailyHours = 4
	days := []calendarDay{{Date: "2025-09-01", Workday: true}}
	got := schedulePlan(oneRowPlan("补足", 12), fm, days).Tasks[0].Activities[0].Rows[0]
	assertEqual(t, got.WeekdayDisplay, "1 2 3")
}

func TestSingleContentRowSpanningWeeksStaysOneRow(t *testing.T) {
	fm := sampleFrontMatter()
	fm.FirstTeachingDay = "2025-09-01"
	fm.DailyHours = 4
	days := []calendarDay{
		{Date: "2025-09-05", Workday: true},
		{Date: "2025-09-08", Workday: true},
		{Date: "2025-09-09", Workday: true},
	}
	got := schedulePlan(oneRowPlan("跨周综合训练", 12), fm, days).Tasks[0].Activities[0].Rows[0]
	assertEqual(t, got.HourDisplay, "12H")
	assertEqual(t, got.WeekDisplay, "1 2")
	assertEqual(t, got.WeekdayDisplay, "5 1 2")
	if strings.Contains(got.WeekDisplay, "、") || strings.Contains(got.WeekdayDisplay, "、") {
		t.Fatal("week and weekday columns should use spaces, not dunhao")
	}
}

func TestWeekendWorkdayUsesRealWeekday(t *testing.T) {
	fm := sampleFrontMatter()
	days := []calendarDay{{Date: "2025-09-06", Workday: true}}
	got := schedulePlan(oneRowPlan("周末", 1), fm, days).Tasks[0].Activities[0].Rows[0]
	assertEqual(t, got.WeekdayDisplay, "6")
}

func TestRenderUsesPrototypePageAndTableConstants(t *testing.T) {
	output := convertMarkdown(exampleMD)
	for _, want := range []string{`paper: "a4"`, `flipped: false`, `margin: (top: 2.54cm, bottom: 2.54cm, left: 2.8cm, right: 2.8cm)`, `columns: (3.15cm, 8.51cm, 1.12cm, 1.29cm, 1.27cm)`, `cell-pad = (x: 4.8pt, y: 4.8pt)`} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q", want)
		}
	}
}

func TestRenderOutputsTitleMetadataAndSignature(t *testing.T) {
	output := convertMarkdown(exampleMD)
	for _, want := range []string{`2025-2026学年第一学期第1 - 2周`, `工学一体化课程/基本技能课程授课进度计划表`, `专业名称：电气自动化技术`, `课程名称：电气设备控制线路安装与调试`, `授课教师：张老师`, `授课班级：29WG电气3`, `系主任：`, `教研室主任：`, `制表：张老师`} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q", want)
		}
	}
}

func TestSignaturePreparedByUsesTeacherName(t *testing.T) {
	input := strings.Replace(withBody("## 任务\n\n### 环节\n\n内容-1\n"), `teacher_name: "张老师"`, `teacher_name: "授课教师甲"`, 1)
	input = strings.Replace(input, `prepared_by: "张老师"`, `prepared_by: "制表人乙"`, 1)
	output := convertMarkdown(input)
	if !strings.Contains(output, "制表：授课教师甲") {
		t.Fatal("signature should use teacher_name for prepared-by display")
	}
	if strings.Contains(output, "制表：制表人乙") {
		t.Fatal("signature must not use prepared_by when teacher_name is present")
	}
}

func TestRenderOutputsFirstTaskAndBodyHeaders(t *testing.T) {
	output := convertMarkdown(exampleMD)
	for _, want := range []string{"学习任务1名称：", "教学内容", "周次", "星期", "学时"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q", want)
		}
	}
}

func TestRenderUsesColspanForTaskRows(t *testing.T) {
	output := convertMarkdown(exampleMD)
	if !strings.Contains(output, "task-th[学习任务1名称：]") || !strings.Contains(output, "task-th[CA6140卧式车床电气控制线路安装与调试]") {
		t.Fatal("missing bold task cells")
	}
	if !strings.Contains(output, "table.cell(colspan: 2, align: center + horizon, inset: cell-pad)[学时]") {
		t.Fatal("missing task hours colspan cell")
	}
}

func TestRenderUsesRowspanOnlyOnFirstActivityRow(t *testing.T) {
	output := convertMarkdown(exampleMD)
	want := "table.cell(rowspan: 3, align: center + horizon, inset: cell-pad)[学习环节1名称：安技教育及旧知识回顾]"
	if !strings.Contains(output, want) {
		t.Fatalf("missing activity rowspan cell %q", want)
	}
	if strings.Count(output, "学习环节1名称：安技教育及旧知识回顾") != 1 {
		t.Fatal("activity label should be emitted once and merged across content rows")
	}
}

func TestRenderMultipleTasksUseSeparatorWithoutRepeatingBodyHeader(t *testing.T) {
	output := convertMarkdown(exampleMD)
	if !strings.Contains(output, "学习任务2名称：") {
		t.Fatal("missing second task")
	}
	if strings.Count(output, "subth[教学内容]") != 1 {
		t.Fatal("body header repeated")
	}
}

func TestRenderContentRowsCarryScheduleCells(t *testing.T) {
	output := convertMarkdown(exampleMD)
	for _, want := range []string{"安技教育", "body-cell[1]", "CA6140卧式车床主电路识读", "body-cell[4]"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q", want)
		}
	}
}

func TestRenderEscapesYamlAndMarkdownContent(t *testing.T) {
	input := `---
school_year: "2025#2026"
semester: "第一]学期"
week_range: "第1\\2周"
major_name: "电气\"专业"
course_name: "#set page(paper: \"a0\")"
teacher_name: "张]老师"
class_name: "29\\班"
prepared_by: "制#表"
first_teaching_day: "2025-09-01"
daily_hours: 8
---

## 任务#一
### 环节]一
内容\一-1
`
	output := convertMarkdown(input)
	for _, want := range []string{`\#`, `\]`, `\\`} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing escaped fragment %q", want)
		}
	}
}

func TestBadCalendarWarningIsEscapedAndVisible(t *testing.T) {
	path := writeTempCalendar(t, `bad`)
	input := strings.Replace(exampleMD, `calendar_json: "presto/calendar.json"`, `calendar_json: "`+path+`"`, 1)
	output := convertMarkdown(input)
	if !strings.Contains(output, badCalendarWarning) {
		t.Fatal("warning not visible")
	}
}

func TestRendererDoesNotEmitRawInjection(t *testing.T) {
	input := withBody("## #set page(paper: \"a0\")\n\n### 环节\n\n#set page(paper: \"a0\")-1\n")
	output := convertMarkdown(input)
	if strings.Contains(output, "[#set page(paper: \"a0\")") {
		t.Fatal("raw injection was emitted")
	}
}

func sampleFrontMatter() frontMatter {
	return frontMatter{
		SchoolYear:       "2025-2026",
		Semester:         "第一学期",
		WeekRange:        "第1 - 2周",
		MajorName:        "电气自动化技术",
		CourseName:       "电气设备控制线路安装与调试",
		TeacherName:      "张老师",
		ClassName:        "29WG电气3",
		PreparedBy:       "张老师",
		FirstTeachingDay: "2025-09-01",
		DailyHours:       8,
	}
}

func oneRowPlan(text string, hours int) learningPlan {
	return learningPlan{Tasks: []learningTask{{
		Name: "任务",
		Activities: []learningActivity{{
			Name: "环节",
			Rows: []contentRow{{Text: text, Hours: hours, HadExplicitHours: true}},
		}},
	}}}
}

func withBody(body string) string {
	return `---
school_year: "2025-2026"
semester: "第一学期"
week_range: "第1 - 2周"
major_name: "电气自动化技术"
course_name: "电气设备控制线路安装与调试"
teacher_name: "张老师"
class_name: "29WG电气3"
prepared_by: "张老师"
first_teaching_day: "2025-09-01"
daily_hours: 8
---

` + body
}

func writeTempCalendar(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "calendar.json")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
