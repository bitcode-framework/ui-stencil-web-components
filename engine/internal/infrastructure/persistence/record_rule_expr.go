package persistence

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/expr-lang/expr/ast"
)

type RecordRuleContext struct {
	UserID        string
	CompanyID     string
	CompanyIDs    []string
	DepartmentID  string
	DepartmentIDs []string
	GroupIDs      []string
	Groups        []string
	Role          string
	TenantID      string
	Now           time.Time
	Today         string
}

func (c *RecordRuleContext) ToMap() map[string]any {
	return map[string]any{
		"user_id":        c.UserID,
		"company_id":     c.CompanyID,
		"company_ids":    c.CompanyIDs,
		"department_id":  c.DepartmentID,
		"department_ids": c.DepartmentIDs,
		"group_ids":      c.GroupIDs,
		"groups":         c.Groups,
		"role":           c.Role,
		"tenant_id":      c.TenantID,
		"now":            c.Now,
		"today":          c.Today,
	}
}

const maxRecordRuleNodes = 200

// countNodes walks the AST and counts total nodes.
func countNodes(node ast.Node) int {
	count := 1
	switch n := node.(type) {
	case *ast.BinaryNode:
		count += countNodes(n.Left) + countNodes(n.Right)
	case *ast.UnaryNode:
		count += countNodes(n.Node)
	case *ast.ConditionalNode:
		count += countNodes(n.Cond) + countNodes(n.Exp1) + countNodes(n.Exp2)
	case *ast.BuiltinNode:
		for _, arg := range n.Arguments {
			count += countNodes(arg)
		}
	case *ast.CallNode:
		count += countNodes(n.Callee)
		for _, arg := range n.Arguments {
			count += countNodes(arg)
		}
	case *ast.ArrayNode:
		for _, elem := range n.Nodes {
			count += countNodes(elem)
		}
	}
	return count
}

type ExprToFilters struct {
	ctxData     map[string]any
	modelFields map[string]bool
	errors      []string
	hasField    bool
}

func NewExprToFilters(ctx *RecordRuleContext, modelFields []string) *ExprToFilters {
	fields := make(map[string]bool, len(modelFields))
	for _, f := range modelFields {
		fields[f] = true
	}
	return &ExprToFilters{
		ctxData:     ctx.ToMap(),
		modelFields: fields,
	}
}

func (w *ExprToFilters) Convert(node ast.Node) ([]WhereClause, error) {
	if nodeCount := countNodes(node); nodeCount > maxRecordRuleNodes {
		return nil, fmt.Errorf("record rule expression too complex: %d nodes (max %d)", nodeCount, maxRecordRuleNodes)
	}
	result := w.walkNode(node)
	if len(w.errors) > 0 {
		return nil, fmt.Errorf("record rule expression errors: %s", strings.Join(w.errors, "; "))
	}
	if !w.hasField {
		return nil, fmt.Errorf("record rule expression must reference at least one model field (tautology rejected)")
	}
	if result == nil {
		return nil, fmt.Errorf("record rule expression produced no filters")
	}
	return []WhereClause{*result}, nil
}

func (w *ExprToFilters) walkNode(node ast.Node) *WhereClause {
	switch n := node.(type) {
	case *ast.BinaryNode:
		return w.walkBinary(n)
	case *ast.UnaryNode:
		if n.Operator == "not" || n.Operator == "!" {
			inner := w.walkNode(n.Node)
			if inner == nil {
				return nil
			}
			if inner.Condition != nil {
				negated := w.negateOperator(inner.Condition.Operator)
				return &WhereClause{Condition: &Condition{
					Field:    inner.Condition.Field,
					Operator: negated,
					Value:    inner.Condition.Value,
				}}
			}
			if inner.Group != nil {
				return &WhereClause{Group: &ConditionGroup{
					Connector:  inner.Group.Connector,
					Conditions: inner.Group.Conditions,
					Negate:     !inner.Group.Negate,
				}}
			}
		}
		w.addError("unsupported unary operator: %s", n.Operator)
		return nil
	case *ast.ConditionalNode:
		return w.walkConditional(n)
	case *ast.BuiltinNode:
		if n.Name == "len" && len(n.Arguments) == 1 {
			return w.walkLenBuiltin(n)
		}
		w.addError("unsupported builtin function: %s", n.Name)
		return nil
	case *ast.CallNode:
		w.addError("function calls not allowed in record rules (got call to %s)", nodeToString(n.Callee))
		return nil
	default:
		w.addError("unsupported node type: %T", node)
		return nil
	}
}

