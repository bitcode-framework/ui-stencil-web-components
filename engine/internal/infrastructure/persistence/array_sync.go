package persistence

import (
	"fmt"
	"log"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"gorm.io/gorm"
)

// SyncArrayModel synchronizes array-backed model data into the database table.
// For read-only models: DELETE all + re-INSERT from source.
// For writable models: seed only if table is empty.
func SyncArrayModel(db *gorm.DB, model *parser.ModelDefinition, tableName string, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}

	if model.IsWritable() {
		var count int64
		if err := db.Table(tableName).Count(&count).Error; err != nil {
			return fmt.Errorf("array sync: failed to count rows in %s: %w", tableName, err)
		}
		if count > 0 {
			log.Printf("[ARRAY] %s: table has %d rows, skipping seed (writable mode)", model.Name, count)
			return nil
		}
		log.Printf("[ARRAY] %s: seeding %d rows (writable mode, table empty)", model.Name, len(rows))
		return bulkInsertRows(db, tableName, model, rows)
	}

	log.Printf("[ARRAY] %s: syncing %d rows (read-only mode)", model.Name, len(rows))
	if err := db.Table(tableName).Where("1=1").Delete(nil).Error; err != nil {
		return fmt.Errorf("array sync: failed to clear table %s: %w", tableName, err)
	}
	return bulkInsertRows(db, tableName, model, rows)
}

func bulkInsertRows(db *gorm.DB, tableName string, model *parser.ModelDefinition, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}

	batchSize := 100
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[i:end]

		filtered := make([]map[string]any, 0, len(batch))
		for _, row := range batch {
			clean := filterRowFields(row, model)
			filtered = append(filtered, clean)
		}

		if err := db.Table(tableName).Create(&filtered).Error; err != nil {
			return fmt.Errorf("array sync: failed to insert batch at offset %d into %s: %w", i, tableName, err)
		}
	}
	return nil
}

func filterRowFields(row map[string]any, model *parser.ModelDefinition) map[string]any {
	clean := make(map[string]any, len(row))
	for key, val := range row {
		if _, ok := model.Fields[key]; ok {
			clean[key] = val
			continue
		}
		if key == "id" {
			clean[key] = val
			continue
		}
		if model.PrimaryKey != nil && model.PrimaryKey.Field == key {
			clean[key] = val
			continue
		}
		log.Printf("WARN: array model %q: row has field %q not defined in fields", model.Name, key)
	}
	return clean
}
