package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Presto-io/presto-official-templates/internal/cli"
	"github.com/Presto-io/presto-official-templates/internal/typst"
	"gopkg.in/yaml.v3"
)

//go:embed manifest.json
var manifestJSON string

//go:embed example.md
var exampleMD string

//go:embed presto/calendar.json
var embeddedCalendarJSON string

func main() {
	cli.Run(manifestJSON, exampleMD, func(input string) string {
		return convertMarkdown(input)
	}, func(input string) cli.OutputInfo {
		return outputInfo(input)
	})
}

type frontMatter struct {
	SchoolYear       string `yaml:"school_year"`
	Semester         string `yaml:"semester"`
	WeekRange        string `yaml:"week_range"`
	MajorName        string `yaml:"major_name"`
	CourseName       string `yaml:"course_name"`
	TeacherName      string `yaml:"teacher_name"`
	ClassName        string `yaml:"class_name"`
	PreparedBy       string `yaml:"prepared_by"`
	FirstTeachingDay string `yaml:"first_teaching_day"`
	DailyHours       int    `yaml:"daily_hours"`
	CalendarJSON     string `yaml:"calendar_json"`
}

type learningPlan struct {
	Tasks []learningTask
	Hint  string
}

type learningTask struct {
	Name       string
	Activities []learningActivity
}

type learningActivity struct {
	Name string
	Rows []contentRow
}

type contentRow struct {
	Text             string
	Hours            int
	HadExplicitHours bool
}

type calendarDay struct {
	Date    string `json:"date"`
	Workday bool   `json:"workday"`
}

type scheduledPlan struct {
	Tasks []scheduledTask
	Hint  string
}

type scheduledTask struct {
	Name       string
	TotalHours int
	Activities []scheduledActivity
}

type scheduledActivity struct {
	Name string
	Rows []scheduledRow
}

type scheduledRow struct {
	Text           string
	Hours          int
	WeekDisplay    string
	WeekdayDisplay string
	HourDisplay    string
}

var contentHoursPattern = regexp.MustCompile(`^(.*)-([0-9]+)\s*$`)

const missingCourseName = "请输入课程名称"
const missingTaskName = "请输入学习任务名称"
const missingActivityName = "请输入学习环节名称"
const missingContent = "请输入教学内容-课时"
const legacyInputHint = "请使用 ## 学习任务 和 ### 学习环节 输入授课计划"
const badCalendarWarning = "日历文件解析失败，已使用默认日历"

func convertMarkdown(input string) string {
	fm, body := parseFrontMatter(input)
	plan := parseLearningPlanMarkdown(body)
	plan = normalizeLearningPlan(plan, strings.TrimSpace(body) != "")
	days, warning := loadCalendar(fm)
	fm = normalizeFrontMatter(fm, days)
	scheduled := schedulePlan(plan, fm, days)
	fm = inferWeekRangeFromSchedule(fm, scheduled)
	return renderTypst(fm, scheduled, warning)
}

func outputInfo(input string) cli.OutputInfo {
	fm, _ := parseFrontMatter(input)
	days, _ := loadCalendar(fm)
	fm = normalizeFrontMatter(fm, days)
	return outputInfoFromFrontMatter(fm)
}

func outputInfoFromFrontMatter(fm frontMatter) cli.OutputInfo {
	title := "授课进度计划表"
	if strings.TrimSpace(fm.CourseName) != "" {
		title += " " + fm.CourseName
	}
	base := title
	if strings.TrimSpace(fm.SchoolYear) != "" {
		base += " " + fm.SchoolYear
	}
	if strings.TrimSpace(fm.Semester) != "" {
		base += " " + fm.Semester
	}
	authors := []string{}
	if strings.TrimSpace(fm.TeacherName) != "" {
		authors = append(authors, fm.TeacherName)
	} else if strings.TrimSpace(fm.PreparedBy) != "" {
		authors = append(authors, fm.PreparedBy)
	}
	return cli.OutputInfo{
		SchemaVersion:  1,
		OutputBaseName: cleanFilenameBase(base),
		PreviewTitle:   title,
		Document: cli.DocumentInfo{
			Title:       title,
			Authors:     authors,
			Date:        fm.FirstTeachingDay,
			Keywords:    []string{"授课计划", "教学计划", "工学一体化"},
			Subject:     "授课进度计划表",
			Description: "Presto 授课进度计划表模板生成的 PDF",
			Language:    "zh-CN",
		},
		TemplateData: map[string]interface{}{
			"schoolYear": fm.SchoolYear,
			"semester":   fm.Semester,
			"courseName": fm.CourseName,
			"majorName":  fm.MajorName,
			"className":  fm.ClassName,
		},
	}
}