func (w *ExprToFilters) walkBinary(n *ast.BinaryNode) *WhereClause {
	switch n.Operator {
	case "&&", "and":
		left := w.walkNode(n.Left)
		right := w.walkNode(n.Right)
		if left == nil || right == nil {
			return nil
		}
		return &WhereClause{Group: &ConditionGroup{
			Connector:  ConnectorAnd,
			Conditions: []WhereClause{*left, *right},
		}}

	case "||", "or":
		left := w.walkNode(n.Left)
		right := w.walkNode(n.Right)
		if left == nil || right == nil {
			return nil
		}
		return &WhereClause{Group: &ConditionGroup{
			Connector:  ConnectorOr,
			Conditions: []WhereClause{*left, *right},
		}}

	case "==", "!=", ">", "<", ">=", "<=":
		return w.extractComparison(n)

	case "in":
		return w.extractInClause(n, false)

	case "not in":
		return w.extractInClause(n, true)

	case "contains":
		return w.extractLikeClause(n, "%%%s%%")

	case "startsWith":
		return w.extractLikeClause(n, "%s%%")

	case "endsWith":
		return w.extractLikeClause(n, "%%%s")

	case "matches":
		w.addError("matches operator rejected in record rules (ReDoS risk on DB)")
		return nil

	default:
		w.addError("unsupported operator: %s", n.Operator)
		return nil
	}
}

func (w *ExprToFilters) extractComparison(n *ast.BinaryNode) *WhereClause {
	field := w.resolveFieldName(n.Left)
	if field == "" {
		return nil
	}
	w.hasField = true
	value, isDenyAll := w.resolveValue(n.Right)
	if isDenyAll {
		return denyAllClause()
	}
	return &WhereClause{Condition: &Condition{
		Field:    field,
		Operator: mapOperator(n.Operator),
		Value:    value,
	}}
}

func (w *ExprToFilters) extractInClause(n *ast.BinaryNode, negate bool) *WhereClause {
	field := w.resolveFieldName(n.Left)
	if field == "" {
		return nil
	}
	w.hasField = true
	value, isDenyAll := w.resolveValue(n.Right)
	if isDenyAll {
		return denyAllClause()
	}
	if arr, ok := value.([]any); ok && len(arr) == 0 {
		return denyAllClause()
	}
	op := "in"
	if negate {
		op = "not in"
	}
	return &WhereClause{Condition: &Condition{
		Field:    field,
		Operator: op,
		Value:    value,
	}}
}

func (w *ExprToFilters) extractLikeClause(n *ast.BinaryNode, pattern string) *WhereClause {
	field := w.resolveFieldName(n.Left)
	if field == "" {
		return nil
	}
	w.hasField = true
	value, isDenyAll := w.resolveValue(n.Right)
	if isDenyAll {
		return denyAllClause()
	}
	valStr, ok := value.(string)
	if !ok {
		w.addError("LIKE pattern value must be a string, got %T", value)
		return nil
	}
	w.hasField = true
	return &WhereClause{Condition: &Condition{
		Field:    field,
		Operator: "like",
		Value:    fmt.Sprintf(pattern, valStr),
	}}
}

func (w *ExprToFilters) walkConditional(n *ast.ConditionalNode) *WhereClause {
	condEnv := map[string]any{"ctx": w.ctxData}
	condExpr := nodeToExprString(n.Cond)

	goRuntime, err := evalBoolForRuleCondition(condExpr, condEnv)
	if err != nil {
		w.addError("cannot evaluate conditional: %v", err)
		return nil
	}
	if goRuntime {
		return w.walkNode(n.Exp1)
	}
	return w.walkNode(n.Exp2)
}

func (w *ExprToFilters) walkLenBuiltin(n *ast.BuiltinNode) *WhereClause {
	w.addError("len() in record rules not yet supported as standalone filter")
	return nil
}

func (w *ExprToFilters) resolveFieldName(node ast.Node) string {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		if !w.modelFields[n.Value] {
			w.addError("unknown model field: %s", n.Value)
			return ""
		}
		if !isSafeFieldName(n.Value) {
			w.addError("unsafe field name: %s", n.Value)
			return ""
		}
		return n.Value
	default:
		w.addError("left side of comparison must be a model field identifier, got %T", node)
		return ""
	}
}

