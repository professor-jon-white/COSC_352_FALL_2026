package main

import (
"flag"
"fmt"
"os"
"path/filepath"
"runtime"
"sync"
"time"

"bmore-engine/internal/engine"
"bmore-engine/internal/ingest"
"bmore-engine/internal/model"
"bmore-engine/internal/render"
"bmore-engine/internal/seed"
)

func main() {
default311 := filepath.Join("data", "311_requests.csv")
defaultVacant := filepath.Join("data", "vacant_buildings.csv")

path311 := flag.String("requests", default311, "Path to Open Baltimore 311 Service Requests CSV")
pathVacant := flag.String("vacants", defaultVacant, "Path to Open Baltimore Vacant Building Notices CSV")
workers := flag.Int("workers", runtime.NumCPU(), "Number of concurrent worker goroutines")
radius := flag.Float64("radius", 150.0, "Spatial proximity join radius in meters")
forceSeed := flag.Bool("seed", false, "Regenerate deterministic Open Baltimore CSV datasets in ./data")
flag.Parse()

if err := seed.EnsureDatasets(*path311, *pathVacant, *forceSeed); err != nil {
fmt.Fprintf(os.Stderr, "error preparing datasets: %v\n", err)
os.Exit(1)
}

ingestStart := time.Now()

var (
requests []model.ServiceRequest
vacants  []model.VacantBuilding
err311   error
errVac   error
wg       sync.WaitGroup
)

wg.Add(2)
go func() {
defer wg.Done()
requests, err311 = ingest.Load311Concurrent(*path311, *workers)
}()
go func() {
defer wg.Done()
vacants, errVac = ingest.LoadVacantConcurrent(*pathVacant, *workers)
}()
wg.Wait()

if err311 != nil {
fmt.Fprintf(os.Stderr, "failed to load 311 requests: %v\n", err311)
os.Exit(1)
}
if errVac != nil {
fmt.Fprintf(os.Stderr, "failed to load vacant buildings: %v\n", errVac)
os.Exit(1)
}

ingestDuration := time.Since(ingestStart)

spatialStrategy := engine.HaversineGridJoin{MaxRadiusMeters: *radius}
procEngine := engine.NewEngineBuilder().
WithWorkers(*workers).
WithHighVacancyThreshold(25).
WithSpatialStrategy(spatialStrategy).
Build()

report := procEngine.Execute(requests, vacants, ingestDuration)
render.RenderTerminal(os.Stdout, report)
}
