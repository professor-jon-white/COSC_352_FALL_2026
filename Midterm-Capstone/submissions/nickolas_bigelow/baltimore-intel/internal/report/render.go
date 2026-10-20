package report

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// NewRenderer is the factory for output strategies.
func NewRenderer(name string, plainASCII bool) (Renderer, error) {
	switch name {
	case "terminal", "term", "text":
		cs := Unicode
		if plainASCII {
			cs = ASCII
		}
		return &Terminal{CS: cs, Plain: plainASCII, BarWidth: 40}, nil
	case "md", "markdown":
		return &Markdown{}, nil
	case "html":
		return &HTML{}, nil
	}
	return nil, fmt.Errorf("unknown format %q (terminal, md, html)", name)
}

// ---- Terminal ---------------------------------------------------------------

type Terminal struct {
	CS       Charset
	Plain    bool
	BarWidth int
}

func (t *Terminal) Name() string { return "terminal" }

var asciiBox = strings.NewReplacer("─", "-", "│", "|", "┌", "+", "┐", "+", "└", "+", "┘", "+",
	"├", "+", "┤", "+", "┬", "+", "┴", "+", "┼", "+", "═", "=", "…", "~", "≈", "~", "•", "*", "▸", ">", "µ", "u")

func (t *Terminal) Render(w io.Writer, r *Report) error {
	bw := bufio.NewWriter(w)
	p := func(format string, a ...any) {
		s := fmt.Sprintf(format, a...)
		if t.Plain {
			s = asciiBox.Replace(s)
		}
		bw.WriteString(s)
	}
	rule := strings.Repeat("═", 78)
	p("\n%s\n  %s\n  Generated %s\n%s\n", rule, strings.ToUpper(r.Title), r.Generated.Format("Mon Jan 2 2006 15:04 MST"), rule)
	if r.DataNote != "" {
		p("  %s\n", r.DataNote)
	}
	for _, kv := range r.Inputs {
		p("  %-26s %s\n", kv[0]+":", kv[1])
	}
	for i, s := range r.Sections {
		p("\n%s\n", strings.Repeat("─", 78))
		p("  Q%d  %s\n", i+1, s.Question)
		p("      Stakeholder: %s   |   Data: %s\n", s.Stakeholder, strings.Join(s.Datasets, " + "))
		p("%s\n", strings.Repeat("─", 78))
		p("\n  ▸ FINDING: %s\n", wrap(s.Finding, 74, "             "))
		for _, b := range s.Blocks {
			if title := b.blockTitle(); title != "" {
				p("\n  %s\n", title)
			}
			var lines []string
			switch v := b.(type) {
			case *Table:
				lines = TableLines(v)
			case *Bars:
				lines = BarLines(v, t.CS, t.BarWidth)
			case *Heatmap:
				lines = HeatLines(v, t.CS)
			case *Scatter:
				lines = ScatterLines(v, t.CS, 60, 16)
			case *KeyValues:
				for _, kv := range v.Pairs {
					lines = append(lines, fmt.Sprintf("  %-38s %s", kv[0], kv[1]))
				}
			}
			for _, l := range lines {
				p("%s\n", l)
			}
		}
		if len(s.Actions) > 0 {
			p("\n  Recommended actions:\n")
			for _, a := range s.Actions {
				p("   • %s\n", wrap(a, 72, "     "))
			}
		}
		for _, n := range s.Notes {
			p("  note: %s\n", wrap(n, 70, "        "))
		}
	}
	if len(r.Runtime) > 0 {
		p("\n%s\n  RUNTIME & RESOURCES\n", strings.Repeat("─", 78))
		for _, kv := range r.Runtime {
			p("  %-26s %s\n", kv[0]+":", kv[1])
		}
	}
	p("%s\n", rule)
	return bw.Flush()
}

func wrap(s string, w int, indent string) string {
	words := strings.Fields(s)
	var b strings.Builder
	n := 0
	for _, word := range words {
		if n > 0 && n+1+width(word) > w {
			b.WriteString("\n" + indent)
			n = 0
		} else if n > 0 {
			b.WriteByte(' ')
			n++
		}
		b.WriteString(word)
		n += width(word)
	}
	return b.String()
}