func cleanFilenameBase(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_")
	value = strings.TrimSpace(replacer.Replace(value))
	if value == "" {
		return "output"
	}
	return value
}

func parseFrontMatter(input string) (frontMatter, string) {
	var fm frontMatter
	input = strings.ReplaceAll(input, "\r\n", "\n")
	if !strings.HasPrefix(input, "---") {
		return fm, input
	}
	rest := input[3:]
	if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	}
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fm, input
	}
	yamlBlock := rest[:idx]
	body := rest[idx+4:]
	if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return frontMatter{}, body
	}
	return fm, body
}

func parseLearningPlanMarkdown(body string) learningPlan {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	plan := learningPlan{}
	var currentTask *learningTask
	var currentActivity *learningActivity

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "## "):
			plan.Tasks = append(plan.Tasks, learningTask{Name: strings.TrimSpace(strings.TrimPrefix(line, "## "))})
			currentTask = &plan.Tasks[len(plan.Tasks)-1]
			currentActivity = nil
		case strings.HasPrefix(line, "### "):
			if currentTask == nil {
				plan.Tasks = append(plan.Tasks, learningTask{Name: missingTaskName})
				currentTask = &plan.Tasks[len(plan.Tasks)-1]
			}
			currentTask.Activities = append(currentTask.Activities, learningActivity{Name: strings.TrimSpace(strings.TrimPrefix(line, "### "))})
			currentActivity = &currentTask.Activities[len(currentTask.Activities)-1]
		default:
			if currentActivity != nil {
				text, hours, explicit := parseContentLine(line)
				currentActivity.Rows = append(currentActivity.Rows, contentRow{Text: text, Hours: hours, HadExplicitHours: explicit})
			}
		}
	}
	return plan
}

func parseContentLine(line string) (text string, hours int, hadExplicitHours bool) {
	line = strings.TrimSpace(line)
	if match := contentHoursPattern.FindStringSubmatch(line); match != nil {
		parsed, err := strconv.Atoi(match[2])
		if err == nil && parsed > 0 {
			return strings.TrimSpace(match[1]), parsed, true
		}
	}
	return line, 2, false
}

func normalizeLearningPlan(plan learningPlan, hadBody bool) learningPlan {
	if len(plan.Tasks) == 0 {
		plan.Tasks = []learningTask{{
			Name: missingTaskName,
			Activities: []learningActivity{{
				Name: missingActivityName,
				Rows: []contentRow{{Text: missingContent, Hours: 2}},
			}},
		}}
		if hadBody {
			plan.Hint = legacyInputHint
		}
		return plan
	}
	for taskIndex := range plan.Tasks {
		task := &plan.Tasks[taskIndex]
		if strings.TrimSpace(task.Name) == "" {
			task.Name = missingTaskName
		}
		if len(task.Activities) == 0 {
			task.Activities = []learningActivity{{
				Name: missingActivityName,
				Rows: []contentRow{{Text: missingContent, Hours: 2}},
			}}
			continue
		}
		for activityIndex := range task.Activities {
			activity := &task.Activities[activityIndex]
			if strings.TrimSpace(activity.Name) == "" {
				activity.Name = missingActivityName
			}
			if len(activity.Rows) == 0 {
				activity.Rows = []contentRow{{Text: missingContent, Hours: 2}}
			}
		}
	}
	return plan
}

