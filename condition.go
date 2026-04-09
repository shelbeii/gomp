package gomp

import (
	"strings"

	"gorm.io/gorm"
)

// conditionMixin 条件构建公共逻辑，通过组合嵌入到各 Wrapper
type conditionMixin struct {
	scopes []func(*gorm.DB) *gorm.DB
	or     bool // 下一个条件是否使用 OR 连接
}

func newConditionMixin() conditionMixin {
	return conditionMixin{
		scopes: make([]func(*gorm.DB) *gorm.DB, 0, 4),
	}
}

// addCond 添加条件到 scopes（内部使用）
func (m *conditionMixin) addCond(query any, args ...any) {
	isOr := m.or
	m.or = false
	m.scopes = append(m.scopes, func(db *gorm.DB) *gorm.DB {
		if isOr {
			return db.Or(query, args...)
		}
		return db.Where(query, args...)
	})
}

// applyScopes 将所有 scopes 应用到 db
func (m *conditionMixin) applyScopes(db *gorm.DB) *gorm.DB {
	for _, scope := range m.scopes {
		db = scope(db)
	}
	return db
}

// hasConditions 是否有任何 WHERE 条件
func (m *conditionMixin) hasConditions() bool {
	return len(m.scopes) > 0
}

// --- 各操作符公共实现，使用字符串拼接替代 fmt.Sprintf 减少分配 ---

func condEq(m *conditionMixin, column string, val any) {
	m.addCond(column+" = ?", val)
}

func condNe(m *conditionMixin, column string, val any) {
	m.addCond(column+" <> ?", val)
}

func condGt(m *conditionMixin, column string, val any) {
	m.addCond(column+" > ?", val)
}

func condGe(m *conditionMixin, column string, val any) {
	m.addCond(column+" >= ?", val)
}

func condLt(m *conditionMixin, column string, val any) {
	m.addCond(column+" < ?", val)
}

func condLe(m *conditionMixin, column string, val any) {
	m.addCond(column+" <= ?", val)
}

func condLike(m *conditionMixin, column string, val string) {
	m.addCond(column+" LIKE ?", "%"+val+"%")
}

func condLikeLeft(m *conditionMixin, column string, val string) {
	m.addCond(column+" LIKE ?", "%"+val)
}

func condLikeRight(m *conditionMixin, column string, val string) {
	m.addCond(column+" LIKE ?", val+"%")
}

func condIn(m *conditionMixin, column string, val any) {
	m.addCond(column+" IN (?)", val)
}

func condNotIn(m *conditionMixin, column string, val any) {
	m.addCond(column+" NOT IN (?)", val)
}

func condIsNull(m *conditionMixin, column string) {
	m.addCond(column + " IS NULL")
}

func condIsNotNull(m *conditionMixin, column string) {
	m.addCond(column + " IS NOT NULL")
}

func condBetween(m *conditionMixin, column string, val1, val2 any) {
	m.addCond(column+" BETWEEN ? AND ?", val1, val2)
}

func condNotBetween(m *conditionMixin, column string, val1, val2 any) {
	m.addCond(column+" NOT BETWEEN ? AND ?", val1, val2)
}

func condRaw(m *conditionMixin, query string, args ...any) {
	m.addCond(query, args...)
}

func condEqColumn(m *conditionMixin, leftColumn, rightColumn string) {
	m.addCond(leftColumn + " = " + rightColumn)
}

func condNeColumn(m *conditionMixin, leftColumn, rightColumn string) {
	m.addCond(leftColumn + " <> " + rightColumn)
}

func condGtColumn(m *conditionMixin, leftColumn, rightColumn string) {
	m.addCond(leftColumn + " > " + rightColumn)
}

func condGeColumn(m *conditionMixin, leftColumn, rightColumn string) {
	m.addCond(leftColumn + " >= " + rightColumn)
}

func condLtColumn(m *conditionMixin, leftColumn, rightColumn string) {
	m.addCond(leftColumn + " < " + rightColumn)
}

func condLeColumn(m *conditionMixin, leftColumn, rightColumn string) {
	m.addCond(leftColumn + " <= " + rightColumn)
}

// buildJoinScope 构造带自定义 ON 条件的 JOIN scope
// 提前执行 Build() 缓存结果，避免每次查询重复构建
func buildJoinScope(joinType, table, leftColumn, rightColumn string, builders ...func(*JoinOnWrapper)) func(*gorm.DB) *gorm.DB {
	onWrapper := NewJoinOnWrapper()
	onWrapper.EqColumn(leftColumn, rightColumn)
	for _, b := range builders {
		if b != nil {
			b(onWrapper)
		}
	}
	clause, args := onWrapper.Build()
	if strings.TrimSpace(clause) == "" {
		return func(db *gorm.DB) *gorm.DB { return db }
	}
	joinSQL := joinType + " " + table + " ON " + clause
	return func(db *gorm.DB) *gorm.DB {
		return db.Joins(joinSQL, args...)
	}
}

// buildSubQueryJoinScope 构造子查询 JOIN scope
// joinType: JOIN 类型，如 "LEFT JOIN", "RIGHT JOIN", "INNER JOIN"
// subQuery: 子查询 SQL（不含括号），如 "SELECT id, AVG(score) AS avg FROM t GROUP BY id"
// alias:    子查询别名
// leftCol:  主表关联列
// rightCol: 子查询关联列
// args:     子查询中的参数
func buildSubQueryJoinScope(joinType, subQuery, alias, leftCol, rightCol string, args ...any) func(*gorm.DB) *gorm.DB {
	var sb strings.Builder
	sb.Grow(len(joinType) + len(subQuery) + len(alias) + len(leftCol) + len(rightCol) + 16)
	sb.WriteString(joinType)
	sb.WriteString(" (")
	sb.WriteString(subQuery)
	sb.WriteString(") ")
	sb.WriteString(alias)
	sb.WriteString(" ON ")
	sb.WriteString(leftCol)
	sb.WriteString(" = ")
	sb.WriteString(rightCol)
	joinSQL := sb.String()
	return func(db *gorm.DB) *gorm.DB {
		return db.Joins(joinSQL, args...)
	}
}
