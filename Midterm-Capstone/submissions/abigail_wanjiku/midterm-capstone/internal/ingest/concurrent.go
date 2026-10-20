package ingest

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
)

// DefaultChunkSize is the target size of the byte chunks handed to workers.
// Raw bytes in flight are roughly (workers + queued + 2) chunks per file, so
// smaller chunks mean less memory; 1 MiB is still large enough that the
// per-chunk overhead (a csv.Reader and small intern maps) is negligible.
const DefaultChunkSize = 1 << 20

// carrySlack is extra capacity in each chunk buffer so the partial record
// carried over from the previous chunk fits without reallocating.
const carrySlack = 64 << 10

// ConcurrentOptions configures Concurrent. Zero values select defaults.
type ConcurrentOptions struct {
	Workers   int // goroutines parsing chunks; default runtime.NumCPU()
	ChunkSize int // target bytes per chunk; default DefaultChunkSize
}

// chunk is a run of whole CSV records cut from the file.
type chunk struct {
	index     int    // position in the file, used to reassemble results in order
	firstLine int    // file line number of data's first line
	lines     int    // newlines in data: an upper bound on its records
	data      []byte // complete records only; owned by whoever holds the chunk
}

// result is what a worker sends back for one chunk.
type result[T any] struct {
	index   int
	records []T
	stats   Stats
	err     error
}

// Concurrent loads a CSV with a pool of workers and returns the same records,
// in the same order, and the same Stats as running scan over a single
// sequential Scanner.
//
// One splitter goroutine reads the file and cuts it into chunks at record
// boundaries. Workers each parse whole chunks with their own Scanner and call
// scan, which must use only the Scanner it is given. The calling goroutine
// collects results and stitches them together in chunk order.
//
// Memory stays bounded: the chunks channel holds at most two chunks, and
// workers hand finished buffers back to the splitter through a free list, so
// the same few buffers are reused for the whole file.
func Concurrent[T any](r io.Reader, columns []string, opts ConcurrentOptions, scan func(*Scanner) []T) ([]T, Stats, error) {
	workers := cmpOr(opts.Workers, runtime.NumCPU())
	chunkSize := cmpOr(opts.ChunkSize, DefaultChunkSize)

	br := bufio.NewReaderSize(r, 1<<16)
	headerBytes, err := readRecord(br)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("reading header: %w", err)
	}
	header, err := newCSVReader(bytes.NewReader(headerBytes)).Read()
	if err != nil {
		return nil, Stats{}, fmt.Errorf("reading header: %w", err)
	}
	idx, err := columnIndex(header, columns)
	if err != nil {
		return nil, Stats{}, err
	}
	firstDataLine := 1 + bytes.Count(headerBytes, []byte{'\n'})

	chunks := make(chan chunk, 2)            // splitter -> workers; small, so few buffers wait
	free := make(chan []byte, workers+2)     // workers -> splitter: buffers to reuse
	results := make(chan result[T], workers) // workers -> collector
	splitErr := make(chan error, 1)          // splitter -> collector, sent once

	// Splitter: the only goroutine that reads from br.
	go func() {
		defer close(chunks) // tells workers no more chunks are coming
		splitErr <- split(br, chunkSize, firstDataLine, chunks, free)
	}()

	// Workers: each owns the Scanner and slices it builds; nothing is shared.
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range chunks {
				sc := newScanner(newCSVReader(bytes.NewReader(c.data)), idx, c.firstLine-1)
				sc.rowsHint = c.lines
				recs := scan(sc)
				results <- result[T]{index: c.index, records: recs, stats: sc.Stats, err: sc.Err()}
				// Nothing kept from the chunk refers to its bytes (csv.Reader
				// copies fields out, and kept strings are interned copies), so
				// the buffer can be recycled. If the free list is full, drop it.
				select {
				case free <- c.data[:0]:
				default:
				}
			}
		}()
	}

	// Closer: once every worker has exited, no more results can be sent.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collector (this goroutine): the only writer of parts. Results arrive in
	// any order; storing them by index restores file order.
	var parts []result[T]
	var errs []error
	for res := range results {
		if res.index >= len(parts) {
			parts = append(parts, make([]result[T], res.index+1-len(parts))...)
		}
		parts[res.index] = res
		if res.err != nil {
			errs = append(errs, res.err)
		}
	}
	if err := <-splitErr; err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return nil, Stats{}, errors.Join(errs...)
	}

	total := 0
	for i := range parts {
		total += len(parts[i].records)
	}
	out := make([]T, 0, total) // exact size: one allocation, no regrowth
	var stats Stats
	for i := range parts {
		out = append(out, parts[i].records...)
		parts[i].records = nil // let the GC reclaim each chunk's slice once copied
		stats.Merge(&parts[i].stats)
	}
	return out, stats, nil
}

