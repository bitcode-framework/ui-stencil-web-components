package persistence

import "gorm.io/gorm"

func MigrateRecordRuleExpr(db *gorm.DB) error {
	type recordRuleExprMigration struct {
		DomainFilterExpr string `gorm:"column:domain_filter_expr;type:text"`
	}
	return db.Table("record_rule").AutoMigrate(&recordRuleExprMigration{})
}