func loadCalendar(fm frontMatter) ([]calendarDay, string) {
	path := strings.TrimSpace(fm.CalendarJSON)
	if path == "" {
		days, err := parseCalendarJSON([]byte(embeddedCalendarJSON))
		if err == nil && validCalendarDays(days) {
			return days, ""
		}
		return generateDefaultCalendar(fm.FirstTeachingDay), badCalendarWarning
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fallbackCalendar(fm), ""
	}
	days, err := parseCalendarJSON(raw)
	if err != nil || !validCalendarDays(days) {
		return fallbackCalendar(fm), badCalendarWarning
	}
	return days, ""
}

func fallbackCalendar(fm frontMatter) []calendarDay {
	days, err := parseCalendarJSON([]byte(embeddedCalendarJSON))
	if err == nil && validCalendarDays(days) {
		return days
	}
	return generateDefaultCalendar(fm.FirstTeachingDay)
}

func parseCalendarJSON(raw []byte) ([]calendarDay, error) {
	var workdays []string
	if err := json.Unmarshal(raw, &workdays); err == nil {
		days := make([]calendarDay, 0, len(workdays))
		for _, date := range workdays {
			days = append(days, calendarDay{Date: date, Workday: true})
		}
		return days, nil
	}
	var days []calendarDay
	if err := json.Unmarshal(raw, &days); err == nil {
		return days, nil
	}
	var wrapped struct {
		Days []calendarDay `json:"days"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Days, nil
}

func validCalendarDays(days []calendarDay) bool {
	if len(days) == 0 {
		return false
	}
	for _, day := range days {
		if _, err := parseISODate(day.Date); err != nil {
			return false
		}
	}
	return true
}

func generateDefaultCalendar(firstTeachingDay string) []calendarDay {
	start, err := parseISODate(firstTeachingDay)
	if err != nil {
		start = time.Now()
	}
	days := make([]calendarDay, 0, 260)
	for len(days) < 260 {
		weekday := start.Weekday()
		workday := weekday >= time.Monday && weekday <= time.Friday
		days = append(days, calendarDay{Date: start.Format("2006-01-02"), Workday: workday})
		start = start.AddDate(0, 0, 1)
	}
	return days
}

func normalizeFrontMatter(fm frontMatter, days []calendarDay) frontMatter {
	if fm.DailyHours <= 0 {
		fm.DailyHours = 8
	}
	if strings.TrimSpace(fm.PreparedBy) == "" {
		fm.PreparedBy = fm.TeacherName
	}
	if len(days) > 0 {
		if start, err := parseISODate(days[0].Date); err == nil {
			if strings.TrimSpace(fm.SchoolYear) == "" {
				fm.SchoolYear = inferSchoolYear(start)
			}
			if strings.TrimSpace(fm.Semester) == "" {
				fm.Semester = inferSemester(start)
			}
		}
	}
	return fm
}

func inferSchoolYear(calendarStart time.Time) string {
	year := calendarStart.Year()
	if calendarStart.Month() <= time.June {
		return fmt.Sprintf("%d-%d", year-1, year)
	}
	return fmt.Sprintf("%d-%d", year, year+1)
}

func inferSemester(calendarStart time.Time) string {
	if calendarStart.Month() <= time.June {
		return "第二学期"
	}
	return "第一学期"
}

func inferWeekRangeFromSchedule(fm frontMatter, plan scheduledPlan) frontMatter {
	if strings.TrimSpace(fm.WeekRange) != "" {
		return fm
	}
	minWeek := 0
	maxWeek := 0
	for _, task := range plan.Tasks {
		for _, activity := range task.Activities {
			for _, row := range activity.Rows {
				for _, part := range strings.Fields(row.WeekDisplay) {
					week, err := strconv.Atoi(part)
					if err != nil {
						continue
					}
					if minWeek == 0 || week < minWeek {
						minWeek = week
					}
					if week > maxWeek {
						maxWeek = week
					}
				}
			}
		}
	}
	if minWeek == 0 {
		return fm
	}
	if minWeek == maxWeek {
		fm.WeekRange = fmt.Sprintf("第%d周", minWeek)
	} else {
		fm.WeekRange = fmt.Sprintf("第%d - %d周", minWeek, maxWeek)
	}
	return fm
}

func schedulePlan(plan learningPlan, fm frontMatter, days []calendarDay) scheduledPlan {
	dailyHours := fm.DailyHours
	if dailyHours <= 0 {
		dailyHours = 8
	}
	if len(days) == 0 {
		days = generateDefaultCalendar(fm.FirstTeachingDay)
	}
	anchor, err := parseISODate(days[0].Date)
	if err != nil {
		anchor, err = parseISODate(fm.FirstTeachingDay)
		if err != nil {
			anchor = time.Now()
		}
	}
	weekOneMonday := mondayOf(anchor)

	result := scheduledPlan{Hint: plan.Hint}
	days, cursor := alignCalendarToFirstTeachingDay(days, fm.FirstTeachingDay)
	remainingToday := 0
	for _, task := range plan.Tasks {
		scheduledTask := scheduledTask{Name: task.Name}
		for _, activity := range task.Activities {
			scheduledActivity := scheduledActivity{Name: activity.Name}
			for _, row := range activity.Rows {
				scheduledRow, nextCursor, nextRemaining := scheduleRow(row, days, cursor, remainingToday, dailyHours, weekOneMonday)
				cursor = nextCursor
				remainingToday = nextRemaining
				scheduledActivity.Rows = append(scheduledActivity.Rows, scheduledRow)
				scheduledTask.TotalHours += row.Hours
			}
			scheduledTask.Activities = append(scheduledTask.Activities, scheduledActivity)
		}
		result.Tasks = append(result.Tasks, scheduledTask)
	}
	return result
}

func alignCalendarToFirstTeachingDay(days []calendarDay, firstTeachingDay string) ([]calendarDay, int) {
	start, err := parseISODate(firstTeachingDay)
	if err != nil || len(days) == 0 {
		return days, 0
	}
	for len(days) > 0 {
		last, err := parseISODate(days[len(days)-1].Date)
		if err != nil || !dateOnly(last).Before(dateOnly(start)) {
			break
		}
		last = last.AddDate(0, 0, 1)
		workday := last.Weekday() >= time.Monday && last.Weekday() <= time.Friday
		days = append(days, calendarDay{Date: last.Format("2006-01-02"), Workday: workday})
	}
	for i, day := range days {
		parsed, err := parseISODate(day.Date)
		if err == nil && !dateOnly(parsed).Before(dateOnly(start)) {
			return days, i
		}
	}
	return days, len(days)
}

func scheduleRow(row contentRow, days []calendarDay, cursor int, remainingToday int, dailyHours int, weekOneMonday time.Time) (scheduledRow, int, int) {
	hoursLeft := row.Hours
	weeks := orderedSet{}
	weekdays := orderedSet{}

	for hoursLeft > 0 {
		for cursor >= len(days) || !days[cursor].Workday {
			if cursor >= len(days) {
				days = append(days, nextGeneratedWorkdays(days[len(days)-1].Date, 1)...)
			}
			if cursor < len(days) && !days[cursor].Workday {
				cursor++
			}
		}
		if remainingToday <= 0 {
			remainingToday = dailyHours
		}
		dayDate, err := parseISODate(days[cursor].Date)
		if err != nil {
			cursor++
			remainingToday = 0
			continue
		}
		weeks.Add(strconv.Itoa(weekNumber(dayDate, weekOneMonday)))
		weekdays.Add(strconv.Itoa(weekdayNumber(dayDate)))

		consume := hoursLeft
		if consume > remainingToday {
			consume = remainingToday
		}
		hoursLeft -= consume
		remainingToday -= consume
		if remainingToday == 0 {
			cursor++
		}
	}

	return scheduledRow{
		Text:           row.Text,
		Hours:          row.Hours,
		WeekDisplay:    weeks.Join(" "),
		WeekdayDisplay: weekdays.Join(" "),
		HourDisplay:    fmt.Sprintf("%dH", row.Hours),
	}, cursor, remainingToday
}

func nextGeneratedWorkdays(lastDate string, minWorkdays int) []calendarDay {
	last, err := parseISODate(lastDate)
	if err != nil {
		last = time.Now()
	}
	var days []calendarDay
	workdays := 0
	for workdays < minWorkdays {
		last = last.AddDate(0, 0, 1)
		workday := last.Weekday() >= time.Monday && last.Weekday() <= time.Friday
		if workday {
			workdays++
		}
		days = append(days, calendarDay{Date: last.Format("2006-01-02"), Workday: workday})
	}
	return days
}

type orderedSet struct {
	values []string
	seen   map[string]bool
}

func (s *orderedSet) Add(value string) {
	if s.seen == nil {
		s.seen = map[string]bool{}
	}
	if !s.seen[value] {
		s.seen[value] = true
		s.values = append(s.values, value)
	}
}

func (s orderedSet) Join(separator string) string {
	return strings.Join(s.values, separator)
}

func parseISODate(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", strings.TrimSpace(value), time.Local)
}

func mondayOf(t time.Time) time.Time {
	offset := int(t.Weekday() - time.Monday)
	if offset < 0 {
		offset = 6
	}
	return dateOnly(t).AddDate(0, 0, -offset)
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func weekNumber(day time.Time, weekOneMonday time.Time) int {
	diff := int(dateOnly(day).Sub(weekOneMonday).Hours() / 24)
	return diff/7 + 1
}

func weekdayNumber(day time.Time) int {
	if day.Weekday() == time.Sunday {
		return 7
	}
	return int(day.Weekday())
}

const rendererPreamble = `// jiaoan-jihua official template
#import "@preview/cuti:0.2.1": show-cn-fakebold
#show: show-cn-fakebold

#set page(
  paper: "a4",
  flipped: false,
  margin: (top: 2.54cm, bottom: 2.54cm, left: 2.8cm, right: 2.8cm)
)

#set text(
  lang: "zh",
  font: "STSong",
  size: 10.5pt,
  hyphenate: false,
)

