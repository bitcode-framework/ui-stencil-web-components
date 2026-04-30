package parser

import (
	"encoding/json"
	"testing"
)

func TestParseModel_ArraySource_Valid(t *testing.T) {
	data := `{
		"name": "currency",
		"source": "array",
		"fields": {
			"code": {"type": "string", "required": true},
			"name": {"type": "string"}
		},
		"rows": [
			{"code": "USD", "name": "US Dollar"},
			{"code": "EUR", "name": "Euro"}
		]
	}`

	model, err := ParseModel([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.Source != "array" {
		t.Errorf("expected source 'array', got %q", model.Source)
	}
	if len(model.DataRows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(model.DataRows))
	}
	if !model.IsArraySource() {
		t.Error("IsArraySource() should return true")
	}
	if model.IsWritable() {
		t.Error("IsWritable() should return false by default")
	}
	if !model.IsReadOnlySource() {
		t.Error("IsReadOnlySource() should return true")
	}
}

func TestParseModel_ArraySource_Writable(t *testing.T) {
	data := `{
		"name": "setting",
		"source": "array",
		"writable": true,
		"fields": {
			"key": {"type": "string", "required": true},
			"value": {"type": "text"}
		},
		"rows": [
			{"key": "app.name", "value": "Test"}
		]
	}`

	model, err := ParseModel([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !model.IsWritable() {
		t.Error("IsWritable() should return true")
	}
	if model.IsReadOnlySource() {
		t.Error("IsReadOnlySource() should return false for writable")
	}
}

func TestParseModel_ArraySource_RowsFile(t *testing.T) {
	data := `{
		"name": "country",
		"source": "array",
		"rows_file": "data/countries.json",
		"fields": {
			"code": {"type": "string", "required": true},
			"name": {"type": "string"}
		}
	}`

	model, err := ParseModel([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.RowsFile != "data/countries.json" {
		t.Errorf("expected rows_file 'data/countries.json', got %q", model.RowsFile)
	}
}

func TestParseModel_ArraySource_ErrorBothRowsAndFile(t *testing.T) {
	data := `{
		"name": "bad",
		"source": "array",
		"rows_file": "data/x.json",
		"fields": {"code": {"type": "string", "required": true}},
		"rows": [{"code": "X"}]
	}`

	_, err := ParseModel([]byte(data))
	if err == nil {
		t.Fatal("expected error for both rows and rows_file")
	}
}

func TestParseModel_ArraySource_ErrorNoData(t *testing.T) {
	data := `{
		"name": "empty",
		"source": "array",
		"fields": {"code": {"type": "string", "required": true}}
	}`

	_, err := ParseModel([]byte(data))
	if err == nil {
		t.Fatal("expected error for array source without rows/rows_file")
	}
}

func TestParseModel_ArraySource_ErrorUnsupportedFormat(t *testing.T) {
	data := `{
		"name": "bad_ext",
		"source": "array",
		"rows_file": "data/x.txt",
		"fields": {"code": {"type": "string", "required": true}}
	}`

	_, err := ParseModel([]byte(data))
	if err == nil {
		t.Fatal("expected error for unsupported file format")
	}
}

func TestParseModel_ArraySource_SyncSourceValidation(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "sync_source without writable",
			json: `{"name":"x","source":"array","sync_source":true,"rows_file":"d.json","fields":{"k":{"type":"string","required":true}}}`,
			wantErr: true,
		},
		{
			name: "sync_source without rows_file",
			json: `{"name":"x","source":"array","sync_source":true,"writable":true,"rows":[{"k":"v"}],"fields":{"k":{"type":"string","required":true}}}`,
			wantErr: true,
		},
		{
			name: "sync_source valid",
			json: `{"name":"x","source":"array","sync_source":true,"writable":true,"rows_file":"d.json","fields":{"k":{"type":"string","required":true}}}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseModel([]byte(tt.json))
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseModel_ProcessSource(t *testing.T) {
	data := `{
		"name": "exchange_rate",
		"source": "process",
		"process": "fetch_rates",
		"refresh": "1h",
		"fields": {
			"from": {"type": "string"},
			"to": {"type": "string"},
			"rate": {"type": "decimal"}
		}
	}`

	model, err := ParseModel([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !model.IsProcessSource() {
		t.Error("IsProcessSource() should return true")
	}
	if model.Process != "fetch_rates" {
		t.Errorf("expected process 'fetch_rates', got %q", model.Process)
	}
	if model.Refresh != "1h" {
		t.Errorf("expected refresh '1h', got %q", model.Refresh)
	}
}

func TestParseModel_ProcessSource_ErrorNoProcessOrScript(t *testing.T) {
	data := `{
		"name": "bad",
		"source": "process",
		"fields": {"x": {"type": "string"}}
	}`

	_, err := ParseModel([]byte(data))
	if err == nil {
		t.Fatal("expected error for process source without process/script")
	}
}

func TestParseModel_ReadOnlySource_DisablesTimestamps(t *testing.T) {
	data := `{
		"name": "fixture",
		"source": "array",
		"fields": {"code": {"type": "string", "required": true}},
		"rows": [{"code": "X"}]
	}`

	model, err := ParseModel([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.IsTimestamps() {
		t.Error("read-only array model should have timestamps disabled")
	}
	if model.IsSoftDeletes() {
		t.Error("read-only array model should have soft_deletes disabled")
	}
}

func TestParseModel_InvalidSource(t *testing.T) {
	data := `{
		"name": "bad",
		"source": "invalid",
		"fields": {"x": {"type": "string"}}
	}`

	_, err := ParseModel([]byte(data))
	if err == nil {
		t.Fatal("expected error for invalid source")
	}
}

func TestParseModel_GetEffectiveSource(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"", "db"},
		{"db", "db"},
		{"array", "array"},
		{"process", "process"},
	}

	for _, tt := range tests {
		m := &ModelDefinition{Source: tt.source}
		if got := m.GetEffectiveSource(); got != tt.expected {
			t.Errorf("source=%q: expected %q, got %q", tt.source, tt.expected, got)
		}
	}
}

func TestParseView_FilterBy_String(t *testing.T) {
	data := `{
		"name": "order_form",
		"type": "form",
		"model": "order",
		"layout": [
			{
				"tabs": [
					{"label": "Items", "view": "item_list", "filter_by": "order_id"}
				]
			}
		]
	}`

	view, err := ParseView([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(view.Layout) == 0 || len(view.Layout[0].Tabs) == 0 {
		t.Fatal("expected layout with tabs")
	}
	tab := view.Layout[0].Tabs[0]
	if tab.FilterBy != "order_id" {
		t.Errorf("expected FilterBy 'order_id', got %q", tab.FilterBy)
	}
}

func TestParseView_FilterBy_Map(t *testing.T) {
	data := `{
		"name": "order_form",
		"type": "form",
		"model": "order",
		"layout": [
			{
				"tabs": [
					{"label": "Active Items", "view": "item_list", "filter_by": {"order_id": "{record.id}", "status": "active"}}
				]
			}
		]
	}`

	view, err := ParseView([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tab := view.Layout[0].Tabs[0]
	if tab.FilterByMap == nil {
		t.Fatal("expected FilterByMap to be set")
	}
	if tab.FilterByMap["status"] != "active" {
		t.Errorf("expected status='active', got %v", tab.FilterByMap["status"])
	}
}

func TestParseView_DataSource_MutualExclusion(t *testing.T) {
	data := `{
		"name": "dashboard",
		"type": "custom",
		"template": "templates/dash.html",
		"data_sources": {
			"stats": {"model": "order", "process": "get_stats"}
		}
	}`

	_, err := ParseView([]byte(data))
	if err == nil {
		t.Fatal("expected error for data source with both model and process")
	}
}

func TestParseView_ViewModifiers(t *testing.T) {
	data := `{
		"name": "contact_form",
		"type": "form",
		"model": "contact",
		"layout": [
			{
				"row": [
					{"field": "name", "width": 6},
					{"field": "company", "width": 6, "visible_if": "type == 'business'", "css_class": "highlight"},
					{"field": "notes", "width": 12, "disabled_if": "status == 'closed'", "help_text": "Internal notes"}
				]
			}
		]
	}`

	view, err := ParseView([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	row := view.Layout[0].Row
	if row[1].VisibleIf != "type == 'business'" {
		t.Errorf("expected visible_if, got %q", row[1].VisibleIf)
	}
	if row[1].CSSClass != "highlight" {
		t.Errorf("expected css_class 'highlight', got %q", row[1].CSSClass)
	}
	if row[2].DisabledIf != "status == 'closed'" {
		t.Errorf("expected disabled_if, got %q", row[2].DisabledIf)
	}
	if row[2].HelpText != "Internal notes" {
		t.Errorf("expected help_text, got %q", row[2].HelpText)
	}
}

// Ensure JSON field name is "rows" not "data_rows"
func TestParseModel_RowsJSONFieldName(t *testing.T) {
	data := `{
		"name": "test",
		"source": "array",
		"fields": {"x": {"type": "string", "required": true}},
		"rows": [{"x": "hello"}]
	}`

	model, err := ParseModel([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(model.DataRows) != 1 {
		t.Errorf("expected 1 row, got %d", len(model.DataRows))
	}

	// Verify it marshals back correctly
	out, _ := json.Marshal(model)
	var raw map[string]json.RawMessage
	json.Unmarshal(out, &raw)
	if _, ok := raw["rows"]; !ok {
		t.Error("expected 'rows' key in JSON output")
	}
}
