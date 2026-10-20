package main

import (
	"fmt"
	"time"
)

type WbResult struct {
	records []WbRecord
	err     error
}

type GVTResult struct {
	records []GVTRecord
	err     error
}

type CVIResult struct {
	records []CVIRecord
	err     error
}

func LoadDataForConcurrency() ([]WbRecord, []GVTRecord, []CVIRecord) {
	beginningTime := time.Now()
	fmt.Println("Data is being process with concurrency, please wait")

	wbChannel := make(chan WbResult, 1)
	gvtChannel := make(chan GVTResult, 1)
	cviChannel := make(chan CVIResult, 1)

	go func() {
		records, err := LoadWbData()
		wbChannel <- WbResult{records: records, err: err}
	}()
	go func() {
		records, err := LoadGVTData()
		gvtChannel <- GVTResult{records: records, err: err}
	}()
	go func() {
		records, err := LoadCVIData()
		cviChannel <- CVIResult{records: records, err: err}
	}()

	resWb := <-wbChannel
	resGvt := <-gvtChannel
	resCvi := <-cviChannel

	timePassed := time.Since(beginningTime)
	fmt.Println("\nTime for execution: ", timePassed)

	var wbRecords []WbRecord
	var gvtRecords []GVTRecord
	var cviRecords []CVIRecord

	if resWb.err == nil {
		fmt.Println("The data has been loaded successfully! Records:", len(resWb.records))
		wbRecords = resWb.records
	} else {
		fmt.Println("There been an error loading the data for the public's well being:", resWb.err)
	}
	if resGvt.err == nil {
		fmt.Println("The data has been loaded successfully! Records:", len(resGvt.records))
		gvtRecords = resGvt.records
	} else {
		fmt.Println("There been an error loading the data for the gun violence tracer:", resGvt.err)
	}
	if resCvi.err == nil {
		fmt.Println("The data has been loaded successfully! Records:", len(resCvi.records))
		cviRecords = resCvi.records
	} else {
		fmt.Println("There been an error loading the data for the community violence intervention", resCvi.err)
	}

	return wbRecords, gvtRecords, cviRecords
}
