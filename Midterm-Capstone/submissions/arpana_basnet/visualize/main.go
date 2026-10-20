package main

import (
	"fmt"
	"html/template"
	"os"
	"strconv"
	"strings"
)

type Bar struct {
	Label   string
	Value   float64
	Display string
	Width   float64
}

type Chart struct {
	Title string
	Unit  string
	Bars  []Bar
}

type Report struct {
	Correlation string
	Charts      []Chart
	Details     string
}

func main() {
	if err := generateReport(); err != nil {
		fmt.Fprintln(os.Stderr, "Report error:", err)
		os.Exit(1)
	}
	fmt.Println("Created report.html")
}

func generateReport() error {
	content, err := os.ReadFile("results.txt")
	if err != nil {
		return err
	}

	report := Report{
		Details: string(content),
		Charts: []Chart{
			{Title: "1. Top neighborhoods by notice count", Unit: "Notice records"},
			{Title: "3. Illegal dumping in the top-ten notice neighborhoods", Unit: "Request records"},
			{Title: "4. Unresolved requests by notice group", Unit: "Percent of included requests"},
			{Title: "5. Average recorded closure time", Unit: "Days"},
		},
	}

	question := 0
	group := ""

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "QUESTION ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				question, _ = strconv.Atoi(
					strings.TrimSuffix(fields[1], ":"),
				)
			}
			continue
		}

		if strings.HasPrefix(line, "Neighborhoods found in both") {
			question = 0
		}

		if strings.HasPrefix(line, "Pearson correlation:") {
			report.Correlation = strings.TrimSpace(
				strings.TrimPrefix(line, "Pearson correlation:"),
			)
		}

		if strings.HasPrefix(line, "Group:") {
			group = strings.TrimSpace(strings.TrimPrefix(line, "Group:"))
		}

		if question == 1 || question == 3 {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}

			notices, err1 := strconv.Atoi(fields[len(fields)-2])
			requests, err2 := strconv.Atoi(fields[len(fields)-1])
			if err1 != nil || err2 != nil {
				continue
			}

			label := strings.Join(fields[:len(fields)-2], " ")
			index, value := 0, notices

			if question == 3 {
				index, value = 1, requests
			}

			report.Charts[index].Bars = append(
				report.Charts[index].Bars,
				Bar{
					Label:   label,
					Value:   float64(value),
					Display: strconv.Itoa(value),
				},
			)
		}

		if question == 4 &&
			strings.HasPrefix(line, "Unresolved percentage:") {
			text := strings.TrimSpace(
				strings.TrimPrefix(line, "Unresolved percentage:"),
			)
			value, err := strconv.ParseFloat(
				strings.TrimSuffix(text, "%"), 64,
			)
			if err == nil {
				report.Charts[2].Bars = append(
					report.Charts[2].Bars,
					Bar{Label: group, Value: value, Display: text},
				)
			}
		}

		if question == 5 &&
			strings.HasPrefix(line, "Average closure time:") {
			text := strings.TrimSpace(
				strings.TrimPrefix(line, "Average closure time:"),
			)
			value, err := strconv.ParseFloat(
				strings.TrimSuffix(text, " days"), 64,
			)
			if err == nil {
				report.Charts[3].Bars = append(
					report.Charts[3].Bars,
					Bar{Label: group, Value: value, Display: text},
				)
			}
		}
	}

	if report.Correlation == "" {
		return fmt.Errorf("Question 2 correlation missing from results.txt")
	}

	for index := range report.Charts {
		chart := &report.Charts[index]
		if len(chart.Bars) == 0 {
			return fmt.Errorf("chart data missing: %s", chart.Title)
		}

		maximum := 0.0
		for _, bar := range chart.Bars {
			if bar.Value > maximum {
				maximum = bar.Value
			}
		}

		for barIndex := range chart.Bars {
			if maximum > 0 {
				chart.Bars[barIndex].Width =
					100 * chart.Bars[barIndex].Value / maximum
			}
		}
	}

	page, err := template.New("report").Parse(pageTemplate)
	if err != nil {
		return err
	}

	file, err := os.Create("report.html")
	if err != nil {
		return err
	}

	if err := page.Execute(file, report); err != nil {
		file.Close()
		return err
	}

	return file.Close()
}

const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Baltimore Municipal Data Report</title>
<style>
body {
	font-family: system-ui, sans-serif;
	background: #eef2f6;
	color: #172b40;
	margin: 0;
	padding: 28px;
}
main { max-width: 1050px; margin: auto; }
h1 { font-size: 32px; margin-bottom: 8px; }
p { line-height: 1.6; }
section {
	background: white;
	border-radius: 12px;
	padding: 24px;
	margin: 22px 0;
	box-shadow: 0 2px 8px #0000000d;
}
h2 { font-size: 21px; margin-top: 0; }
.row {
	display: grid;
	grid-template-columns: minmax(170px, 2fr) 3fr 100px;
	gap: 14px;
	align-items: center;
	margin: 14px 0;
}
.label { font-size: 13px; overflow-wrap: anywhere; }
.track { background: #edf1f5; height: 18px; border-radius: 4px; }
.bar { background: #2576ad; height: 100%; border-radius: 4px; }
.value { text-align: right; font-variant-numeric: tabular-nums; }
.number { font-size: 42px; font-weight: 700; color: #2576ad; }
.note { color: #526579; font-size: 14px; }
pre {
	white-space: pre;
	overflow-x: auto;
	background: #f4f6f8;
	padding: 18px;
	font-size: 12px;
}
@media (max-width: 600px) {
	body { padding: 12px; }
	section { padding: 16px; }
	.row { grid-template-columns: 1fr; gap: 5px; }
	.value { text-align: left; }
}
</style>
</head>
<body>
<main>
<h1>Baltimore Municipal Data Report</h1>
<p>311 service requests joined with vacant-building notice records by neighborhood.</p>

<section>
<h2>2. Relationship between notice and request counts</h2>
<div class="number">{{.Correlation}}</div>
<p>Pearson correlation across matched neighborhoods.</p>
<p class="note">Correlation ranges from -1 to +1.
This measures association, not causation. Counts are not adjusted for population.</p>
</section>

{{range .Charts}}
<section>
<h2>{{.Title}}</h2>
<p class="note">{{.Unit}}. Bars start at zero and scale to the largest value in this chart.</p>
{{range .Bars}}
<div class="row">
<div class="label">{{.Label}}</div>
<div class="track"><div class="bar" style="width: {{.Width}}%"></div></div>
<div class="value">{{.Display}}</div>
</div>
{{end}}
</section>
{{end}}

<section>
<h2>Interpretation and limitations</h2>
<p>Use the findings to guide neighborhood reviews and investigate differences
in service-request categories. These comparisons do not establish causes or
adjust for neighborhood size and request mix.</p>
<p>The 100-notice threshold is a project choice. Notice records are not
necessarily unique buildings. Dataset time periods may differ.
Recorded closure does not prove an issue was resolved.</p>
</section>

<section>
<h2>Complete results and data-quality counts</h2>
<pre>{{.Details}}</pre>
</section>
</main>
</body>
</html>`
