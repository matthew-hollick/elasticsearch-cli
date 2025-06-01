package format

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// Formatter handles formatting of tabular data
type Formatter struct {
	format string
	writer io.Writer
	style  string // For fancy format style customization
}

// New creates a new Formatter
func New(format string) *Formatter {
	return &Formatter{
		format: format,
		writer: os.Stdout,
		style:  "dark", // Default style
	}
}

// NewWithStyle creates a new Formatter with a specific style for fancy output
func NewWithStyle(format string, style string) *Formatter {
	return &Formatter{
		format: format,
		writer: os.Stdout,
		style:  style,
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
	case "fancy":
		return f.writeFancy(headers, rows)
	default: // plain is now default
		return f.writePlain(headers, rows)
	}
}

// writeFancy writes the data in a fancy table format using go-pretty
func (f *Formatter) writeFancy(headers []string, rows [][]string) error {
	t := table.NewWriter()
	t.SetOutputMirror(f.writer)

	// Convert headers to table.Row
	headerRow := make(table.Row, len(headers))
	for i, h := range headers {
		headerRow[i] = h
	}
	t.AppendHeader(headerRow)

	// Convert data rows to table.Row
	for _, row := range rows {
		tableRow := make(table.Row, len(row))
		for i, cell := range row {
			tableRow[i] = cell
		}
		t.AppendRow(tableRow)
	}

	// Apply style based on user preference
	switch f.style {
	case "light":
		t.SetStyle(table.StyleLight)
		t.Style().Format.Header = text.FormatTitle
		t.Style().Options.DrawBorder = true
		t.Style().Options.SeparateRows = true
	case "double":
		t.SetStyle(table.StyleDouble)
		t.Style().Format.Header = text.FormatTitle
		t.Style().Options.DrawBorder = true
		t.Style().Options.SeparateRows = true
	case "bright":
		t.SetStyle(table.StyleColoredBright)
		t.Style().Format.Header = text.FormatTitle
		t.Style().Color.Header = text.Colors{text.BgHiGreen, text.FgHiWhite, text.Bold}
		t.Style().Options.DrawBorder = true
		t.Style().Options.SeparateRows = true
		t.Style().Options.SeparateColumns = true
	case "blue":
		t.SetStyle(table.StyleColoredBlueWhiteOnBlack)
		t.Style().Format.Header = text.FormatTitle
		t.Style().Options.DrawBorder = true
		t.Style().Options.SeparateRows = true
		t.Style().Options.SeparateColumns = true
	default: // "dark" is default
		t.SetStyle(table.StyleColoredDark)
		t.Style().Format.Header = text.FormatTitle
		t.Style().Color.Header = text.Colors{text.BgHiBlue, text.FgHiWhite, text.Bold}
		t.Style().Options.DrawBorder = true
		t.Style().Options.SeparateRows = true
		t.Style().Options.SeparateColumns = true
	}
	
	// Auto-size columns based on content
	t.SetAutoIndex(false)
	
	// Set column configurations for better readability
	configs := make([]table.ColumnConfig, 0, len(headers))
	for i := 0; i < len(headers); i++ {
		maxWidth := 40
		if i > 0 {
			maxWidth = 30
		}
		configs = append(configs, table.ColumnConfig{
			Number:    i + 1,
			AutoMerge: false,
			WidthMax:  maxWidth,
		})
	}
	t.SetColumnConfigs(configs)
	
	// Set title if available
	if len(headers) > 0 {
		t.SetTitle("Elasticsearch CLI - Results")
	}
	
	// Configure footer
	t.SetPageSize(20) // Paginate large results
	if len(rows) > 0 {
		t.SetCaption(fmt.Sprintf("Total: %d records", len(rows)))
	}

	// Render the table
	t.Render()
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
