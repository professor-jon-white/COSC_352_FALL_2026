// Package analysis holds the five stakeholder research questions. Each is a
// Question strategy over the read-only joined data, so they run concurrently
// and new questions can be added without touching ingestion or rendering.
package analysis

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"baltimore-intel/internal/join"
	"baltimore-intel/internal/report"
)

// Question is the analysis Strategy.
type Question interface {
	ID() string
	Run(j *join.Joined) (*report.Section, error)
}

// All returns the registered questions in report order.
func All() []Question {
	return []Question{CompoundDistress{}, LiquorCrime{}, VacancySanitation{}, ServiceEquity{}, LateNight{}}
}

// Result pairs a question's output with its timing.
type Result struct {
	Section  *report.Section
	Err      error
	Duration time.Duration
}

// RunAll executes every question on its own goroutine and returns results
// in registration order (deterministic output regardless of finish order).
func RunAll(j *join.Joined, qs []Question) []Result {
	out := make([]Result, len(qs))
	var wg sync.WaitGroup
	for i, q := range qs {
		wg.Add(1)
		go func(i int, q Question) {
			defer wg.Done()
			defer func() { // one failing analysis must not crash the report
				if r := recover(); r != nil {
					out[i].Err = fmt.Errorf("%s panicked: %v", q.ID(), r)
				}
			}()
			start := time.Now()
			s, err := q.Run(j)
			out[i] = Result{Section: s, Err: err, Duration: time.Since(start)}
		}(i, q)
	}
	wg.Wait()
	return out
}

// ---- helpers shared by the questions ------------------------------------

type ranked struct {
	key   string
	value float64
}

func topN(m map[string]float64, n int) []ranked {
	out := make([]ranked, 0, len(m))
	for k, v := range m {
		out = append(out, ranked{k, v})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].value != out[b].value {
			return out[a].value > out[b].value
		}
		return out[a].key < out[b].key
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func share(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

func ratio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
