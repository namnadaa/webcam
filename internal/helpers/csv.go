package helpers

import (
	"strconv"
	"strings"
)

// CSV files use ';' between columns and a decimal comma, so Excel with a
// Russian (comma-decimal) locale opens them as a table without an import dialog.
const (
	CSVSeparator = ";"
	CSVBOM       = "\uFEFF" // marks the file as UTF-8 for Excel
)

// CSVFloat formats a number with prec decimals and a decimal comma: 1.25 -> "1,25".
func CSVFloat(v float64, prec int) string {
	return strings.Replace(strconv.FormatFloat(v, 'f', prec, 64), ".", ",", 1)
}

// CSVRow joins fields with the separator and ends the line.
func CSVRow(fields ...string) string {
	return strings.Join(fields, CSVSeparator) + "\n"
}
