// Code generated from CSV.g4 by ANTLR 4.13.1. DO NOT EDIT.

package csv_paser // CSV

import "github.com/antlr4-go/antlr/v4"

// CSVListener is a complete listener for a parse tree produced by CSVParser.
type CSVListener interface {
	antlr.ParseTreeListener

	// EnterCsvFile is called when entering the csvFile production.
	EnterCsvFile(c *CsvFileContext)

	// EnterHdr is called when entering the hdr production.
	EnterHdr(c *HdrContext)

	// EnterRow is called when entering the row production.
	EnterRow(c *RowContext)

	// EnterField is called when entering the field production.
	EnterField(c *FieldContext)

	// ExitCsvFile is called when exiting the csvFile production.
	ExitCsvFile(c *CsvFileContext)

	// ExitHdr is called when exiting the hdr production.
	ExitHdr(c *HdrContext)

	// ExitRow is called when exiting the row production.
	ExitRow(c *RowContext)

	// ExitField is called when exiting the field production.
	ExitField(c *FieldContext)
}
