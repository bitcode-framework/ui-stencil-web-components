package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/bitcode-framework/go-json/runtime"
	"github.com/expr-lang/expr/ast"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupExprIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.Exec(`CREATE TABLE "user" (
		id TEXT PRIMARY KEY,
		username TEXT,
		is_superuser INTEGER DEFAULT 0
	)`)
	sqlDB.Exec(`CREATE TABLE "group" (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE,
		display_name TEXT,
		category TEXT
	)`)
	sqlDB.Exec(`CREATE TABLE user_group (
		user_id TEXT,
		group_id TEXT,
		PRIMARY KEY (user_id, group_id)
	)`)
	sqlDB.Exec(`CREATE TABLE group_implies (
		group_id TEXT,
		implied_group_id TEXT,
		PRIMARY KEY (group_id, implied_group_id)
	)`)
	sqlDB.Exec(`CREATE TABLE record_rule (
		id TEXT PRIMARY KEY,
		name TEXT,
		model_name TEXT,
		group_names TEXT DEFAULT '',
		domain_filter TEXT,
		domain_filter_expr TEXT,
		can_read INTEGER DEFAULT 1,
		can_create INTEGER DEFAULT 1,
		can_write INTEGER DEFAULT 1,
		can_delete INTEGER DEFAULT 0,
		is_global INTEGER DEFAULT 0,
		active INTEGER DEFAULT 1,
		module TEXT,
		modified_source TEXT DEFAULT 'json'
	)`)
	sqlDB.Exec(`CREATE TABLE record_rule_groups (
		record_rule_id TEXT,
		group_id TEXT,
		PRIMARY KEY (record_rule_id, group_id)
	)`)
	sqlDB.Exec(`CREATE TABLE contact (
		id TEXT PRIMARY KEY,
		name TEXT,
		email TEXT,
		department_id TEXT,
		created_by TEXT,
		active INTEGER DEFAULT 1,
		deleted_at DATETIME
	)`)

	return db
}

func wireTestRuntime(t *testing.T) func() {
	t.Helper()
	origEval := runtimeEvalExprBool
	origParse := parseExprForRule

	SetRuntimeEvalExprBool(runtime.EvalExprBool)
	SetParseExprForRule(func(expression string) (ast.Node, error) {
		tree, err := runtime.ParseExpr(expression)
		if err != nil {
			return nil, err
		}
		return tree.Root, nil
	})

	return func() {
		runtimeEvalExprBool = origEval
		parseExprForRule = origParse
	}
}

func TestExprIntegration_GetExprFilters_SimpleEquality(t *testing.T) {
	cleanup := wireTestRuntime(t)
	defer cleanup()

	db := setupExprIntegrationDB(t)
	sqlDB, _ := db.DB()

	sqlDB.Exec(`INSERT INTO "user" (id, username) VALUES ('usr-42', 'alice')`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter_expr, can_read, active) VALUES ('r-1', 'own_records', 'contact', 'created_by == ctx.user_id', 1, 1)`)

	svc := NewRecordRuleService(db)
	svc.SetModelFieldsResolver(func(name string) []string {
		return []string{"id", "name", "email", "department_id", "created_by", "active"}
	})

	ruleCtx := &RecordRuleContext{
		UserID: "usr-42",
		Now:    time.Now(),
		Today:  time.Now().Format("2006-01-02"),
	}

	clauses, err := svc.GetExprFilters("usr-42", "contact", "read", ruleCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) == 0 {
		t.Fatal("expected at least 1 clause")
	}

	c := clauses[0].Condition
	if c == nil {
		t.Fatal("expected leaf condition")
	}
	if c.Field != "created_by" || c.Operator != "=" || c.Value != "usr-42" {
		t.Errorf("got field=%s op=%s value=%v, want created_by = usr-42", c.Field, c.Operator, c.Value)
	}
}

func TestExprIntegration_GetExprFilters_SuperuserBypass(t *testing.T) {
	cleanup := wireTestRuntime(t)
	defer cleanup()

	db := setupExprIntegrationDB(t)
	sqlDB, _ := db.DB()

	sqlDB.Exec(`INSERT INTO "user" (id, username, is_superuser) VALUES ('admin-1', 'admin', 1)`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter_expr, can_read, active) VALUES ('r-1', 'strict', 'contact', 'created_by == ctx.user_id', 1, 1)`)

	svc := NewRecordRuleService(db)
	ruleCtx := &RecordRuleContext{UserID: "admin-1"}

	clauses, err := svc.GetExprFilters("admin-1", "contact", "read", ruleCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clauses != nil {
		t.Errorf("expected nil for superuser, got %v", clauses)
	}
}

