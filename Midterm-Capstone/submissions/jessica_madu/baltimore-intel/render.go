package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type kv struct {
	K string
	V int
}

func topN(m map[string]int, n int) []kv {
	out := make([]kv, 0, len(m))
	for k, v := range m {
		out = append(out, kv{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].V != out[j].V {
			return out[i].V > out[j].V
		}
		return out[i].K < out[j].K
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

func bar(v, max float64, width int) string {
	if max <= 0 {
		return ""
	}
	n := int(v / max * float64(width))
	if n == 0 && v > 0 {
		n = 1
	}
	return strings.Repeat("█", n)
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func heading(w io.Writer, i int, title string) {
	fmt.Fprintf(w, "\n%s\nQ%d. %s\n%s\n", strings.Repeat("=", 78), i, title, strings.Repeat("=", 78))
}