// split reads br to EOF and sends it as chunks of about size bytes, each
// ending on a record boundary. firstLine is the file line number of the
// first byte read. Buffers come from free when available.
func split(br *bufio.Reader, size, firstLine int, out chan<- chunk, free chan []byte) error {
	index, line := 0, firstLine
	buf := getBuffer(free, size) // starts with the carried-over partial record, if any
	for {
		need := len(buf) + size
		if cap(buf) < need { // a record longer than carrySlack: grow (rare)
			bigger := make([]byte, len(buf), need+carrySlack)
			copy(bigger, buf)
			buf = bigger
		}
		n, err := io.ReadFull(br, buf[len(buf):need])
		buf = buf[:len(buf)+n]
		eof := err == io.EOF || err == io.ErrUnexpectedEOF
		if err != nil && !eof {
			return err
		}

		cut := len(buf)
		if !eof {
			cut = lastRecordEnd(buf)
			if cut == 0 { // one record bigger than the buffer: read more into it
				continue
			}
		}
		// Copy the partial record into the next buffer before sending this
		// one: once sent, a worker owns buf and may recycle it at any time.
		var next []byte
		if !eof {
			next = append(getBuffer(free, size), buf[cut:]...)
		}
		if cut > 0 {
			lines := bytes.Count(buf[:cut], []byte{'\n'})
			out <- chunk{index: index, firstLine: line, lines: lines + 1, data: buf[:cut]}
			index++
			line += lines
		}
		if eof {
			return nil
		}
		buf = next
	}
}

// getBuffer returns an empty buffer with room for size bytes plus a carried
// record, reusing one from free if possible.
func getBuffer(free chan []byte, size int) []byte {
	select {
	case b := <-free:
		if cap(b) >= size+carrySlack {
			return b[:0]
		}
	default:
	}
	return make([]byte, 0, size+carrySlack)
}

// lastRecordEnd returns the index just past the last newline in b that ends a
// record, or 0 if there is none. b must start at a record boundary.
//
// It follows encoding/csv's rules rather than just counting quotes: a quote
// opens a quoted field only at the start of a field, "" inside one is an
// escaped quote, CRLF ends a line like LF, and after a quoting error csv
// skips to the end of the line. That keeps chunk boundaries where a
// sequential csv.Reader would put them, even for malformed rows.
//
// Only quotes and newlines can change where a record ends, so instead of
// visiting every byte it jumps between them with bytes.IndexByte, which is
// vectorized. The splitter runs on one goroutine, so its speed caps the
// whole concurrent load.
func lastRecordEnd(b []byte) int {
	end := 0
	nextQuote, nextNL := -1, -1 // cached positions of the next '"' and '\n' at or after i
	for i := 0; i < len(b); {
		// Outside quotes at position i.
		if nextQuote < i {
			nextQuote = indexFrom(b, i, '"')
		}
		if nextNL < i {
			nextNL = indexFrom(b, i, '\n')
		}
		if nextNL < nextQuote { // line ends before any quote: record boundary
			end, i = nextNL+1, nextNL+1
			continue
		}
		if nextQuote == len(b) { // no quote or newline left
			break
		}

		q := nextQuote
		if q > 0 && b[q-1] != ',' && b[q-1] != '\n' {
			// Bare quote inside an unquoted field: csv rejects the record and
			// resumes after the end of this line.
			if nextNL == len(b) {
				break
			}
			end, i = nextNL+1, nextNL+1
			continue
		}

		// Quoted field: find the closing quote, skipping escaped "" pairs.
		j := q + 1
		for {
			c := indexFrom(b, j, '"')
			if c+1 >= len(b) {
				return end // field (or what follows its quote) continues past b
			}
			switch k := c + 1; b[k] {
			case '"': // escaped quote
				j = k + 1
				continue
			case ',': // next field starts after the comma
				i = k + 1
			case '\n':
				end, i = k+1, k+1
			case '\r':
				if k+1 >= len(b) {
					return end
				}
				if b[k+1] == '\n' {
					end, i = k+2, k+2
					break
				}
				fallthrough
			default:
				// Extraneous quote in a quoted field: csv rejects the record
				// and resumes after the end of this line.
				nl := indexFrom(b, k, '\n')
				if nl == len(b) {
					return end
				}
				end, i = nl+1, nl+1
			}
			break
		}
	}
	return end
}

// indexFrom returns the index of the first c in b at or after i, or len(b).
func indexFrom(b []byte, i int, c byte) int {
	if j := bytes.IndexByte(b[i:], c); j >= 0 {
		return i + j
	}
	return len(b)
}

// readRecord reads one complete CSV record (which may span lines if a quoted
// field contains a newline), including its trailing newline.
func readRecord(br *bufio.Reader) ([]byte, error) {
	var rec []byte
	for {
		line, err := br.ReadBytes('\n')
		rec = append(rec, line...)
		if bytes.Count(rec, []byte{'"'})%2 == 0 && len(rec) > 0 {
			return rec, nil
		}
		if err != nil {
			if err == io.EOF && len(rec) > 0 {
				return rec, nil
			}
			return nil, err
		}
	}
}

func cmpOr(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}
