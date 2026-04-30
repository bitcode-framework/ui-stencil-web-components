package persistence

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupArrayTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

func createArrayTestTable(t *testing.T, db *gorm.DB, tableName string) {
	t.Helper()
	sql := "CREATE TABLE IF NOT EXISTS " + tableName + " (id TEXT PRIMARY KEY, code TEXT, name TEXT)"
	if err := db.Exec(sql).Error; err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
}

func arrayBoolPtr(b bool) *bool { return &b }

func TestSyncArrayModel_ReadOnly(t *testing.T) {
	db := setupArrayTestDB(t)
	createArrayTestTable(t, db, "currencies")

	model := &parser.ModelDefinition{
		Name:   "currency",
		Source: "array",
		Fields: map[string]parser.FieldDefinition{
			"code": {Type: parser.FieldString},
			"name": {Type: parser.FieldString},
		},
	}

	rows := []map[string]any{
		{"id": "1", "code": "USD", "name": "US Dollar"},
		{"id": "2", "code": "EUR", "name": "Euro"},
	}

	if err := SyncArrayModel(db, model, "currencies", rows); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	var count int64
	db.Table("currencies").Count(&count)
	if count != 2 {
		t.Errorf("expected 2 rows, got %d", count)
	}

	rows2 := []map[string]any{
		{"id": "1", "code": "USD", "name": "US Dollar"},
		{"id": "2", "code": "EUR", "name": "Euro"},
		{"id": "3", "code": "GBP", "name": "British Pound"},
	}

	if err := SyncArrayModel(db, model, "currencies", rows2); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	db.Table("currencies").Count(&count)
	if count != 3 {
		t.Errorf("expected 3 rows after re-sync, got %d", count)
	}
}

func TestSyncArrayModel_Writable_SeedOnlyIfEmpty(t *testing.T) {
	db := setupArrayTestDB(t)
	createArrayTestTable(t, db, "settings")

	model := &parser.ModelDefinition{
		Name:     "setting",
		Source:   "array",
		Writable: arrayBoolPtr(true),
		Fields: map[string]parser.FieldDefinition{
			"code": {Type: parser.FieldString},
			"name": {Type: parser.FieldString},
		},
	}

	rows := []map[string]any{
		{"id": "1", "code": "key1", "name": "Value 1"},
		{"id": "2", "code": "key2", "name": "Value 2"},
	}

	if err := SyncArrayModel(db, model, "settings", rows); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	var count int64
	db.Table("settings").Count(&count)
	if count != 2 {
		t.Errorf("expected 2 rows after seed, got %d", count)
	}

	db.Table("settings").Create(map[string]any{"id": "3", "code": "key3", "name": "User Added"})

	rows2 := []map[string]any{
		{"id": "1", "code": "key1", "name": "Changed"},
	}
	if err := SyncArrayModel(db, model, "settings", rows2); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	db.Table("settings").Count(&count)
	if count != 3 {
		t.Errorf("expected 3 rows (writable mode should not re-sync), got %d", count)
	}
}

func TestSyncArrayModel_EmptyRows(t *testing.T) {
	db := setupArrayTestDB(t)
	createArrayTestTable(t, db, "empty_model")

	model := &parser.ModelDefinition{
		Name:   "empty",
		Source: "array",
		Fields: map[string]parser.FieldDefinition{
			"code": {Type: parser.FieldString},
		},
	}

	if err := SyncArrayModel(db, model, "empty_model", nil); err != nil {
		t.Fatalf("sync with nil rows should not error: %v", err)
	}

	var count int64
	db.Table("empty_model").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 rows, got %d", count)
	}
}

