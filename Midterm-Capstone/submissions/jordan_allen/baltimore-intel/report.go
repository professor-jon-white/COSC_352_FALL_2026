package main

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Section is the self-contained answer to one research question.
type Section struct {
	ID          string
	Stakeholder string
	Title       string
	Question    string
	Findings    []string // key takeaways, one sentence each
	Body        string   // pre-rendered tables and ASCII charts
}

// Report is the complete, renderer-independent output of a run.
type Report struct {
	Title     string
	Generated time.Time
	Meta      []string
	Sections  []Section
	Notes     []string
}

// ReportBuilder assembles a Report step by step (Builder pattern), keeping the
// analysis code free of any formatting concerns.
type ReportBuilder struct {
	r Report
}

// NewReportBuilder starts a report with the given title.
func NewReportBuilder(title string) *ReportBuilder {
	return &ReportBuilder{r: Report{Title: title, Generated: time.Now()}}
}

// Meta appends a line to the data-summary header.
func (b *ReportBuilder) Meta(line string) *ReportBuilder {
	b.r.Meta = append(b.r.Meta, line)
	return b
}

// Section appends a finished analysis section.
func (b *ReportBuilder) Section(s Section) *ReportBuilder {
	b.r.Sections = append(b.r.Sections, s)
	return b
}

// Note appends a line to the closing notes/limitations block.
func (b *ReportBuilder) Note(line string) *ReportBuilder {
	b.r.Notes = append(b.r.Notes, line)
	return b
}

// Build returns the finished report.
func (b *ReportBuilder) Build() *Report {
	r := b.r
	return &r
}

// Renderer turns a Report into bytes (Strategy pattern): the terminal view and
// the Markdown file are two implementations of the same interface.
type Renderer interface {
	Render(w io.Writer, r *Report) error
}

// TextRenderer produces the terminal report.
type TextRenderer struct{}

// Render implements Renderer.
func (TextRenderer) Render(w io.Writer, r *Report) error {
	const heavy = "================================================================================"
	const light = "--------------------------------------------------------------------------------"
	var sb strings.Builder

	sb.WriteString(heavy + "\n")
	sb.WriteString(" " + r.Title + "\n")
	sb.WriteString(" Generated " + r.Generated.Format("2006-01-02 15:04:05 MST") + "\n")
	sb.WriteString(heavy + "\n")
	for _, m := range r.Meta {
		sb.WriteString(" " + m + "\n")
	}
	for _, s := range r.Sections {
		sb.WriteString("\n" + light + "\n")
		fmt.Fprintf(&sb, " %s | For: %s\n", s.ID, s.Stakeholder)
		sb.WriteString(" " + s.Title + "\n")
		sb.WriteString(" Question: " + s.Question + "\n")
		sb.WriteString(light + "\n")
		if len(s.Findings) > 0 {
			sb.WriteString(" Key findings:\n")
			for _, f := range s.Findings {
				sb.WriteString("  * " + f + "\n")
			}
			sb.WriteString("\n")
		}
		sb.WriteString(s.Body)
	}
	if len(r.Notes) > 0 {
		sb.WriteString("\n" + light + "\n Notes and limitations\n" + light + "\n")
		for _, n := range r.Notes {
			sb.WriteString("  - " + n + "\n")
		}
	}
	sb.WriteString(heavy + "\n")
	_, err := io.WriteString(w, sb.String())
	return err
}

// MarkdownRenderer produces a shareable Markdown file; charts go in code fences.
type MarkdownRenderer struct{}

// Render implements Renderer.
func (MarkdownRenderer) Render(w io.Writer, r *Report) error {
	var sb strings.Builder
	sb.WriteString("# " + r.Title + "\n\n")
	sb.WriteString("_Generated " + r.Generated.Format("2006-01-02 15:04:05 MST") + "_\n\n")
	if len(r.Meta) > 0 {
		sb.WriteString("```text\n")
		for _, m := range r.Meta {
			sb.WriteString(m + "\n")
		}
		sb.WriteString("```\n")
	}
	for _, s := range r.Sections {
		fmt.Fprintf(&sb, "\n## %s: %s\n\n", s.ID, s.Title)
		fmt.Fprintf(&sb, "**For:** %s\n\n> %s\n\n", s.Stakeholder, s.Question)
		if len(s.Findings) > 0 {
			sb.WriteString("**Key findings**\n\n")
			for _, f := range s.Findings {
				sb.WriteString("- " + f + "\n")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("```text\n" + s.Body + "```\n")
	}
	if len(r.Notes) > 0 {
		sb.WriteString("\n## Notes and limitations\n\n")
		for _, n := range r.Notes {
			sb.WriteString("- " + n + "\n")
		}
	}
	_, err := io.WriteString(w, sb.String())
	return err
}