func TestExprIntegration_GetExprFilters_OperationFilter(t *testing.T) {
	cleanup := wireTestRuntime(t)
	defer cleanup()

	db := setupExprIntegrationDB(t)
	sqlDB, _ := db.DB()

	sqlDB.Exec(`INSERT INTO "user" (id, username) VALUES ('usr-1', 'bob')`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter_expr, can_read, can_write, can_create, can_delete, active) VALUES ('r-1', 'read_only', 'contact', 'created_by == ctx.user_id', 1, 0, 0, 0, 1)`)

	svc := NewRecordRuleService(db)
	svc.SetModelFieldsResolver(func(name string) []string {
		return []string{"id", "name", "email", "created_by"}
	})

	ruleCtx := &RecordRuleContext{UserID: "usr-1"}

	readClauses, _ := svc.GetExprFilters("usr-1", "contact", "read", ruleCtx)
	if len(readClauses) == 0 {
		t.Error("expected clauses for read operation")
	}

	writeClauses, _ := svc.GetExprFilters("usr-1", "contact", "write", ruleCtx)
	if len(writeClauses) != 0 {
		t.Errorf("expected no clauses for write operation (can_write=0), got %d", len(writeClauses))
	}
}

func TestExprIntegration_GetExprFilters_InClause(t *testing.T) {
	cleanup := wireTestRuntime(t)
	defer cleanup()

	db := setupExprIntegrationDB(t)
	sqlDB, _ := db.DB()

	sqlDB.Exec(`INSERT INTO "user" (id, username) VALUES ('usr-1', 'carol')`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter_expr, can_read, active) VALUES ('r-1', 'dept_filter', 'contact', 'department_id in ctx.department_ids', 1, 1)`)

	svc := NewRecordRuleService(db)
	svc.SetModelFieldsResolver(func(name string) []string {
		return []string{"id", "name", "department_id", "created_by"}
	})

	ruleCtx := &RecordRuleContext{
		UserID:        "usr-1",
		DepartmentIDs: []string{"dept-1", "dept-2", "dept-3"},
	}

	clauses, err := svc.GetExprFilters("usr-1", "contact", "read", ruleCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) == 0 {
		t.Fatal("expected clauses")
	}
	c := clauses[0].Condition
	if c == nil || c.Operator != "in" {
		t.Errorf("expected 'in' operator, got %+v", c)
	}
	arr, ok := c.Value.([]any)
	if !ok || len(arr) != 3 {
		t.Errorf("expected 3 values, got %v", c.Value)
	}
}

