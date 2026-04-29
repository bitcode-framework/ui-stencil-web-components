package persistence

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/expr-lang/expr/ast"
	"gorm.io/gorm"
)

// Odoo-compatible rule composition:
//   - Global rules INTERSECT (AND)
//   - Group rules UNION (OR)
//   - Final = AND(global) AND OR(group)
type RecordRuleService struct {
	db          *gorm.DB
	tableName   func(string) string
	modelFields func(string) []string
}

func NewRecordRuleService(db *gorm.DB) *RecordRuleService {
	return &RecordRuleService{
		db:          db,
		tableName:   func(n string) string { return n },
		modelFields: func(n string) []string { return nil },
	}
}

func (s *RecordRuleService) SetTableNameResolver(fn func(string) string) {
	s.tableName = fn
}

func (s *RecordRuleService) SetModelFieldsResolver(fn func(string) []string) {
	s.modelFields = fn
}

func (s *RecordRuleService) tn(model string) string {
	return s.tableName(model)
}

func (s *RecordRuleService) GetFilters(userID string, modelName string, operation string) ([][]any, error) {
	var superCount int64
	if err := s.db.Table(s.tn("user")).Where("id = ? AND is_superuser = ?", userID, true).Count(&superCount).Error; err != nil {
		return nil, nil
	}
	if superCount > 0 {
		return nil, nil
	}

	groupIDs, err := s.resolveUserGroupIDs(userID)
	if err != nil {
		return nil, err
	}

	type ruleRow struct {
		ID               string `gorm:"column:id"`
		DomainFilter     string `gorm:"column:domain_filter"`
		DomainFilterExpr string `gorm:"column:domain_filter_expr"`
		CanRead          bool   `gorm:"column:can_read"`
		CanCreate        bool   `gorm:"column:can_create"`
		CanWrite         bool   `gorm:"column:can_write"`
		CanDelete        bool   `gorm:"column:can_delete"`
		GroupNames       string `gorm:"column:group_names"`
	}

	var rules []ruleRow
	if err := s.db.Table(s.tn("record_rule")).
		Select("id, domain_filter, domain_filter_expr, can_read, can_create, can_write, can_delete, group_names").
		Where("model_name = ? AND active = ?", modelName, true).
		Find(&rules).Error; err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, nil
	}

	var globalDomains [][]any
	var groupDomains [][]any

	for _, rule := range rules {
		if !ruleAppliesToOperation(rule.CanRead, rule.CanCreate, rule.CanWrite, rule.CanDelete, operation) {
			continue
		}

		domain, err := parseDomainFilter(rule.DomainFilter)
		if err != nil || len(domain) == 0 {
			continue
		}

		ruleGroupIDs := s.getRuleGroupIDs(rule.ID, rule.GroupNames)

		if len(ruleGroupIDs) == 0 {
			globalDomains = append(globalDomains, domain...)
		} else {
			if hasIntersection(ruleGroupIDs, groupIDs) {
				groupDomains = append(groupDomains, domain...)
			}
		}
	}

	var result [][]any
	result = append(result, globalDomains...)
	if len(groupDomains) > 0 {
		result = append(result, groupDomains...)
	}

	return result, nil
}

func (s *RecordRuleService) GetExprFilters(userID string, modelName string, operation string, ruleCtx *RecordRuleContext) ([]WhereClause, error) {
	var superCount int64
	if err := s.db.Table(s.tn("user")).Where("id = ? AND is_superuser = ?", userID, true).Count(&superCount).Error; err != nil {
		return nil, nil
	}
	if superCount > 0 {
		return nil, nil
	}

	groupIDs, err := s.resolveUserGroupIDs(userID)
	if err != nil {
		return nil, err
	}

	type ruleRow struct {
		ID               string `gorm:"column:id"`
		DomainFilterExpr string `gorm:"column:domain_filter_expr"`
		CanRead          bool   `gorm:"column:can_read"`
		CanCreate        bool   `gorm:"column:can_create"`
		CanWrite         bool   `gorm:"column:can_write"`
		CanDelete        bool   `gorm:"column:can_delete"`
		GroupNames       string `gorm:"column:group_names"`
	}

	var rules []ruleRow
	if err := s.db.Table(s.tn("record_rule")).
		Select("id, domain_filter_expr, can_read, can_create, can_write, can_delete, group_names").
		Where("model_name = ? AND active = ? AND domain_filter_expr != '' AND domain_filter_expr IS NOT NULL", modelName, true).
		Find(&rules).Error; err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, nil
	}

	fields := s.modelFields(modelName)
	var allClauses []WhereClause

	for _, rule := range rules {
		if !ruleAppliesToOperation(rule.CanRead, rule.CanCreate, rule.CanWrite, rule.CanDelete, operation) {
			continue
		}

		ruleGroupIDs := s.getRuleGroupIDs(rule.ID, rule.GroupNames)
		if len(ruleGroupIDs) > 0 && !hasIntersection(ruleGroupIDs, groupIDs) {
			continue
		}

		clauses, err := s.evaluateExprRule(rule.DomainFilterExpr, ruleCtx, fields)
		if err != nil {
			log.Printf("[RECORD_RULE] expr evaluation failed for rule %s: %v (deny-all)", rule.ID, err)
			return []WhereClause{*denyAllClause()}, nil
		}
		allClauses = append(allClauses, clauses...)
	}

	return allClauses, nil
}

