package ingest

import (
	"context"
	"testing"
	"time"
)

// Live integration check; skipped with -short.
func TestFetchAllLive(t *testing.T) {
	if testing.Short() {
		t.Skip("live network test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	d, err := FetchAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("311: %d, vacant: %d", len(d.ServiceRequests), len(d.VacantBuildings))
	if len(d.ServiceRequests) != maxServiceRequests || len(d.VacantBuildings) < 10000 {
		t.Fatalf("unexpected counts")
	}
	t.Logf("sample: %+v | %+v", d.ServiceRequests[0], d.VacantBuildings[0])
}