#set par(justify: true, leading: 0.52em)

#let cell-pad = (x: 4.8pt, y: 4.8pt)
#let task-th(body) = table.cell(align: center + horizon, inset: cell-pad)[#text(weight: 700)[#body]]
#let th(body) = table.cell(align: center + horizon, inset: cell-pad)[#body]
#let subth(body) = table.cell(align: center + horizon, inset: cell-pad)[#body]
#let body-cell(body) = table.cell(align: center + horizon, inset: cell-pad)[#body]
#let content-cell(body) = table.cell(align: left + horizon, inset: cell-pad)[#body]
`

func renderTypst(fm frontMatter, plan scheduledPlan, warning string) string {
	var b strings.Builder
	b.WriteString(rendererPreamble)
	b.WriteString("\n")
	info := outputInfoFromFrontMatter(fm)
	author := "Presto"
	if len(info.Document.Authors) > 0 {
		author = info.Document.Authors[0]
	}
	b.WriteString(fmt.Sprintf("#set document(\n  title: \"%s\",\n  author: \"%s\",\n  keywords: \"授课计划, 教学计划, 工学一体化\",\n)\n\n", typst.EscapeString(info.Document.Title), typst.EscapeString(author)))
	b.WriteString("#align(center)[#text(size: 14pt, weight: \"bold\")[")
	b.WriteString(escape(placeholder(fm.SchoolYear, "请输入学年")))
	b.WriteString("学年")
	b.WriteString(escape(placeholder(fm.Semester, "请输入学期")))
	b.WriteString(escape(placeholder(fm.WeekRange, "请输入周次范围")))
	b.WriteString("]]\n")
	b.WriteString("#v(0.45em)\n")
	b.WriteString("#align(center)[#text(size: 14pt, weight: \"bold\")[工学一体化课程/基本技能课程授课进度计划表]]\n")
	b.WriteString("#v(0.72em)\n\n")
	writeMetadata(&b, fm)
	if warning != "" {
		b.WriteString("#block(below: 8pt)[")
		b.WriteString(escape(warning))
		b.WriteString("]\n")
	}
	if plan.Hint != "" {
		b.WriteString("// ")
		b.WriteString(plan.Hint)
		b.WriteString("\n")
		b.WriteString("#block(below: 8pt)[")
		b.WriteString(escape(plan.Hint))
		b.WriteString("]\n")
	}
	writePlanTable(&b, plan)
	writeSignature(&b, fm)
	return b.String()
}

func writeMetadata(b *strings.Builder, fm frontMatter) {
	items := []string{
		"专业名称：" + placeholder(fm.MajorName, "请输入专业名称"),
		"课程名称：" + placeholder(fm.CourseName, missingCourseName),
		"授课教师：" + placeholder(fm.TeacherName, "请输入授课教师"),
		"授课班级：" + placeholder(fm.ClassName, "请输入授课班级"),
	}
	b.WriteString("#text(size: 10.5pt)[\n")
	b.WriteString("  #grid(columns: (1fr, 1fr), row-gutter: 0.75em,\n")
	for _, item := range items {
		b.WriteString("    [")
		b.WriteString(escape(item))
		b.WriteString("],\n")
	}
	b.WriteString("  )\n")
	b.WriteString("]\n\n")
	b.WriteString("#v(0.9em)\n\n")
}

func writePlanTable(b *strings.Builder, plan scheduledPlan) {
	b.WriteString(`#align(center)[
  #table(
    columns: (3.15cm, 8.51cm, 1.12cm, 1.29cm, 1.27cm),
    stroke: 0.5pt,
    align: center + horizon,