func (s *RecordRuleService) evaluateExprRule(expression string, ruleCtx *RecordRuleContext, fields []string) ([]WhereClause, error) {
	tree, err := parseExprForRule(expression)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	walker := NewExprToFilters(ruleCtx, fields)
	return walker.Convert(tree)
}

var parseExprForRule = defaultParseExprForRule

func defaultParseExprForRule(expression string) (ast.Node, error) {
	return nil, fmt.Errorf("runtime.ParseExpr not wired")
}

func SetParseExprForRule(fn func(string) (ast.Node, error)) {
	parseExprForRule = fn
}

func (s *RecordRuleService) getRuleGroupIDs(ruleID string, legacyGroupNames string) []string {
	var groupIDs []string
	s.db.Table(s.tn("record_rule") + "_groups").
		Select("group_id").
		Where("record_rule_id = ?", ruleID).
		Pluck("group_id", &groupIDs)

	if len(groupIDs) > 0 {
		return groupIDs
	}

	if legacyGroupNames == "" {
		return nil
	}

	names := strings.Split(legacyGroupNames, ",")
	trimmedNames := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			trimmedNames = append(trimmedNames, n)
		}
	}
	if len(trimmedNames) == 0 {
		return nil
	}

	var resolvedIDs []string
	s.db.Table(s.tn("group")).
		Select("id").
		Where("name IN ?", trimmedNames).
		Pluck("id", &resolvedIDs)

	return resolvedIDs
}

func (s *RecordRuleService) resolveUserGroupIDs(userID string) ([]string, error) {
	var directGroupIDs []string
	if err := s.db.Table(s.tn("user") + "_" + s.tn("group")).
		Select("group_id").
		Where("user_id = ?", userID).
		Pluck("group_id", &directGroupIDs).Error; err != nil {
		return nil, err
	}

	if len(directGroupIDs) == 0 {
		return nil, nil
	}

	allGroupIDs := make(map[string]bool)
	queue := make([]string, len(directGroupIDs))
	copy(queue, directGroupIDs)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if allGroupIDs[current] {
			continue
		}
		allGroupIDs[current] = true

		var impliedIDs []string
		if err := s.db.Table(s.tn("group") + "_implies").
			Select("implied_group_id").
			Where("group_id = ?", current).
			Pluck("implied_group_id", &impliedIDs).Error; err != nil {
			continue
		}
		queue = append(queue, impliedIDs...)
	}

	result := make([]string, 0, len(allGroupIDs))
	for id := range allGroupIDs {
		result = append(result, id)
	}
	return result, nil
}

func ruleAppliesToOperation(canRead, canCreate, canWrite, canDelete bool, operation string) bool {
	switch operation {
	case "read", "list":
		return canRead
	case "create":
		return canCreate
	case "write", "update":
		return canWrite
	case "delete":
		return canDelete
	default:
		return false
	}
}

func parseDomainFilter(domainStr string) ([][]any, error) {
	if domainStr == "" || domainStr == "null" || domainStr == "[]" {
		return nil, nil
	}
	var domain [][]any
	if err := json.Unmarshal([]byte(domainStr), &domain); err != nil {
		var single []any
		if err2 := json.Unmarshal([]byte(domainStr), &single); err2 == nil && len(single) >= 3 {
			return [][]any{single}, nil
		}
		return nil, fmt.Errorf("invalid domain filter: %w", err)
	}
	return domain, nil
}

func hasIntersection(a, b []string) bool {
	set := make(map[string]bool, len(b))
	for _, v := range b {
		set[v] = true
	}
	for _, v := range a {
		if set[v] {
			return true
		}
	}
	return false
}

func InterpolateDomainFilters(filters [][]any, vars map[string]string) [][]any {
	if len(filters) == 0 {
		return filters
	}
	result := make([][]any, len(filters))
	for i, f := range filters {
		newF := make([]any, len(f))
		copy(newF, f)
		for j, v := range newF {
			if s, ok := v.(string); ok {
				for key, val := range vars {
					s = strings.ReplaceAll(s, fmt.Sprintf("{{%s}}", key), val)
				}
				newF[j] = s
			}
		}
		result[i] = newF
	}
	return result
}
