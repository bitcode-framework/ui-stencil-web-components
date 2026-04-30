package persistence

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

func LoadRowsFromFile(filePath string, basePath string) ([]map[string]any, error) {
	fullPath := filePath
	if !filepath.IsAbs(filePath) {
		fullPath = filepath.Join(basePath, filePath)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("rows_file %q not found (resolved: %s)", filePath, fullPath)
	}

	ext := strings.ToLower(filepath.Ext(fullPath))
	switch ext {
	case ".json":
		return parseJSONRows(fullPath)
	case ".csv":
		return parseCSVRows(fullPath)
	case ".xlsx":
		return parseXLSXRows(fullPath)
	case ".xml":
		return parseXMLRows(fullPath)
	default:
		return nil, fmt.Errorf("unsupported file format %q for rows_file", ext)
	}
}

func parseJSONRows(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file %s: %w", path, err)
	}

	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("failed to parse JSON rows from %s: %w", path, err)
	}
	return rows, nil
}

func parseCSVRows(path string) ([]map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header from %s: %w", path, err)
	}

	var rows []map[string]any
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row from %s: %w", path, err)
		}

		row := make(map[string]any, len(headers))
		for i, header := range headers {
			if i < len(record) {
				row[strings.TrimSpace(header)] = record[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func parseXLSXRows(path string) ([]map[string]any, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open XLSX file %s: %w", path, err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("XLSX file %s has no sheets", path)
	}

	xlsxRows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read XLSX rows from %s: %w", path, err)
	}
	if len(xlsxRows) < 2 {
		return []map[string]any{}, nil
	}

	headers := xlsxRows[0]
	var rows []map[string]any
	for _, xlsxRow := range xlsxRows[1:] {
		row := make(map[string]any, len(headers))
		for i, header := range headers {
			header = strings.TrimSpace(header)
			if header == "" {
				continue
			}
			if i < len(xlsxRow) {
				row[header] = xlsxRow[i]
			} else {
				row[header] = ""
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

type xmlRows struct {
	XMLName xml.Name  `xml:"rows"`
	Rows    []xmlRow  `xml:"row"`
}

type xmlRow struct {
	Fields []xmlField `xml:",any"`
}

type xmlField struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func parseXMLRows(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML file %s: %w", path, err)
	}

	var container xmlRows
	if err := xml.Unmarshal(data, &container); err != nil {
		return nil, fmt.Errorf("failed to parse XML rows from %s: %w", path, err)
	}

	rows := make([]map[string]any, 0, len(container.Rows))
	for _, xmlR := range container.Rows {
		row := make(map[string]any, len(xmlR.Fields))
		for _, field := range xmlR.Fields {
			row[field.XMLName.Local] = field.Value
		}
		rows = append(rows, row)
	}
	return rows, nil
}
