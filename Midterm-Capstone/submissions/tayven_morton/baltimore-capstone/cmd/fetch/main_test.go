package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func fakeSource(t *testing.T, mode string) (*httptest.Server, source) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("returnCountOnly") == "true" {
			fmt.Fprint(w, `{"count":3}`)
			return
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("resultOffset"))
		ids := []int{1, 2}
		more := true
		if offset > 0 {
			ids = []int{3}
			more = false
		}
		if mode == "duplicate" && offset > 0 {
			ids = []int{2}
		}
		if mode == "truncated" {
			ids = []int{1}
			more = false
		}
		features := []map[string]any{}
		for _, id := range ids {
			features = append(features, map[string]any{"attributes": map[string]any{"OID": id, "Neighborhood": "A"}})
		}
		json.NewEncoder(w).Encode(map[string]any{"features": features, "exceededTransferLimit": more})
	}))
	return server, source{Name: "test", URL: server.URL, Where: "1=1", Fields: "OID,Neighborhood", ID: "OID", PageSize: 2}
}
func TestDownloadPaginationAndHash(t *testing.T) {
	s, src := fakeSource(t, "normal")
	defer s.Close()
	dir := t.TempDir()
	m, err := download(dir, src)
	if err != nil {
		t.Fatal(err)
	}
	if m.Records != 3 || m.Pages != 2 || m.After != 3 {
		t.Fatal(m)
	}
	b, err := os.ReadFile(filepath.Join(dir, "test.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(b)
	if m.SHA256 != hex.EncodeToString(hash[:]) || strings.Count(string(b), "\n") != 3 {
		t.Fatal(m)
	}
}
func TestRejectDuplicateAndTruncatedDownloads(t *testing.T) {
	for _, mode := range []string{"duplicate", "truncated"} {
		t.Run(mode, func(t *testing.T) {
			s, src := fakeSource(t, mode)
			defer s.Close()
			dir := t.TempDir()
			if _, err := download(dir, src); err == nil {
				t.Fatal("invalid download accepted")
			}
			if _, err := os.Stat(filepath.Join(dir, "test.jsonl")); !os.IsNotExist(err) {
				t.Fatal("invalid final file exists")
			}
			if _, err := os.Stat(filepath.Join(dir, "test.jsonl.partial")); !os.IsNotExist(err) {
				t.Fatal("partial file not removed")
			}
		})
	}
}
