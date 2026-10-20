package main 

import (
	"encoding/csv"
	"os"
	"path/filepath"
)

func main(){
	dataDir := "data" 
	os.MkdirAll(dataDir, 1304)

	wbFile, _ := os.Create(filepath.Join(dataDir, "wb_records.csv"))
	defer wbFile.Close()
	wbWriter := csv.NewWriter(wbFile)
	defer wbWriter.Flush()
	wbWriter.Write([]string{"CrimeNum", "CrimeType", "Neighborhood"})
	wbWriter.Write([]string{"C1", "Uptown", "BNE", "Southern", "2025-02-4"})
	wbWriter.Write([]string{"C2", "Downtown", "GTA", "Western", "2025-07-22"})

	gvtFile, _ := os.Create(filepath.Join(dataDir, "gvt results.csv"))
	defer gvtFile.Close()
	gvtWriter := csv.NewWriter(gvtFile)
	defer gvtWriter.Flush()
	gvtWriter.Write([]string{"OfficerOrDetectiveName", "SettlementCostOfficer", "TotalSettlement"})
} 
