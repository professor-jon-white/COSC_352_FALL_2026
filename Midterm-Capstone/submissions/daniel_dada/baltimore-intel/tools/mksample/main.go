// mksample builds small, evenly spread sample CSVs from the full datasets
// so the repository can run offline. Usage:
//
//	go run ./tools/mksample
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const dateLayout = "1/2/2006 3:04:05 PM"

// sample copies every step-th row whose date falls in `year`.
func sample(inPath, outPath, dateCol string, year, step int) (matched, kept int, err error) {
	in, err := os.Open(inPath)
	if err != nil {
		return 0, 0, err
	}
	defer in.Close()

	out, err := os.Create(outPath)
	if err != nil {
		return 0, 0, err
	}
	defer out.Close()

	r := csv.NewReader(in)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	w := csv.NewWriter(out)
	defer w.Flush()

	header, err := r.Read()
	if err != nil {
		return 0, 0, err
	}
	col := -1
	for i, h := range header {
		if strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")) == dateCol {
			col = i
		}
	}
	if col < 0 {
		return 0, 0, fmt.Errorf("%s: column %q not found", inPath, dateCol)
	}
	if err := w.Write(header); err != nil {
		return 0, 0, err
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || col >= len(rec) {
			continue
		}
		t, err := time.Parse(dateLayout, strings.TrimSpace(rec[col]))
		if err != nil || t.Year() != year {
			continue
		}
		matched++
		if (matched-1)%step == 0 {
			if err := w.Write(rec); err != nil {
				return matched, kept, err
			}
			kept++
		}
	}
	return matched, kept, w.Error()
}

func main() {
	in := flag.String("in", "./data", "folder with the full CSV files")
	out := flag.String("out", "./data/sample", "folder to write the samples to")
	year := flag.Int("year", 2024, "year to sample")
	crimeStep := flag.Int("crime-step", 9, "keep every Nth crime row")
	reqStep := flag.Int("req-step", 200, "keep every Nth 311 row")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	jobs := []struct {
		name, dateCol string
		step          int
	}{
		{"crime.csv", "CrimeDateTime", *crimeStep},
		{fmt.Sprintf("311_%d.csv", *year), "CreatedDate", *reqStep},
	}
	for _, j := range jobs {
		matched, kept, err := sample(filepath.Join(*in, j.name), filepath.Join(*out, j.name), j.dateCol, *year, j.step)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Printf("%s: %d rows from %d, written to %s\n", j.name, kept, matched, *out)
	}
}
