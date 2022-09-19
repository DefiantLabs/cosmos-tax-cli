package parsers

type Csv struct {
	Rows []CsvRow
}

type CsvRow interface {
	GetRowForCsv() []string
}

type CsvColumn interface {
	String() string
}
