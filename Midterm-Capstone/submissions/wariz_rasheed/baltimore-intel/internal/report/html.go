package report

import (
	"fmt"
	"html/template"
	"io"
	"math"
)

type htmlRenderer struct{}

type barView struct {
	Label string
	Text  string
	Style template.CSS
	Neg   bool
}

type chartView struct {
	Title string
	Bars  []barView
}

type resultView struct {
	Num int
	Result
	ChartViews []chartView
}

type pageView struct {
	Summary
	Items []resultView
}

func (htmlRenderer) Render(w io.Writer, s Summary) error {
	pv := pageView{Summary: s}
	for i, r := range s.Results {
		rv := resultView{Num: i + 1, Result: r}
		for _, c := range r.Charts {
			cv := chartView{Title: c.Title}
			max := maxAbs(c.Bars)
			for _, b := range c.Bars {
				pct := 0.0
				if max > 0 {
					pct = math.Abs(b.Value) / max * 100
				}
				if b.Value != 0 && pct < 1 {
					pct = 1
				}
				cv.Bars = append(cv.Bars, barView{
					Label: b.Label,
					Text:  b.Text,
					Style: template.CSS(fmt.Sprintf("width:%.1f%%", pct)),
					Neg:   b.Value < 0,
				})
			}
			rv.ChartViews = append(rv.ChartViews, cv)
		}
		pv.Items = append(pv.Items, rv)
	}
	return pageTmpl.Execute(w, pv)
}

var pageTmpl = template.Must(template.New("page").Parse(pageSrc))

const pageSrc = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
:root { --bg:#f6f7f9; --card:#ffffff; --ink:#1c2430; --muted:#5b6675; --line:#dde2e8; --accent:#1f5fbf; --neg:#b5452f; --call:#eaf1fc; }
@media (prefers-color-scheme: dark) {
  :root { --bg:#12161c; --card:#1b212a; --ink:#e6e9ee; --muted:#9aa5b4; --line:#2c3541; --accent:#6aa3ff; --neg:#e07a63; --call:#1e2a3d; }
}
* { box-sizing: border-box; }
body { margin:0; background:var(--bg); color:var(--ink); font:15px/1.5 -apple-system, "Segoe UI", Helvetica, Arial, sans-serif; }
main { max-width:980px; margin:0 auto; padding:24px 16px 64px; }
h1 { font-size:26px; margin:0 0 4px; }
h2 { font-size:19px; margin:0 0 4px; }
h3 { font-size:14px; margin:20px 0 8px; color:var(--muted); text-transform:uppercase; letter-spacing:.04em; }
.sub { color:var(--muted); margin:0 0 20px; }
.card { background:var(--card); border:1px solid var(--line); border-radius:10px; padding:20px; margin:0 0 20px; }
.aud { display:inline-block; font-size:12px; color:var(--accent); border:1px solid var(--accent); border-radius:99px; padding:1px 10px; margin-bottom:10px; }
.finding { background:var(--call); border-left:4px solid var(--accent); padding:12px 14px; border-radius:4px; margin:14px 0; }
table { border-collapse:collapse; width:100%; font-size:14px; }
.scroll { overflow-x:auto; }
th, td { text-align:left; padding:6px 10px; border-bottom:1px solid var(--line); white-space:nowrap; }
th { color:var(--muted); font-weight:600; }
.chart { margin:6px 0 4px; }
.row { display:grid; grid-template-columns:minmax(120px,240px) 1fr minmax(90px,170px); gap:10px; align-items:center; margin:3px 0; font-size:13px; }
.lab { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.track { background:var(--line); border-radius:3px; height:16px; }
.bar { background:var(--accent); height:100%; border-radius:3px; }
.bar.neg { background:var(--neg); }
.val { color:var(--muted); }
.note { color:var(--muted); font-size:13px; margin:6px 0; }
footer { color:var(--muted); font-size:13px; margin-top:24px; }
</style>
</head>
<body>
<main>
<h1>{{.Title}}</h1>
<p class="sub">{{.Subtitle}}</p>

<div class="card">
<h2>Data overview</h2>
<div class="scroll">
<table>
<tr>{{range .Overview.Headers}}<th>{{.}}</th>{{end}}</tr>
{{range .Overview.Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>
{{end}}</table>
</div>
</div>

{{range .Items}}
<section class="card" id="q{{.Num}}">
<span class="aud">For: {{.Audience}}</span>
<h2>Question {{.Num}}: {{.Title}}</h2>
<p>{{.Question}}</p>
<div class="finding"><strong>Finding.</strong> {{.Finding}}</div>
{{range .ChartViews}}
<h3>{{.Title}}</h3>
<div class="chart">
{{range .Bars}}<div class="row"><span class="lab" title="{{.Label}}">{{.Label}}</span><div class="track"><div class="bar{{if .Neg}} neg{{end}}" style="{{.Style}}"></div></div><span class="val">{{.Text}}</span></div>
{{end}}</div>
{{end}}
{{range .Tables}}
<h3>{{.Title}}</h3>
<div class="scroll">
<table>
<tr>{{range .Headers}}<th>{{.}}</th>{{end}}</tr>
{{range .Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>
{{end}}</table>
</div>
{{end}}
{{range .Notes}}<p class="note">Note: {{.}}</p>
{{end}}
</section>
{{end}}

<footer>
{{range .Footer}}<div>{{.}}</div>
{{end}}</footer>
</main>
</body>
</html>
`
