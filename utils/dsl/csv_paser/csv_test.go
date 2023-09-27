package csv_paser

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"os"
	"strings"
	"testing"
)

func TestCSVLexerInit(t *testing.T) {
	csvFile := os.Args[1]
	is, err := antlr.NewFileStream(csvFile)
	if err != nil {
		fmt.Printf("new file stream error: %s\n", err)
		return
	}

	// Create the Lexer
	lexer := NewCSVLexer(is)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	// Create the Parser
	p := NewCSVParser(stream)

	// Finally parse the expression
	l := &CSVMapListener{}
	antlr.ParseTreeWalkerDefault.Walk(l, p.CsvFile())
	fmt.Printf("%s\n", l.String())
}

type CSVMapListener struct {
	*BaseCSVListener
	headers []string
	cm      []map[string]string
	fields  []string // a slice of fields in current row
}

func (cl *CSVMapListener) lastHeader(header string) bool {
	return header == cl.headers[len(cl.headers)-1]
}

func (cl *CSVMapListener) String() string {
	var s strings.Builder
	s.WriteString("[")

	for i, m := range cl.cm {
		s.WriteString("{")
		for _, h := range cl.headers {
			s.WriteString(fmt.Sprintf("%s=%v", h, m[h]))
			if !cl.lastHeader(h) {
				s.WriteString(", ")
			}
		}
		s.WriteString("}")
		if i != len(cl.cm)-1 {
			s.WriteString(",\n")
			continue
		}
	}
	s.WriteString("]")
	return s.String()
}

/* for debug
func (this *CSVMapListener) EnterEveryRule(ctx antlr.ParserRuleContext) {
	fmt.Println(ctx.GetText())
}
*/

func (cl *CSVMapListener) ExitHdr(c *HdrContext) {
	cl.headers = cl.fields
}

func (cl *CSVMapListener) ExitField(c *FieldContext) {
	cl.fields = append(cl.fields, c.GetText())
}

func (cl *CSVMapListener) EnterRow(c *RowContext) {
	cl.fields = []string{} // create a new field slice
}

func (cl *CSVMapListener) ExitRow(c *RowContext) {
	// get the rule index of parent context
	if i, ok := c.GetParent().(antlr.RuleContext); ok {
		if i.GetRuleIndex() == CSVParserRULE_hdr {
			// ignore this row
			return
		}
	}

	// it is a data row
	m := map[string]string{}

	for i, h := range cl.headers {
		m[h] = cl.fields[i]
	}
	cl.cm = append(cl.cm, m)
}
