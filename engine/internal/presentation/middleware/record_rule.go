package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
)

type RecordRuleEngine interface {
	GetFilters(userID string, modelName string, operation string) ([][]any, error)
}

type ExprRecordRuleEngine interface {
	GetExprFilters(userID string, modelName string, operation string, ruleCtx *persistence.RecordRuleContext) ([]persistence.WhereClause, error)
}

func RecordRuleMiddleware(engine RecordRuleEngine, modelName string, operation string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(string)
		if !ok || userID == "" {
			return c.Next()
		}

		// Legacy domain_filter path
		filters, err := engine.GetFilters(userID, modelName, operation)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "record rule evaluation failed"})
		}

		if len(filters) > 0 {
			filters = interpolateFilters(filters, userID)
		}

		c.Locals("rls_filters", filters)

		// New domain_filter_expr path
		if exprEngine, ok := engine.(ExprRecordRuleEngine); ok {
			ruleCtx := buildRecordRuleContext(c)
			exprFilters, err := exprEngine.GetExprFilters(userID, modelName, operation, ruleCtx)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "record rule expression evaluation failed"})
			}
			if len(exprFilters) > 0 {
				c.Locals("rls_expr_filters", exprFilters)
			}
		}

		return c.Next()
	}
}

func buildRecordRuleContext(c *fiber.Ctx) *persistence.RecordRuleContext {
	ctx := &persistence.RecordRuleContext{
		Now:   time.Now(),
		Today: time.Now().Format("2006-01-02"),
	}
	if userID, ok := c.Locals("user_id").(string); ok {
		ctx.UserID = userID
	}
	if tenantID, ok := c.Locals("tenant_id").(string); ok {
		ctx.TenantID = tenantID
	}
	if roles, ok := c.Locals("roles").([]string); ok && len(roles) > 0 {
		ctx.Role = roles[0]
	}
	if groups, ok := c.Locals("groups").([]string); ok {
		ctx.Groups = groups
		ctx.GroupIDs = groups
	}
	return ctx
}

func interpolateFilters(filters [][]any, userID string) [][]any {
	result := make([][]any, len(filters))
	for i, f := range filters {
		newF := make([]any, len(f))
		copy(newF, f)
		for j, v := range newF {
			if s, ok := v.(string); ok {
				if s == "{{user.id}}" {
					newF[j] = userID
				}
			}
		}
		result[i] = newF
	}
	return result
}
