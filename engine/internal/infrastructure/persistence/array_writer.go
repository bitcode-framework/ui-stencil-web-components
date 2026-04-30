package persistence

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// WriteBackToFile reads all records from the DB table and writes them to the source file.
// Used when sync_source is enabled on writable array models.
func WriteBackToFile(db *gorm.DB, model *parser.ModelDefinition, tableName string, basePath string) error {
	if model.RowsFile == "" {
		return fmt.Errorf("write-back requires rows_file for model %q", model.Name)
	}

	fullPath := model.RowsFile
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(basePath, model.RowsFile)
	}

	var rows []map[string]any
	if err := db.Table(tableName).Find(&rows).Error; err != nil {
		return fmt.Errorf("write-back: failed to read records from %s: %w", tableName, err)
	}

	fieldNames := sortedFieldNames(model)

	ext := strings.ToLower(filepath.Ext(fullPath))
	switch ext {
	case ".json":
		return writeJSONRows(fullPath, rows, fieldNames)
	case ".csv":
		return writeCSVRows(fullPath, rows, fieldNames)
	case ".xlsx":
		return writeXLSXRows(fullPath, rows, fieldNames)
	case ".xml":
		return writeXMLRows(fullPath, rows, fieldNames)
	default:
		return fmt.Errorf("unsupported file format %q for write-back", ext)
	}
}

func sortedFieldNames(model *parser.ModelDefinition) []string {
	names := make([]string, 0, len(model.Fields))
	for name := range model.Fields {
		names = append(names, name)
	}
	sort.Strings(names)

	if model.PrimaryKey != nil && model.PrimaryKey.Field != "" {
		pk := model.PrimaryKey.Field
		if _, exists := model.Fields[pk]; !exists {
			names = append([]string{pk}, names...)
		}
	}
	return names
}

func writeJSONRows(path string, rows []map[string]any, fields []string) error {
	cleaned := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		clean := filterByFields(row, fields)
		cleaned = append(cleaned, clean)
	}

	data, err := json.MarshalIndent(cleaned, "", "  ")
	if err != nil {
		return fmt.Errorf("write-back JSON: marshal failed: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func writeCSVRows(path string, rows []map[string]any, fields []string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("write-back CSV: create file failed: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if err := writer.Write(fields); err != nil {
		return fmt.Errorf("write-back CSV: write header failed: %w", err)
	}

	for _, row := range rows {
		record := make([]string, len(fields))
		for i, field := range fields {
			if val, ok := row[field]; ok && val != nil {
				record[i] = fmt.Sprintf("%v", val)
			}
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write-back CSV: write row failed: %w", err)
		}
	}
	return nil
}

func writeXLSXRows(path string, rows []map[string]any, fields []string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"
	for i, field := range fields {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, field)
	}

	for rowIdx, row := range rows {
		for colIdx, field := range fields {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if val, ok := row[field]; ok && val != nil {
				f.SetCellValue(sheet, cell, val)
			}
		}
	}

	return f.SaveAs(path)
}

func writeXMLRows(path string, rows []map[string]any, fields []string) error {
	type xmlField struct {
		XMLName xml.Name
		Value   string `xml:",chardata"`
	}
	type xmlRow struct {
		XMLName xml.Name   `xml:"row"`
		Fields  []xmlField `xml:",any"`
	}
	type xmlRows struct {
		XMLName xml.Name `xml:"rows"`
		Rows    []xmlRow `xml:"row"`
	}

	container := xmlRows{}
	for _, row := range rows {
		xr := xmlRow{}
		for _, field := range fields {
			if val, ok := row[field]; ok && val != nil {
				xr.Fields = append(xr.Fields, xmlField{
					XMLName: xml.Name{Local: field},
					Value:   fmt.Sprintf("%v", val),
				})
			}
		}
		container.Rows = append(container.Rows, xr)
	}

	data, err := xml.MarshalIndent(container, "", "  ")
	if err != nil {
		return fmt.Errorf("write-back XML: marshal failed: %w", err)
	}

	header := []byte(xml.Header)
	return os.WriteFile(path, append(header, data...), 0644)
}

func filterByFields(row map[string]any, fields []string) map[string]any {
	clean := make(map[string]any, len(fields))
	for _, f := range fields {
		if val, ok := row[f]; ok {
			clean[f] = val
		}
	}
	if id, ok := row["id"]; ok {
		clean["id"] = id
	}
	return clean
}
