#!/bin/sh
# Downloads the five raw datasets from Open Baltimore (data.baltimorecity.gov)
# into ./data. Total download is about 465 MB; the 311 file is 380 MB of it.
# Usage: sh scripts/fetch_data.sh [target-dir]
set -eu

DIR="${1:-data}"
BASE="https://data.baltimorecity.gov/api/download/v1/items"
mkdir -p "$DIR"

fetch() { # <output file> <item id> <format> <layer>
    if [ -s "$DIR/$1" ]; then
        echo "have  $1"
        return
    fi
    echo "fetch $1"
    curl -fL --retry 3 --progress-bar -o "$DIR/$1.part" "$BASE/$2/$3?layers=$4"
    mv "$DIR/$1.part" "$DIR/$1"
}

fetch crime_nibrs.csv               204beefe92a645d79fdf0969957bbdf8 csv     0
fetch service_requests_311_2025.csv 52c763f4496d4101ba0c20af33336a9c csv     0
fetch vacant_building_notices.csv   691d65a5f85640e6aaa46930bd9dc102 csv     1
fetch liquor_licenses.csv           ae5ed61365e74579aea25656ac9ce45e csv     0
fetch citiwatch_cameras.geojson     b676679350764f56a990c25a8ce2bda0 geojson 0

echo "done:"
ls -lh "$DIR"