`)
	for taskIndex, task := range plan.Tasks {
		b.WriteString(fmt.Sprintf("    task-th[%s],\n    task-th[%s],\n    table.cell(colspan: 2, align: center + horizon, inset: cell-pad)[学时],\n    th[%dH],\n\n",
			escape(fmt.Sprintf("学习任务%d名称：", taskIndex+1)),
			escape(placeholder(task.Name, missingTaskName)),
			task.TotalHours,
		))
		if taskIndex == 0 {
			b.WriteString("    subth[],\n    subth[教学内容],\n    subth[周次],\n    subth[星期],\n    subth[学时],\n\n")
		}
		for activityIndex, activity := range task.Activities {
			activityLabel := escape(fmt.Sprintf("学习环节%d名称：%s", activityIndex+1, placeholder(activity.Name, missingActivityName)))
			for rowIndex, row := range activity.Rows {
				if rowIndex == 0 {
					b.WriteString(fmt.Sprintf("    table.cell(rowspan: %d, align: center + horizon, inset: cell-pad)[%s],\n", len(activity.Rows), activityLabel))
				}
				b.WriteString(fmt.Sprintf("    content-cell[%s],\n    body-cell[%s],\n    body-cell[%s],\n    body-cell[%s],\n\n",
					escape(placeholder(row.Text, missingContent)),
					escape(row.WeekDisplay),
					escape(row.WeekdayDisplay),
					escape(strconv.Itoa(row.Hours)),
				))
			}
		}
	}
	b.WriteString("  )\n]\n\n")
}

func writeSignature(b *strings.Builder, fm frontMatter) {
	b.WriteString("#v(1.1em)\n")
	b.WriteString("#grid(columns: (1fr, 1fr, 1fr),\n")
	labels := []string{"系主任：", "教研室主任：", "制表：" + placeholder(fm.TeacherName, "请输入授课教师")}
	for _, label := range labels {
		b.WriteString("  [#align(center)[")
		b.WriteString(escape(label))
		b.WriteString("]],\n")
	}
	b.WriteString(")\n")
}

func placeholder(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func escape(value string) string {
	return typst.EscapeContent(value)
}
