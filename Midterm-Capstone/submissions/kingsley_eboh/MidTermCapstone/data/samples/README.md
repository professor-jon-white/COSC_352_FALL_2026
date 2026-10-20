# Bundled GeoJSON excerpts

These files are compact source excerpts for `go run . -offline`; they are not a representative statistical sample and should not be used for citywide conclusions.

- `311_sample.geojson`: first 250 neighborhood-labeled features returned in ascending `RowID` order from Baltimore City's 2026 311 service-request FeatureServer, retrieved October 11, 2026. Original feature attributes and geometry fields are retained.
- `vacant_rehabs_sample.geojson`: first 250 features from the city's Vacant Building Rehabs GeoJSON download, retrieved October 11, 2026. Original feature attributes and geometry fields are retained.

The default live run downloads the current query results instead. Source descriptions and links are in the project README.
