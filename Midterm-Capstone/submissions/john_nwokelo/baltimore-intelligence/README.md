Baltimore City Service Intelligence Engine

Project Overview

This Go program combines two datasets from the Open Baltimore website to help city officials understand 311 service requests in different neighborhoods. It answers five questions and shows rankings and bar charts in the terminal.

Requirements

A Mac computer with Terminal
Go version 1.21 or newer from https://go.dev/dl/
No Python or extra Go packages needed

Datasets

Download both datasets from https://data.baltimorecity.gov/ in GeoJSON format.

1. 311 Customer Service Requests
   Save as data/requests.geojson
2. Neighborhood Statistical Area (NSA) Boundaries
   Save as data/neighborhoods.geojson

Put both files in the data folder. The included sample files contain made-up data for testing and should not be treated as real Baltimore statistics.

How to Compile and Run on macOS

Open Terminal in the baltimore-intelligence folder containing go.mod.

Step 1. Check Go and build the program.

go version
go build -o baltimore-engine ./cmd/engine

Step 2. Run the program with the real datasets.

./baltimore-engine -requests data/requests.geojson -boundaries data/neighborhoods.geojson

You can also use the included execution script.

chmod +x run.sh
./run.sh -requests data/requests.geojson -boundaries data/neighborhoods.geojson

Step 3. Verify the program.

go test ./...
go vet ./...

You should see five questions with neighborhood statistics. To try the program with sample data, run ./run.sh without additional arguments.

On Windows, you can double-click run-windows.cmd after adding the real datasets.

Research Questions

1. Which neighborhoods report the most problems?
2. What problems are most common in each neighborhood?
3. Where is the highest percentage of problems still unresolved?
4. Where does it take the longest to fix problems?
5. Which neighborhoods need the most attention?

The fifth question calculates a priority score using 40 percent request volume, 40 percent unresolved request rate, and 20 percent average closing time when dates are available. A higher score means the area may deserve closer review. It does not prove the area has worse city services.

How the Program Works

Data loading: Go goroutines load both datasets at the same time.
Matching: Worker pools match 311 requests to neighborhood boundaries.
Analysis: Workers calculate counts, unresolved rates, and closing times.
Output: The program displays tables, rankings, and ASCII bar charts.

This follows a pipeline design that separates loading, matching, analysis, and output. Go manages memory automatically. Limited worker pools and separate worker data help avoid unsafe shared access. Large datasets may use substantial memory.

Data Limitations

Some requests have missing dates or locations. Neighborhood population and differences in how often residents use 311 can also affect the results.
