// Package report separates WHAT is reported (the Report model, assembled with
// a Builder) from HOW it is shown (Renderer strategies for terminal text and
// Markdown). Analysis code never formats output itself.
package report

const (
	KindPara = iota
	KindBullet
	KindChart
	KindTable
)

// Chart is a horizontal bar chart.
type Chart struct {
	Title  string
	Labels []string
	Values []float64
	Unit   string
}

// Table is a simple grid of strings.
type Table struct {
	Headers []string
	Rows    [][]string
}

type Block struct {
	Kind  int
	Text  string
	Chart Chart
	Table Table
}

type Section struct {
	Heading    string
	Subheading string
	Blocks     []Block
}

type Report struct {
	Title    string
	Meta     []string
	Sections []Section
}

// Builder assembles a Report step by step (Builder pattern).
type Builder struct{ r Report }

func NewBuilder(title string) *Builder { return &Builder{r: Report{Title: title}} }

func (b *Builder) Meta(s string) *Builder { b.r.Meta = append(b.r.Meta, s); return b }

func (b *Builder) Section(heading, sub string) *Builder {
	b.r.Sections = append(b.r.Sections, Section{Heading: heading, Subheading: sub})
	return b
}

func (b *Builder) add(bl Block) *Builder {
	if len(b.r.Sections) == 0 {
		b.Section("", "")
	}
	s := &b.r.Sections[len(b.r.Sections)-1]
	s.Blocks = append(s.Blocks, bl)
	return b
}

func (b *Builder) Para(t string) *Builder   { return b.add(Block{Kind: KindPara, Text: t}) }
func (b *Builder) Bullet(t string) *Builder { return b.add(Block{Kind: KindBullet, Text: t}) }
func (b *Builder) Chart(c Chart) *Builder   { return b.add(Block{Kind: KindChart, Chart: c}) }
func (b *Builder) Table(t Table) *Builder   { return b.add(Block{Kind: KindTable, Table: t}) }

func (b *Builder) Build() *Report { r := b.r; return &r }
