package main
import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)
type CommunityData struct {
	Name string
	Crime float64
	Poverty float64
}
func loadData(filename string, valueColumn string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	headers := records[0]
	communityIndex := -1
	valueIndex := -1

	for i, column := range headers {
		if column == "CSA2020" {
			communityIndex = i
		}
		if column == valueColumn {
			valueIndex = i
		}
	}
	if communityIndex == -1 || valueIndex == -1{
		return nil, fmt.Errorf("required column not found")
	}
	data := make(map[string]string)
	for _, row := range records [1:] {
		community := row[communityIndex]
		value := row[valueIndex]
		data[community] = value
	}
	return data, nil
}
func convertNumber(value string)(float64, error) {
	return strconv.ParseFloat(value, 64)
}