func (w *ExprToFilters) resolveValue(node ast.Node) (any, bool) {
	switch n := node.(type) {
	case *ast.IntegerNode:
		return n.Value, false
	case *ast.FloatNode:
		return n.Value, false
	case *ast.StringNode:
		return n.Value, false
	case *ast.BoolNode:
		return n.Value, false
	case *ast.NilNode:
		return nil, true
	case *ast.MemberNode:
		return w.resolveMemberValue(n)
	case *ast.ArrayNode:
		return w.resolveArrayValue(n)
	case *ast.IdentifierNode:
		if n.Value == "nil" || n.Value == "null" {
			return nil, true
		}
		if n.Value == "true" {
			return true, false
		}
		if n.Value == "false" {
			return false, false
		}
		w.addError("bare identifier %q not allowed as value in record rules (use ctx.%s)", n.Value, n.Value)
		return nil, true
	default:
		w.addError("unsupported value node type: %T", node)
		return nil, true
	}
}

func (w *ExprToFilters) resolveMemberValue(n *ast.MemberNode) (any, bool) {
	ident, ok := n.Node.(*ast.IdentifierNode)
	if !ok {
		w.addError("member access must start with an identifier, got %T", n.Node)
		return nil, true
	}
	if ident.Value != "ctx" {
		w.addError("only ctx.* access allowed in record rules, got %s.*", ident.Value)
		return nil, true
	}
	prop, ok := n.Property.(*ast.StringNode)
	if !ok {
		w.addError("ctx member must be a string property")
		return nil, true
	}
	val, exists := w.ctxData[prop.Value]
	if !exists {
		w.addError("unknown ctx field: ctx.%s", prop.Value)
		return nil, true
	}
	if val == nil {
		return nil, true
	}
	if strSlice, ok := val.([]string); ok {
		result := make([]any, len(strSlice))
		for i, s := range strSlice {
			result[i] = s
		}
		return result, false
	}
	return val, false
}

func (w *ExprToFilters) resolveArrayValue(n *ast.ArrayNode) (any, bool) {
	result := make([]any, 0, len(n.Nodes))
	for _, elem := range n.Nodes {
		val, isDeny := w.resolveValue(elem)
		if isDeny {
			return nil, true
		}
		result = append(result, val)
	}
	return result, false
}

func (w *ExprToFilters) addError(format string, args ...any) {
	w.errors = append(w.errors, fmt.Sprintf(format, args...))
}

func (w *ExprToFilters) negateOperator(op string) string {
	switch op {
	case "=":
		return "!="
	case "!=":
		return "="
	case ">":
		return "<="
	case "<":
		return ">="
	case ">=":
		return "<"
	case "<=":
		return ">"
	case "in":
		return "not in"
	case "not in":
		return "in"
	case "like":
		return "not like"
	case "not like":
		return "like"
	default:
		return op
	}
}

func mapOperator(exprOp string) string {
	switch exprOp {
	case "==":
		return "="
	default:
		return exprOp
	}
}

var safeFieldNameRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func isSafeFieldName(name string) bool {
	return safeFieldNameRe.MatchString(name)
}

func denyAllClause() *WhereClause {
	return &WhereClause{Condition: &Condition{
		Field:    "1",
		Operator: "=",
		Value:    0,
	}}
}

func nodeToString(node ast.Node) string {
	if n, ok := node.(*ast.IdentifierNode); ok {
		return n.Value
	}
	return fmt.Sprintf("%T", node)
}

func nodeToExprString(node ast.Node) string {
	return fmt.Sprintf("%s", node)
}

func evalBoolForRuleCondition(expression string, env map[string]any) (bool, error) {
	goRuntime, err := runtimeEvalExprBool(expression, env)
	if err != nil {
		return false, err
	}
	return goRuntime, nil
}

var runtimeEvalExprBool = defaultEvalExprBool

func defaultEvalExprBool(expression string, env map[string]any) (bool, error) {
	return false, fmt.Errorf("runtime.EvalExprBool not wired")
}

func SetRuntimeEvalExprBool(fn func(string, map[string]any) (bool, error)) {
	runtimeEvalExprBool = fn
}