// ---- Markdown ---------------------------------------------------------------

type Markdown struct{}

func (Markdown) Name() string { return "md" }

func (Markdown) Render(w io.Writer, r *Report) error {
	bw := bufio.NewWriter(w)
	p := func(f string, a ...any) { fmt.Fprintf(bw, f, a...) }
	p("# %s\n\n_Generated %s_\n\n", r.Title, r.Generated.Format("January 2, 2006 15:04 MST"))
	if r.DataNote != "" {
		p("> %s\n\n", r.DataNote)
	}
	p("| Input | Value |\n|---|---|\n")
	for _, kv := range r.Inputs {
		p("| %s | %s |\n", kv[0], kv[1])
	}
	p("\n## Contents\n\n")
	for i, s := range r.Sections {
		p("%d. [%s](#q%d) — *%s*\n", i+1, s.Question, i+1, s.Stakeholder)
	}
	for i, s := range r.Sections {
		p("\n---\n\n<a id=\"q%d\"></a>\n## Q%d. %s\n\n**Stakeholder:** %s  \n**Datasets joined:** %s\n\n**Finding.** %s\n",
			i+1, i+1, s.Question, s.Stakeholder, strings.Join(s.Datasets, " + "), s.Finding)
		for _, b := range s.Blocks {
			if t := b.blockTitle(); t != "" {
				p("\n**%s**\n\n", t)
			}
			switch v := b.(type) {
			case *Table:
				p("| %s |\n|", strings.Join(v.Headers, " | "))
				for i := range v.Headers {
					if i < len(v.Right) && v.Right[i] {
						p("---:|")
					} else {
						p("---|")
					}
				}
				p("\n")
				for _, row := range v.Rows {
					esc := make([]string, len(row))
					for i, c := range row {
						esc[i] = strings.ReplaceAll(c, "|", "\\|")
					}
					p("| %s |\n", strings.Join(esc, " | "))
				}
			case *Bars:
				p("```\n%s\n```\n", strings.Join(BarLines(v, Unicode, 40), "\n"))
			case *Heatmap:
				p("```\n%s\n```\n", strings.Join(HeatLines(v, Unicode), "\n"))
			case *Scatter:
				p("```\n%s\n```\n", strings.Join(ScatterLines(v, Unicode, 60, 16), "\n"))
			case *KeyValues:
				p("| Measure | Value |\n|---|---:|\n")
				for _, kv := range v.Pairs {
					p("| %s | %s |\n", kv[0], kv[1])
				}
			}
		}
		if len(s.Actions) > 0 {
			p("\n**Recommended actions**\n\n")
			for _, a := range s.Actions {
				p("- %s\n", a)
			}
		}
		if len(s.Notes) > 0 {
			p("\n")
			for _, n := range s.Notes {
				p("> _Note:_ %s\n>\n", n)
			}
		}
	}
	if len(r.Runtime) > 0 {
		p("\n---\n\n## Runtime and resources\n\n| Measure | Value |\n|---|---|\n")
		for _, kv := range r.Runtime {
			p("| %s | %s |\n", kv[0], kv[1])
		}
	}
	return bw.Flush()
}

// ---- HTML -------------------------------------------------------------------

type HTML struct{}

func (HTML) Name() string { return "html" }

type htmlBar struct {
	Label, Value string
	Pct          float64
}

type htmlHeat struct {
	Cols []string
	Rows []struct {
		Label string
		Cells []struct {
			Alpha float64
			Value string
		}
	}
}