func TestLoadRowsFromFile_JSON(t *testing.T) {
	dir := t.TempDir()
	jsonFile := filepath.Join(dir, "data.json")
	content := `[{"code":"USD","name":"US Dollar"},{"code":"EUR","name":"Euro"}]`
	os.WriteFile(jsonFile, []byte(content), 0644)

	rows, err := LoadRowsFromFile("data.json", dir)
	if err != nil {
		t.Fatalf("failed to load JSON: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["code"] != "USD" {
		t.Errorf("expected code=USD, got %v", rows[0]["code"])
	}
}

func TestLoadRowsFromFile_CSV(t *testing.T) {
	dir := t.TempDir()
	csvFile := filepath.Join(dir, "data.csv")
	content := "code,name\nUSD,US Dollar\nEUR,Euro\n"
	os.WriteFile(csvFile, []byte(content), 0644)

	rows, err := LoadRowsFromFile("data.csv", dir)
	if err != nil {
		t.Fatalf("failed to load CSV: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["code"] != "USD" {
		t.Errorf("expected code=USD, got %v", rows[0]["code"])
	}
	if rows[1]["name"] != "Euro" {
		t.Errorf("expected name=Euro, got %v", rows[1]["name"])
	}
}

func TestLoadRowsFromFile_XML(t *testing.T) {
	dir := t.TempDir()
	xmlFile := filepath.Join(dir, "data.xml")
	content := `<rows><row><code>USD</code><name>US Dollar</name></row><row><code>EUR</code><name>Euro</name></row></rows>`
	os.WriteFile(xmlFile, []byte(content), 0644)

	rows, err := LoadRowsFromFile("data.xml", dir)
	if err != nil {
		t.Fatalf("failed to load XML: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["code"] != "USD" {
		t.Errorf("expected code=USD, got %v", rows[0]["code"])
	}
}

func TestLoadRowsFromFile_NotFound(t *testing.T) {
	_, err := LoadRowsFromFile("nonexistent.json", "/tmp")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadRowsFromFile_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.txt"), []byte("hello"), 0644)

	_, err := LoadRowsFromFile("data.txt", dir)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestWriteBackToFile_JSON(t *testing.T) {
	db := setupArrayTestDB(t)
	createArrayTestTable(t, db, "wb_test")

	db.Table("wb_test").Create(map[string]any{"id": "1", "code": "USD", "name": "US Dollar"})
	db.Table("wb_test").Create(map[string]any{"id": "2", "code": "EUR", "name": "Euro"})

	dir := t.TempDir()
	model := &parser.ModelDefinition{
		Name:     "wb_test",
		Source:   "array",
		Writable: arrayBoolPtr(true),
		RowsFile: "output.json",
		Fields: map[string]parser.FieldDefinition{
			"code": {Type: parser.FieldString},
			"name": {Type: parser.FieldString},
		},
	}

	if err := WriteBackToFile(db, model, "wb_test", dir); err != nil {
		t.Fatalf("write-back failed: %v", err)
	}

	outPath := filepath.Join(dir, "output.json")
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatal("output file not created")
	}

	rows, err := LoadRowsFromFile("output.json", dir)
	if err != nil {
		t.Fatalf("failed to re-read output: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows in output, got %d", len(rows))
	}
}

func TestEagerLoading_WithConditions(t *testing.T) {
	db := setupArrayTestDB(t)

	db.Exec("CREATE TABLE posts (id TEXT PRIMARY KEY, title TEXT)")
	db.Exec("CREATE TABLE comments (id TEXT PRIMARY KEY, post_id TEXT, status TEXT, body TEXT)")

	db.Table("posts").Create(map[string]any{"id": "p1", "title": "Post 1"})
	db.Table("comments").Create(map[string]any{"id": "c1", "post_id": "p1", "status": "approved", "body": "Good"})
	db.Table("comments").Create(map[string]any{"id": "c2", "post_id": "p1", "status": "pending", "body": "Meh"})
	db.Table("comments").Create(map[string]any{"id": "c3", "post_id": "p1", "status": "approved", "body": "Great"})

	modelDef := &parser.ModelDefinition{
		Name: "post",
		Fields: map[string]parser.FieldDefinition{
			"title":    {Type: parser.FieldString},
			"comments": {Type: parser.FieldOne2Many, Model: "comment", Inverse: "post_id"},
		},
	}

	repo := NewGenericRepositoryWithModel(db, "posts", modelDef)
	repo.SetTableNameResolver(&staticTableResolver{
		tables: map[string]string{"post": "posts", "comment": "comments"},
	})

	query := NewQuery()
	query.With = []WithClause{
		{
			Relation: "comments",
			Conditions: []Condition{
				{Field: "status", Operator: "=", Value: "approved"},
			},
		},
	}

	results, _, err := repo.FindAll(context.Background(), query, 1, 10)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 post, got %d", len(results))
	}

	comments, ok := results[0]["_comments"].([]map[string]any)
	if !ok {
		t.Fatal("expected _comments to be []map[string]any")
	}
	if len(comments) != 2 {
		t.Errorf("expected 2 approved comments, got %d", len(comments))
	}
	for _, c := range comments {
		if c["status"] != "approved" {
			t.Errorf("expected status=approved, got %v", c["status"])
		}
	}
}

func TestEagerLoading_WithLimit(t *testing.T) {
	db := setupArrayTestDB(t)

	db.Exec("CREATE TABLE authors (id TEXT PRIMARY KEY, name TEXT)")
	db.Exec("CREATE TABLE books (id TEXT PRIMARY KEY, author_id TEXT, title TEXT)")

	db.Table("authors").Create(map[string]any{"id": "a1", "name": "Author 1"})
	for i := 1; i <= 5; i++ {
		db.Table("books").Create(map[string]any{"id": "b" + string(rune('0'+i)), "author_id": "a1", "title": "Book " + string(rune('0'+i))})
	}

	modelDef := &parser.ModelDefinition{
		Name: "author",
		Fields: map[string]parser.FieldDefinition{
			"name":  {Type: parser.FieldString},
			"books": {Type: parser.FieldOne2Many, Model: "book", Inverse: "author_id"},
		},
	}

	repo := NewGenericRepositoryWithModel(db, "authors", modelDef)
	repo.SetTableNameResolver(&staticTableResolver{
		tables: map[string]string{"author": "authors", "book": "books"},
	})

	query := NewQuery()
	query.With = []WithClause{
		{
			Relation: "books",
			Limit:    2,
		},
	}

	results, _, err := repo.FindAll(context.Background(), query, 1, 10)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	books, ok := results[0]["_books"].([]map[string]any)
	if !ok {
		t.Fatal("expected _books to be []map[string]any")
	}
	if len(books) != 2 {
		t.Errorf("expected 2 books (limit), got %d", len(books))
	}
}

func TestEagerLoading_Many2One(t *testing.T) {
	db := setupArrayTestDB(t)

	db.Exec("CREATE TABLE categories (id TEXT PRIMARY KEY, name TEXT)")
	db.Exec("CREATE TABLE products (id TEXT PRIMARY KEY, category_id TEXT, name TEXT)")

	db.Table("categories").Create(map[string]any{"id": "cat1", "name": "Electronics"})
	db.Table("products").Create(map[string]any{"id": "p1", "category_id": "cat1", "name": "Phone"})

	modelDef := &parser.ModelDefinition{
		Name: "product",
		Fields: map[string]parser.FieldDefinition{
			"name":        {Type: parser.FieldString},
			"category_id": {Type: parser.FieldMany2One, Model: "category"},
		},
	}

	repo := NewGenericRepositoryWithModel(db, "products", modelDef)
	repo.SetTableNameResolver(&staticTableResolver{
		tables: map[string]string{"product": "products", "category": "categories"},
	})

	query := NewQuery()
	query.With = []WithClause{{Relation: "category_id"}}

	results, _, err := repo.FindAll(context.Background(), query, 1, 10)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	cat, ok := results[0]["_category_id"].(map[string]any)
	if !ok {
		t.Fatal("expected _category_id to be map[string]any")
	}
	if cat["name"] != "Electronics" {
		t.Errorf("expected category name=Electronics, got %v", cat["name"])
	}
}

type staticTableResolver struct {
	tables map[string]string
}

func (r *staticTableResolver) TableName(name string) string {
	if t, ok := r.tables[name]; ok {
		return t
	}
	return name
}

func (r *staticTableResolver) Get(name string) (*parser.ModelDefinition, error) {
	return nil, nil
}
