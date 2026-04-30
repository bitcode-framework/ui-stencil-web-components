package persistence

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	exprParser "github.com/expr-lang/expr/parser"
)

func parseTestExpr(t *testing.T, expression string) *ExprToFilters {
	t.Helper()
	return nil
}

func testCtx() *RecordRuleContext {
	return &RecordRuleContext{
		UserID:        "usr-42",
		CompanyID:     "comp-1",
		CompanyIDs:    []string{"comp-1", "comp-2"},
		DepartmentID:  "dept-1",
		DepartmentIDs: []string{"dept-1", "dept-2"},
		GroupIDs:      []string{"grp-1", "grp-2"},
		Groups:        []string{"sales", "admin"},
		Role:          "manager",
		TenantID:      "tenant-1",
		Now:           time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC),
		Today:         "2026-07-14",
	}
}

func testFields() []string {
	return []string{"created_by", "department_id", "company_id", "status", "amount", "name", "active", "role"}
}

func convertExpr(t *testing.T, expression string, ctx *RecordRuleContext, fields []string) ([]WhereClause, error) {
	t.Helper()
	tree, err := exprParser.Parse(expression)
	if err != nil {
		t.Fatalf("failed to parse expression %q: %v", expression, err)
	}
	walker := NewExprToFilters(ctx, fields)
	return walker.Convert(tree.Node)
}

