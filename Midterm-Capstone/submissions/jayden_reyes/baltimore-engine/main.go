package main

import (
	"fmt"
)

func main() {
	fmt.Println("Processing data for program")

	wbRecords, gvtRecords, cviRecords := LoadDataForConcurrency()
	AnalysisOfData(wbRecords, gvtRecords, cviRecords)
	fmt.Println("Successfully loaded data for public's well being. Here are the records:", len(wbRecords))
	fmt.Println("Successfully loaded data for gun violence tracer. Here are the records:", len(gvtRecords))
	fmt.Println("Successfully loaded data for community violence intervention. Here are the records:", len(cviRecords))
	fmt.Println("Data Process Complete")
}