func (HTML) Render(w io.Writer, r *Report) error {
	funcs := template.FuncMap{
		"inc": func(i int) int { return i + 1 },
		"kind": func(b Block) string {
			switch b.(type) {
			case *Table:
				return "table"
			case *Bars:
				return "bars"
			case *Heatmap:
				return "heat"
			case *Scatter:
				return "scatter"
			case *KeyValues:
				return "kv"
			}
			return ""
		},
		"table": func(b Block) *Table { return b.(*Table) },
		"kv":    func(b Block) *KeyValues { return b.(*KeyValues) },
		"right": func(t *Table, i int) bool { return i < len(t.Right) && t.Right[i] },
		"bars": func(b Block) []htmlBar {
			v := b.(*Bars)
			maxV := 0.0
			for _, x := range v.Values {
				if !math.IsNaN(x) {
					maxV = math.Max(maxV, x)
				}
			}
			f := v.Format
			if f == "" {
				f = "%.0f"
			}
			out := make([]htmlBar, len(v.Values))
			for i, x := range v.Values {
				pct := 0.0
				if maxV > 0 && !math.IsNaN(x) {
					pct = x / maxV * 100
				}
				out[i] = htmlBar{v.Labels[i], fmt.Sprintf(f, x) + v.Unit, pct}
			}
			return out
		},
		"heat": func(b Block) htmlHeat {
			v := b.(*Heatmap)
			maxV := 0.0
			for _, row := range v.Values {
				for _, x := range row {
					maxV = math.Max(maxV, x)
				}
			}
			h := htmlHeat{Cols: v.ColLabels}
			for i, row := range v.Values {
				var hr struct {
					Label string
					Cells []struct {
						Alpha float64
						Value string
					}
				}
				hr.Label = v.RowLabels[i]
				for _, x := range row {
					a := 0.0
					if maxV > 0 {
						a = x / maxV
					}
					hr.Cells = append(hr.Cells, struct {
						Alpha float64
						Value string
					}{math.Round(a*100) / 100, fmt.Sprintf("%.0f", x)})
				}
				h.Rows = append(h.Rows, hr)
			}
			return h
		},
		"scatter": func(b Block) string { return strings.Join(ScatterLines(b.(*Scatter), Unicode, 60, 16), "\n") },
	}
	t, err := template.New("r").Funcs(funcs).Parse(htmlTemplate)
	if err != nil {
		return err
	}
	return t.Execute(w, r)
}