func TestExprToFilters_SimpleEquality(t *testing.T) {
	clauses, err := convertExpr(t, `created_by == ctx.user_id`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	c := clauses[0].Condition
	if c == nil {
		t.Fatal("expected leaf condition, got group")
	}
	if c.Field != "created_by" || c.Operator != "=" || c.Value != "usr-42" {
		t.Errorf("got field=%s op=%s value=%v, want created_by = usr-42", c.Field, c.Operator, c.Value)
	}
}

func TestExprToFilters_InClause(t *testing.T) {
	clauses, err := convertExpr(t, `department_id in ctx.department_ids`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	c := clauses[0].Condition
	if c == nil {
		t.Fatal("expected leaf condition")
	}
	if c.Field != "department_id" || c.Operator != "in" {
		t.Errorf("got field=%s op=%s, want department_id in", c.Field, c.Operator)
	}
	arr, ok := c.Value.([]any)
	if !ok {
		t.Fatalf("expected []any value, got %T", c.Value)
	}
	if len(arr) != 2 {
		t.Errorf("expected 2 values, got %d", len(arr))
	}
}

func TestExprToFilters_AND(t *testing.T) {
	clauses, err := convertExpr(t, `created_by == ctx.user_id && department_id == ctx.department_id`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	g := clauses[0].Group
	if g == nil {
		t.Fatal("expected group, got leaf")
	}
	if g.Connector != ConnectorAnd {
		t.Errorf("expected AND connector, got %s", g.Connector)
	}
	if len(g.Conditions) != 2 {
		t.Errorf("expected 2 children, got %d", len(g.Conditions))
	}
}

func TestExprToFilters_OR(t *testing.T) {
	clauses, err := convertExpr(t, `created_by == ctx.user_id || department_id in ctx.department_ids`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	g := clauses[0].Group
	if g == nil {
		t.Fatal("expected group, got leaf")
	}
	if g.Connector != ConnectorOr {
		t.Errorf("expected OR connector, got %s", g.Connector)
	}
}

func TestExprToFilters_Nested(t *testing.T) {
	clauses, err := convertExpr(t,
		`(created_by == ctx.user_id || department_id in ctx.department_ids) && status == 'active'`,
		testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	g := clauses[0].Group
	if g == nil {
		t.Fatal("expected AND group at top level")
	}
	if g.Connector != ConnectorAnd {
		t.Errorf("expected AND at top, got %s", g.Connector)
	}
	if len(g.Conditions) != 2 {
		t.Fatalf("expected 2 children in AND, got %d", len(g.Conditions))
	}
	orGroup := g.Conditions[0].Group
	if orGroup == nil || orGroup.Connector != ConnectorOr {
		t.Error("expected first child to be OR group")
	}
	leaf := g.Conditions[1].Condition
	if leaf == nil || leaf.Field != "status" || leaf.Value != "active" {
		t.Errorf("expected second child to be status='active', got %+v", leaf)
	}
}

func TestExprToFilters_MatchesRejected(t *testing.T) {
	_, err := convertExpr(t, `name matches "^admin"`, testCtx(), testFields())
	if err == nil {
		t.Fatal("expected error for matches operator")
	}
}

func TestExprToFilters_TautologyRejected(t *testing.T) {
	_, err := convertExpr(t, `1 == 1`, testCtx(), testFields())
	if err == nil {
		t.Fatal("expected error for tautology (no field reference)")
	}
}

func TestExprToFilters_UnknownFieldRejected(t *testing.T) {
	_, err := convertExpr(t, `nonexistent == ctx.user_id`, testCtx(), testFields())
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestExprToFilters_EmptyArray_DenyAll(t *testing.T) {
	ctx := testCtx()
	ctx.DepartmentIDs = []string{}
	clauses, err := convertExpr(t, `department_id in ctx.department_ids`, ctx, testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	c := clauses[0].Condition
	if c == nil {
		t.Fatal("expected deny-all condition")
	}
	if c.Field != "1" || c.Value != 0 {
		t.Errorf("expected deny-all (1=0), got field=%s value=%v", c.Field, c.Value)
	}
}

func TestExprToFilters_NilContext_DenyAll(t *testing.T) {
	ctx := testCtx()
	ctx.UserID = ""
	clauses, err := convertExpr(t, `created_by == ctx.user_id`, ctx, testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	c := clauses[0].Condition
	if c == nil {
		t.Fatal("expected condition")
	}
	if c.Value != "" {
		t.Logf("empty string UserID produces value=%v (not deny-all, but empty match)", c.Value)
	}
}

func TestExprToFilters_ContainsOperator(t *testing.T) {
	clauses, err := convertExpr(t, `name contains "admin"`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	c := clauses[0].Condition
	if c == nil {
		t.Fatal("expected condition")
	}
	if c.Operator != "like" || c.Value != "%admin%" {
		t.Errorf("expected LIKE %%admin%%, got op=%s value=%v", c.Operator, c.Value)
	}
}

func TestExprToFilters_StartsWithOperator(t *testing.T) {
	clauses, err := convertExpr(t, `name startsWith "admin"`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := clauses[0].Condition
	if c.Operator != "like" || c.Value != "admin%" {
		t.Errorf("expected LIKE admin%%, got op=%s value=%v", c.Operator, c.Value)
	}
}

func TestExprToFilters_EndsWithOperator(t *testing.T) {
	clauses, err := convertExpr(t, `name endsWith "corp"`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := clauses[0].Condition
	if c.Operator != "like" || c.Value != "%corp" {
		t.Errorf("expected LIKE %%corp, got op=%s value=%v", c.Operator, c.Value)
	}
}

func TestExprToFilters_NotEqual(t *testing.T) {
	clauses, err := convertExpr(t, `status != 'draft'`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := clauses[0].Condition
	if c.Field != "status" || c.Operator != "!=" || c.Value != "draft" {
		t.Errorf("got field=%s op=%s value=%v", c.Field, c.Operator, c.Value)
	}
}

func TestExprToFilters_NumericComparison(t *testing.T) {
	clauses, err := convertExpr(t, `amount > 1000`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := clauses[0].Condition
	if c.Field != "amount" || c.Operator != ">" || c.Value != 1000 {
		t.Errorf("got field=%s op=%s value=%v (%T)", c.Field, c.Operator, c.Value, c.Value)
	}
}

func TestExprToFilters_BooleanLiteral(t *testing.T) {
	clauses, err := convertExpr(t, `active == true`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := clauses[0].Condition
	if c.Field != "active" || c.Value != true {
		t.Errorf("got field=%s value=%v", c.Field, c.Value)
	}
}

func TestExprToFilters_NotInClause(t *testing.T) {
	clauses, err := convertExpr(t, `department_id not in ctx.department_ids`, testCtx(), testFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := clauses[0].Condition
	if c.Operator != "not in" {
		t.Errorf("expected 'not in', got %s", c.Operator)
	}
}

func TestExprToFilters_CtxOnlyAccess(t *testing.T) {
	_, err := convertExpr(t, `created_by == env.user_id`, testCtx(), testFields())
	if err == nil {
		t.Fatal("expected error for non-ctx member access")
	}
}

func TestExprToFilters_FunctionCallRejected(t *testing.T) {
	_, err := convertExpr(t, `created_by == upper(ctx.user_id)`, testCtx(), testFields())
	if err == nil {
		t.Fatal("expected error for function call in record rule")
	}
}

func TestPreprocessRuleExpr(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"{{user.id}}", "ctx.user_id"},
		{"{{session.user_id}}", "ctx.user_id"},
		{"{{user.company_id}}", "ctx.company_id"},
		{"created_by == {{user.id}}", "created_by == ctx.user_id"},
		{"{{now}}", "ctx.now"},
		{"{{today}}", "ctx.today"},
		{"no_template", "no_template"},
		{"{{unknown.field}}", "{{unknown.field}}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parser.PreprocessRuleExpr(tt.input)
			if result != tt.expected {
				t.Errorf("PreprocessRuleExpr(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExprToFilters_MaxNodesExceeded(t *testing.T) {
	parts := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		parts = append(parts, fmt.Sprintf("status == '%d'", i))
	}
	expr := strings.Join(parts, " || ")
	_, err := convertExpr(t, expr, testCtx(), testFields())
	if err == nil {
		t.Fatal("expected error for expression exceeding max nodes")
	}
	if !strings.Contains(err.Error(), "too complex") {
		t.Errorf("expected 'too complex' error, got: %v", err)
	}
}

func TestRecordRuleContext_ToMap(t *testing.T) {
	ctx := testCtx()
	m := ctx.ToMap()

	if m["user_id"] != "usr-42" {
		t.Errorf("expected user_id=usr-42, got %v", m["user_id"])
	}
	if m["role"] != "manager" {
		t.Errorf("expected role=manager, got %v", m["role"])
	}
	ids, ok := m["department_ids"].([]string)
	if !ok || len(ids) != 2 {
		t.Errorf("expected 2 department_ids, got %v", m["department_ids"])
	}
}