func TestExprIntegration_GetExprFilters_EmptyDepartments_DenyAll(t *testing.T) {
	cleanup := wireTestRuntime(t)
	defer cleanup()

	db := setupExprIntegrationDB(t)
	sqlDB, _ := db.DB()

	sqlDB.Exec(`INSERT INTO "user" (id, username) VALUES ('usr-1', 'dave')`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter_expr, can_read, active) VALUES ('r-1', 'dept_filter', 'contact', 'department_id in ctx.department_ids', 1, 1)`)

	svc := NewRecordRuleService(db)
	svc.SetModelFieldsResolver(func(name string) []string {
		return []string{"id", "name", "department_id", "created_by"}
	})

	ruleCtx := &RecordRuleContext{
		UserID:        "usr-1",
		DepartmentIDs: []string{},
	}

	clauses, err := svc.GetExprFilters("usr-1", "contact", "read", ruleCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) == 0 {
		t.Fatal("expected deny-all clause")
	}
	c := clauses[0].Condition
	if c == nil || c.Field != "1" || c.Value != 0 {
		t.Errorf("expected deny-all (1=0), got %+v", c)
	}
}

func TestExprIntegration_LegacyAndExprCoexist(t *testing.T) {
	cleanup := wireTestRuntime(t)
	defer cleanup()

	db := setupExprIntegrationDB(t)
	sqlDB, _ := db.DB()

	sqlDB.Exec(`INSERT INTO "user" (id, username) VALUES ('usr-1', 'eve')`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter, can_read, active) VALUES ('r-legacy', 'legacy_rule', 'contact', '[["created_by","=","{{user.id}}"]]', 1, 1)`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter_expr, can_read, active) VALUES ('r-expr', 'expr_rule', 'contact', 'department_id in ctx.department_ids', 1, 1)`)

	svc := NewRecordRuleService(db)
	svc.SetModelFieldsResolver(func(name string) []string {
		return []string{"id", "name", "department_id", "created_by"}
	})

	legacyFilters, err := svc.GetFilters("usr-1", "contact", "read")
	if err != nil {
		t.Fatalf("GetFilters error: %v", err)
	}
	if len(legacyFilters) == 0 {
		t.Error("expected legacy filters")
	}

	ruleCtx := &RecordRuleContext{
		UserID:        "usr-1",
		DepartmentIDs: []string{"dept-1"},
	}
	exprFilters, err := svc.GetExprFilters("usr-1", "contact", "read", ruleCtx)
	if err != nil {
		t.Fatalf("GetExprFilters error: %v", err)
	}
	if len(exprFilters) == 0 {
		t.Error("expected expr filters")
	}
}

func TestExprIntegration_ApplyFiltersToQuery(t *testing.T) {
	cleanup := wireTestRuntime(t)
	defer cleanup()

	db := setupExprIntegrationDB(t)
	sqlDB, _ := db.DB()

	sqlDB.Exec(`INSERT INTO "user" (id, username) VALUES ('usr-42', 'frank')`)
	sqlDB.Exec(`INSERT INTO contact (id, name, created_by, active) VALUES ('c-1', 'Alice', 'usr-42', 1)`)
	sqlDB.Exec(`INSERT INTO contact (id, name, created_by, active) VALUES ('c-2', 'Bob', 'usr-99', 1)`)
	sqlDB.Exec(`INSERT INTO contact (id, name, created_by, active) VALUES ('c-3', 'Carol', 'usr-42', 1)`)

	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter_expr, can_read, active) VALUES ('r-1', 'own_records', 'contact', 'created_by == ctx.user_id', 1, 1)`)

	svc := NewRecordRuleService(db)
	svc.SetModelFieldsResolver(func(name string) []string {
		return []string{"id", "name", "email", "department_id", "created_by", "active"}
	})

	ruleCtx := &RecordRuleContext{UserID: "usr-42"}
	clauses, err := svc.GetExprFilters("usr-42", "contact", "read", ruleCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo := NewGenericRepository(db, "contact")
	query := NewQuery()
	query.WhereClauses = append(query.WhereClauses, clauses...)
	query.Where("active", "=", 1)

	results, total, err := repo.FindAll(context.Background(), query, 1, 100)
	if err != nil {
		t.Fatalf("FindAll error: %v", err)
	}

	if total != 2 {
		t.Errorf("expected 2 records (owned by usr-42), got %d", total)
		for _, r := range results {
			t.Logf("  record: id=%v name=%v created_by=%v", r["id"], r["name"], r["created_by"])
		}
	}
}

func TestExprIntegration_ComplexNestedCondition(t *testing.T) {
	cleanup := wireTestRuntime(t)
	defer cleanup()

	db := setupExprIntegrationDB(t)
	sqlDB, _ := db.DB()

	sqlDB.Exec(`INSERT INTO "user" (id, username) VALUES ('usr-1', 'grace')`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter_expr, can_read, active) VALUES ('r-1', 'complex', 'contact', 'created_by == ctx.user_id || department_id in ctx.department_ids', 1, 1)`)

	svc := NewRecordRuleService(db)
	svc.SetModelFieldsResolver(func(name string) []string {
		return []string{"id", "name", "department_id", "created_by"}
	})

	ruleCtx := &RecordRuleContext{
		UserID:        "usr-1",
		DepartmentIDs: []string{"dept-a", "dept-b"},
	}

	clauses, err := svc.GetExprFilters("usr-1", "contact", "read", ruleCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) == 0 {
		t.Fatal("expected clauses")
	}

	g := clauses[0].Group
	if g == nil {
		t.Fatal("expected OR group at top level")
	}
	if g.Connector != ConnectorOr {
		t.Errorf("expected OR, got %s", g.Connector)
	}
}
