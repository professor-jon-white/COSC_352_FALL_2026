package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

func text(r Row, k string) string {
	if r[k] == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(r[k]))
}
func norm(s string) string { return strings.Join(strings.Fields(strings.ToUpper(s)), " ") }
func incidentID(r Row, k string) string {
	s := norm(text(r, k))
	if s == "" || s == "UNKNOWN" || s == "N/A" || s == "NULL" || s == "0" {
		return ""
	}
	return s
}
func label(r Row, k string) string {
	s := norm(text(r, k))
	if s == "" {
		return "UNKNOWN"
	}
	return s
}
func epoch(v any) int64 {
	var n float64
	switch x := v.(type) {
	case float64:
		n = x
	case int64:
		n = float64(x)
	case string:
		n, _ = strconv.ParseFloat(x, 64)
	}
	if n <= 0 || n > 4102444800000 {
		return 0
	}
	return int64(n)
}
func month(v any) string {
	n := epoch(v)
	if n == 0 {
		return "UNKNOWN"
	}
	loc, _ := time.LoadLocation("America/New_York")
	return time.UnixMilli(n).In(loc).Format("2006-01")
}
func cell(m map[string]*Cell, k string) *Cell {
	if m[k] == nil {
		m[k] = &Cell{Name: k}
	}
	return m[k]
}
func newResult() Result {
	return Result{CategoryByNeighborhood: map[string]map[string]int{}, WeaponByNeighborhood: map[string]map[string]int{}, Neighborhoods: map[string]*Cell{}, Months: map[string]*Cell{}, Categories: map[string]*Cell{}, Weapons: map[string]*Cell{}}
}
