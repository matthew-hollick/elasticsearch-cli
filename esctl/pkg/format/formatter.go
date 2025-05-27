package format

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/olekukonko/tablewriter"
)

// Formatter handles formatting of tabular data
type Formatter struct {
	format string
	writer io.Writer
}

// New creates a new Formatter
func New(format string) *Formatter {
	return &Formatter{
		format: format,
		writer: os.Stdout,
	}
}

// SetWriter sets the output writer
func (f *Formatter) SetWriter(w io.Writer) {
	f.writer = w
}

// Write writes the data with the specified format
func (f *Formatter) Write(headers []string, rows [][]string) error {
	switch f.format {
	case "json":
		return f.writeJSON(headers, rows)
	case "csv":
		return f.writeCSV(headers, rows)
	case "plain":
		return f.writePlain(headers, rows)
	default: // rich
		return f.writeRich(headers, rows)
	}
}

func (f *Formatter) writeRich(headers []string, rows [][]string) error {
	table := tablewriter.NewWriter(f.writer)
	table.SetHeader(headers)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(rows)
	table.Render()
	return nil
}

func (f *Formatter) writePlain(headers []string, rows [][]string) error {
	w := tabwriter.NewWriter(f.writer, 0, 0, 1, ' ', 0)
	
	// Write headers
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	
	// Write rows
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	
	return w.Flush()
}

func (f *Formatter) writeCSV(headers []string, rows [][]string) error {
	w := csv.NewWriter(f.writer)
	if err := w.Write(headers); err != nil {
		return err
	}
	return w.WriteAll(rows)
}

func (f *Formatter) writeJSON(headers []string, rows [][]string) error {
	var result []map[string]string
	for _, row := range rows {
		item := make(map[string]string)
		for i, h := range headers {
			if i < len(row) {
				item[h] = row[i]
			}
		}
		result = append(result, item)
	}
	return json.NewEncoder(f.writer).Encode(result)
}