const htmlTemplate = `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Title}}</title>
<style>
:root{--bg:#f7f7f5;--card:#fff;--ink:#1d2330;--muted:#5d6676;--line:#e3e4e8;--accent:#6b3fa0;--accent2:#c9a227}
@media (prefers-color-scheme:dark){:root{--bg:#14161b;--card:#1d2028;--ink:#e8e9ee;--muted:#9aa1b0;--line:#2e323c;--accent:#b48ce6;--accent2:#e2c25c}}
body{margin:0;background:var(--bg);color:var(--ink);font:15px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif}
main{max-width:1000px;margin:0 auto;padding:24px 16px 64px}
h1{font-size:28px;margin:0 0 4px}h2{font-size:20px;margin:0 0 6px}
.muted{color:var(--muted)}.card{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:20px;margin:18px 0}
.tag{display:inline-block;font-size:12px;padding:2px 8px;border-radius:99px;background:color-mix(in srgb,var(--accent) 15%,transparent);color:var(--accent);margin-right:6px}
.finding{border-left:4px solid var(--accent2);padding:8px 12px;margin:12px 0;background:color-mix(in srgb,var(--accent2) 10%,transparent)}
table{border-collapse:collapse;width:100%;font-size:13px;margin:8px 0}th,td{border-bottom:1px solid var(--line);padding:5px 8px;text-align:left}
th{color:var(--muted);font-weight:600}td.r,th.r{text-align:right;font-variant-numeric:tabular-nums}
.bar{display:grid;grid-template-columns:minmax(120px,240px) 1fr 90px;gap:8px;align-items:center;font-size:13px;margin:3px 0}
.bar .track{background:var(--line);border-radius:3px;height:14px}.bar .fill{background:var(--accent);height:14px;border-radius:3px}
.bar .v{text-align:right;font-variant-numeric:tabular-nums}
.heat{overflow-x:auto}.heat td{padding:0;width:26px;height:20px;text-align:center;font-size:10px;border:1px solid var(--card)}
.heat th{font-size:11px;padding:2px 4px}
pre{font:12px/1.25 ui-monospace,Menlo,monospace;overflow-x:auto;background:var(--bg);padding:10px;border-radius:6px}
h3{font-size:14px;margin:16px 0 6px}ul{padding-left:20px}.note{font-size:13px;color:var(--muted)}
@media (max-width:600px){.bar{grid-template-columns:110px 1fr 70px}}
</style></head><body><main>
<h1>{{.Title}}</h1><div class="muted">Generated {{.Generated.Format "January 2, 2006 15:04 MST"}}</div>
{{if .DataNote}}<p class="finding">{{.DataNote}}</p>{{end}}
<div class="card"><table>{{range .Inputs}}<tr><th>{{index . 0}}</th><td>{{index . 1}}</td></tr>{{end}}</table></div>
{{range $i, $s := .Sections}}<section class="card" id="{{$s.ID}}">
<span class="tag">Q{{inc $i}}</span><span class="tag">{{$s.Stakeholder}}</span>
<h2>{{$s.Question}}</h2><div class="muted">Datasets: {{range $j, $d := $s.Datasets}}{{if $j}} + {{end}}{{$d}}{{end}}</div>
<div class="finding"><strong>Finding.</strong> {{$s.Finding}}</div>
{{range $b := $s.Blocks}}{{$k := kind $b}}
{{if eq $k "table"}}{{$t := table $b}}<h3>{{$t.Title}}</h3><table><tr>{{range $c, $h := $t.Headers}}<th{{if right $t $c}} class="r"{{end}}>{{$h}}</th>{{end}}</tr>
{{range $t.Rows}}<tr>{{range $c, $v := .}}<td{{if right $t $c}} class="r"{{end}}>{{$v}}</td>{{end}}</tr>{{end}}</table>
{{else if eq $k "bars"}}<h3>{{$b.Title}}</h3>{{range bars $b}}<div class="bar"><span>{{.Label}}</span><div class="track"><div class="fill" style="width:{{printf "%.1f" .Pct}}%"></div></div><span class="v">{{.Value}}</span></div>{{end}}
{{else if eq $k "heat"}}{{$h := heat $b}}<h3>{{$b.Title}}</h3><div class="heat"><table><tr><th></th>{{range $h.Cols}}<th>{{.}}</th>{{end}}</tr>
{{range $h.Rows}}<tr><th>{{.Label}}</th>{{range .Cells}}<td title="{{.Value}}" style="background:rgba(107,63,160,{{.Alpha}})"></td>{{end}}</tr>{{end}}</table></div>
{{else if eq $k "scatter"}}<h3>{{$b.Title}}</h3><pre>{{scatter $b}}</pre>
{{else if eq $k "kv"}}{{$v := kv $b}}<h3>{{$v.Title}}</h3><table>{{range $v.Pairs}}<tr><td>{{index . 0}}</td><td class="r">{{index . 1}}</td></tr>{{end}}</table>{{end}}
{{end}}
{{if $s.Actions}}<h3>Recommended actions</h3><ul>{{range $s.Actions}}<li>{{.}}</li>{{end}}</ul>{{end}}
{{range $s.Notes}}<p class="note">Note: {{.}}</p>{{end}}
</section>{{end}}
{{if .Runtime}}<div class="card"><h2>Runtime and resources</h2><table>{{range .Runtime}}<tr><th>{{index . 0}}</th><td>{{index . 1}}</td></tr>{{end}}</table></div>{{end}}
</main></body></html>`

// ---- CSV export -------------------------------------------------------------

// ExportCSV writes every Table that has a CSVName into dir. Returns file names.
func ExportCSV(dir string, r *Report) ([]string, error) {
	var files []string
	for _, s := range r.Sections {
		for _, b := range s.Blocks {
			t, ok := b.(*Table)
			if !ok || t.CSVName == "" {
				continue
			}
			path := filepath.Join(dir, t.CSVName)
			f, err := os.Create(path)
			if err != nil {
				return files, err
			}
			w := csv.NewWriter(f)
			_ = w.Write(t.Headers)
			_ = w.WriteAll(t.Rows)
			w.Flush()
			if err := f.Close(); err != nil {
				return files, err
			}
			files = append(files, path)
		}
	}
	return files, nil
}
