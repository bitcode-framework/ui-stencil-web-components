package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/spf13/cobra"
)

func modelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Model management (array-backed models)",
	}

	cmd.AddCommand(modelSyncCmd())
	cmd.AddCommand(modelDiffCmd())
	cmd.AddCommand(modelListCmd())

	return cmd
}

func modelSyncCmd() *cobra.Command {
	var toFile bool
	var fromFile bool

	cmd := &cobra.Command{
		Use:   "sync [model_name]",
		Short: "Sync array model data between DB and file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelName := args[0]

			app, err := buildApp()
			if err != nil {
				return err
			}
			if err := app.LoadModules(); err != nil {
				return err
			}

			modelDef, err := app.ModelRegistry.Get(modelName)
			if err != nil {
				return fmt.Errorf("model %q not found", modelName)
			}
			if !modelDef.IsArraySource() {
				return fmt.Errorf("model %q is not an array source (source: %s)", modelName, modelDef.GetEffectiveSource())
			}
			if modelDef.RowsFile == "" {
				return fmt.Errorf("model %q has no rows_file configured", modelName)
			}

			tableName := app.ModelRegistry.TableName(modelName)

			if toFile {
				fmt.Printf("Syncing %s: DB → File (%s)...\n", modelName, modelDef.RowsFile)
				if err := persistence.WriteBackToFile(app.DB, modelDef, tableName, modelDef.ModulePath); err != nil {
					return fmt.Errorf("write-back failed: %w", err)
				}
				fmt.Println("Done.")
				return nil
			}

			if fromFile {
				fmt.Printf("Syncing %s: File (%s) → DB...\n", modelName, modelDef.RowsFile)
				rows, err := persistence.LoadRowsFromFile(modelDef.RowsFile, modelDef.ModulePath)
				if err != nil {
					return fmt.Errorf("failed to load file: %w", err)
				}
				if err := persistence.SyncArrayModel(app.DB, modelDef, tableName, rows); err != nil {
					return fmt.Errorf("sync failed: %w", err)
				}
				fmt.Printf("Done. %d rows synced.\n", len(rows))
				return nil
			}

			return fmt.Errorf("specify --to-file or --from-file")
		},
	}

	cmd.Flags().BoolVar(&toFile, "to-file", false, "Export DB data to source file")
	cmd.Flags().BoolVar(&fromFile, "from-file", false, "Import source file data to DB (overwrites)")

	return cmd
}

func modelDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff [model_name]",
		Short: "Show differences between DB and source file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelName := args[0]

			app, err := buildApp()
			if err != nil {
				return err
			}
			if err := app.LoadModules(); err != nil {
				return err
			}

			modelDef, err := app.ModelRegistry.Get(modelName)
			if err != nil {
				return fmt.Errorf("model %q not found", modelName)
			}
			if !modelDef.IsArraySource() {
				return fmt.Errorf("model %q is not an array source", modelName)
			}
			if modelDef.RowsFile == "" {
				return fmt.Errorf("model %q has no rows_file configured", modelName)
			}

			tableName := app.ModelRegistry.TableName(modelName)

			fileRows, err := persistence.LoadRowsFromFile(modelDef.RowsFile, modelDef.ModulePath)
			if err != nil {
				return fmt.Errorf("failed to load file: %w", err)
			}

			var dbRows []map[string]any
			if err := app.DB.Table(tableName).Find(&dbRows).Error; err != nil {
				return fmt.Errorf("failed to read DB: %w", err)
			}

			pkField := "id"
			if modelDef.PrimaryKey != nil && modelDef.PrimaryKey.Field != "" {
				pkField = modelDef.PrimaryKey.Field
			}

			fileMap := indexByPK(fileRows, pkField)
			dbMap := indexByPK(dbRows, pkField)

			var onlyInFile, onlyInDB, different []string

			for key := range fileMap {
				if _, ok := dbMap[key]; !ok {
					onlyInFile = append(onlyInFile, key)
				}
			}
			for key := range dbMap {
				if _, ok := fileMap[key]; !ok {
					onlyInDB = append(onlyInDB, key)
				}
			}
			for key, fileRow := range fileMap {
				if dbRow, ok := dbMap[key]; ok {
					if !rowsEqual(fileRow, dbRow) {
						different = append(different, key)
					}
				}
			}

			sort.Strings(onlyInFile)
			sort.Strings(onlyInDB)
			sort.Strings(different)

			if len(onlyInFile) == 0 && len(onlyInDB) == 0 && len(different) == 0 {
				fmt.Printf("Model %q: DB and file are in sync (%d records)\n", modelName, len(dbRows))
				return nil
			}

			fmt.Printf("Model %q diff:\n", modelName)
			if len(onlyInFile) > 0 {
				fmt.Printf("  Only in file (%d): %s\n", len(onlyInFile), strings.Join(onlyInFile, ", "))
			}
			if len(onlyInDB) > 0 {
				fmt.Printf("  Only in DB (%d): %s\n", len(onlyInDB), strings.Join(onlyInDB, ", "))
			}
			if len(different) > 0 {
				fmt.Printf("  Different (%d): %s\n", len(different), strings.Join(different, ", "))
			}
			fmt.Printf("  Total: file=%d, db=%d\n", len(fileRows), len(dbRows))

			return nil
		},
	}
}

func modelListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all array/process source models",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := buildApp()
			if err != nil {
				return err
			}
			if err := app.LoadModules(); err != nil {
				return err
			}

			models := app.ModelRegistry.List()
			found := false
			for _, m := range models {
				if m.IsArraySource() || m.IsProcessSource() {
					writable := "read-only"
					if m.IsWritable() {
						writable = "writable"
					}
					syncSource := ""
					if m.SyncSource {
						syncSource = " [sync_source]"
					}
					refreshInfo := ""
					if m.Refresh != "" {
						refreshInfo = fmt.Sprintf(" [refresh: %s]", m.Refresh)
					}
					fmt.Printf("  %-20s source=%-8s %s%s%s\n", m.Name, m.GetEffectiveSource(), writable, syncSource, refreshInfo)
					found = true
				}
			}
			if !found {
				fmt.Println("No array/process source models found.")
			}
			return nil
		},
	}
}

func indexByPK(rows []map[string]any, pkField string) map[string]map[string]any {
	result := make(map[string]map[string]any, len(rows))
	for _, row := range rows {
		if pk, ok := row[pkField]; ok {
			result[fmt.Sprintf("%v", pk)] = row
		}
	}
	return result
}

func rowsEqual(a, b map[string]any) bool {
	aJSON, _ := json.Marshal(normalizeRow(a))
	bJSON, _ := json.Marshal(normalizeRow(b))
	return string(aJSON) == string(bJSON)
}

func normalizeRow(row map[string]any) map[string]any {
	clean := make(map[string]any, len(row))
	for k, v := range row {
		switch k {
		case "created_at", "updated_at", "deleted_at", "created_by", "updated_by", "deleted_by":
			continue
		}
		clean[k] = v
	}
	return clean
}
