package main
type Result struct {
	Filename string
	Data map[string]string
	Err error
}
func loadConcurrent(filename string, column string, results chan Result) {
	data, err := loadData(filename, column)
	results <- Result {
		Filename: filename,
		Data: data,
		Err: err,
	}
}