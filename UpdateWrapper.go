package gomp

import (
	"strings"

	"gorm.io/gorm"
)

// UpdateWrapper 更新条件构造器
type UpdateWrapper[T any] struct {
	conditionMixin
	values    map[string]any
	tableName string
	hasJoin   bool // 是否包含 JOIN（联表 UPDATE 需走 Exec 路径）
}

// NewUpdateWrapper 创建更新条件构造器
func NewUpdateWrapper[T any]() *UpdateWrapper[T] {
	return &UpdateWrapper[T]{
		conditionMixin: newConditionMixin(),
		values:         make(map[string]any, 8), // 预分配常见容量
	}
}

// Table 指定表名
func (w *UpdateWrapper[T]) Table(name string) *UpdateWrapper[T] {
	w.tableName = name
	return w
}

// Raw 添加原生 SQL 条件片段
func (w *UpdateWrapper[T]) Raw(query string, args ...any) *UpdateWrapper[T] {
	condRaw(&w.conditionMixin, query, args...)
	return w
}

// Or 设置下一个条件为 OR，或将多个子条件以 OR 连接后追加
func (w *UpdateWrapper[T]) Or(conditions ...func(*UpdateWrapper[T])) *UpdateWrapper[T] {
	if len(conditions) == 0 {
		w.or = true
		return w
	}
	w.or = false

	// 预先构建所有子条件，避免在 scope 闭包中重复创建
	subWrappers := make([]*UpdateWrapper[T], 0, len(conditions))
	for _, cond := range conditions {
		sub := NewUpdateWrapper[T]()
		cond(sub)
		// 只保留有条件的子 wrapper
		if sub.hasConditions() {
			subWrappers = append(subWrappers, sub)
		}
	}

	// 如果所有子条件都为空，直接返回
	if len(subWrappers) == 0 {
		return w
	}

	// 使用单个 scope 处理所有子条件，减少闭包分配
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		// 使用空 DB session 避免继承外部条件
		sess := db.Session(&gorm.Session{NewDB: true})
		subDB := subWrappers[0].applyScopes(sess)

		for i := 1; i < len(subWrappers); i++ {
			nextDB := subWrappers[i].applyScopes(sess)
			subDB = subDB.Or(nextDB)
		}
		return db.Or(subDB)
	})
	return w
}

// And 将多个子条件以 AND 连接后追加
func (w *UpdateWrapper[T]) And(conditions ...func(*UpdateWrapper[T])) *UpdateWrapper[T] {
	if len(conditions) == 0 {
		w.or = false
		return w
	}

	isOr := w.or
	w.or = false

	// 预先构建所有子条件，避免在 scope 闭包中重复创建
	subWrappers := make([]*UpdateWrapper[T], 0, len(conditions))
	for _, cond := range conditions {
		sub := NewUpdateWrapper[T]()
		cond(sub)
		// 只保留有条件的子 wrapper
		if sub.hasConditions() {
			subWrappers = append(subWrappers, sub)
		}
	}

	// 如果所有子条件都为空，直接返回
	if len(subWrappers) == 0 {
		return w
	}

	// 使用单个 scope 处理所有子条件，减少闭包分配
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		// 使用空 DB session 避免继承外部条件
		sess := db.Session(&gorm.Session{NewDB: true})
		subDB := subWrappers[0].applyScopes(sess)

		for i := 1; i < len(subWrappers); i++ {
			nextDB := subWrappers[i].applyScopes(sess)
			subDB = subDB.Where(nextDB)
		}

		if isOr {
			return db.Or(subDB)
		}
		return db.Where(subDB)
	})
	return w
}

// Set 设置更新字段
func (w *UpdateWrapper[T]) Set(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.values[column] = val
	return w
}

// SetRaw 设置字段为表达式值，如 SetRaw("json_col", "JSON_SET(json_col, '$.key', ?)" , val)
func (w *UpdateWrapper[T]) SetRaw(column string, expr string, args ...any) *UpdateWrapper[T] {
	w.values[column] = gorm.Expr(expr, args...)
	return w
}

// SetIncrBy 字段自增
func (w *UpdateWrapper[T]) SetIncrBy(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.values[column] = gorm.Expr(column+" + ?", val)
	return w
}

// SetDecrBy 字段自减
func (w *UpdateWrapper[T]) SetDecrBy(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.values[column] = gorm.Expr(column+" - ?", val)
	return w
}

// Eq 等于 =
func (w *UpdateWrapper[T]) Eq(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condEq(&w.conditionMixin, column, val)
	return w
}

// EqColumn 列与列比较 left = right
func (w *UpdateWrapper[T]) EqColumn(leftColumn, rightColumn string, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condEqColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// Ne 不等于 <>
func (w *UpdateWrapper[T]) Ne(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNe(&w.conditionMixin, column, val)
	return w
}

// Gt 大于 >
func (w *UpdateWrapper[T]) Gt(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condGt(&w.conditionMixin, column, val)
	return w
}

// Ge 大于等于 >=
func (w *UpdateWrapper[T]) Ge(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condGe(&w.conditionMixin, column, val)
	return w
}

// Lt 小于 <
func (w *UpdateWrapper[T]) Lt(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLt(&w.conditionMixin, column, val)
	return w
}

// Le 小于等于 <=
func (w *UpdateWrapper[T]) Le(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLe(&w.conditionMixin, column, val)
	return w
}

// Like 模糊查询 LIKE '%值%'
func (w *UpdateWrapper[T]) Like(column string, val string, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLike(&w.conditionMixin, column, val)
	return w
}

// LikeLeft 左模糊 LIKE '%值'
func (w *UpdateWrapper[T]) LikeLeft(column string, val string, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLikeLeft(&w.conditionMixin, column, val)
	return w
}

// LikeRight 右模糊 LIKE '值%'
func (w *UpdateWrapper[T]) LikeRight(column string, val string, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLikeRight(&w.conditionMixin, column, val)
	return w
}

// In IN 查询
func (w *UpdateWrapper[T]) In(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condIn(&w.conditionMixin, column, val)
	return w
}

// NotIn NOT IN 查询
func (w *UpdateWrapper[T]) NotIn(column string, val any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNotIn(&w.conditionMixin, column, val)
	return w
}

// IsNull IS NULL
func (w *UpdateWrapper[T]) IsNull(column string, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condIsNull(&w.conditionMixin, column)
	return w
}

// IsNotNull IS NOT NULL
func (w *UpdateWrapper[T]) IsNotNull(column string, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condIsNotNull(&w.conditionMixin, column)
	return w
}

// Between BETWEEN AND
func (w *UpdateWrapper[T]) Between(column string, val1, val2 any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condBetween(&w.conditionMixin, column, val1, val2)
	return w
}

// NotBetween NOT BETWEEN AND
func (w *UpdateWrapper[T]) NotBetween(column string, val1, val2 any, condition ...bool) *UpdateWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNotBetween(&w.conditionMixin, column, val1, val2)
	return w
}

// LeftJoin 左连接
func (w *UpdateWrapper[T]) LeftJoin(table string, leftColumn string, rightColumn string) *UpdateWrapper[T] {
	w.hasJoin = true
	// 使用 strings.Builder 预构建 JOIN SQL，避免每次查询时重复拼接
	var sb strings.Builder
	sb.Grow(11 + len(table) + 5 + len(leftColumn) + 3 + len(rightColumn))
	sb.WriteString("LEFT JOIN ")
	sb.WriteString(table)
	sb.WriteString(" ON ")
	sb.WriteString(leftColumn)
	sb.WriteString(" = ")
	sb.WriteString(rightColumn)
	joinSQL := sb.String()

	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins(joinSQL)
	})
	return w
}

// RightJoin 右连接
func (w *UpdateWrapper[T]) RightJoin(table string, leftColumn string, rightColumn string) *UpdateWrapper[T] {
	w.hasJoin = true
	var sb strings.Builder
	sb.Grow(12 + len(table) + 5 + len(leftColumn) + 3 + len(rightColumn))
	sb.WriteString("RIGHT JOIN ")
	sb.WriteString(table)
	sb.WriteString(" ON ")
	sb.WriteString(leftColumn)
	sb.WriteString(" = ")
	sb.WriteString(rightColumn)
	joinSQL := sb.String()

	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins(joinSQL)
	})
	return w
}

// InnerJoin 内连接
func (w *UpdateWrapper[T]) InnerJoin(table string, leftColumn string, rightColumn string) *UpdateWrapper[T] {
	w.hasJoin = true
	var sb strings.Builder
	sb.Grow(12 + len(table) + 5 + len(leftColumn) + 3 + len(rightColumn))
	sb.WriteString("INNER JOIN ")
	sb.WriteString(table)
	sb.WriteString(" ON ")
	sb.WriteString(leftColumn)
	sb.WriteString(" = ")
	sb.WriteString(rightColumn)
	joinSQL := sb.String()

	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins(joinSQL)
	})
	return w
}

// LeftJoinOn 左连接（自定义 ON 条件）
func (w *UpdateWrapper[T]) LeftJoinOn(table string, leftColumn string, rightColumn string, builders ...func(*JoinOnWrapper)) *UpdateWrapper[T] {
	w.hasJoin = true
	w.scopes = append(w.scopes, buildJoinScope("LEFT JOIN", table, leftColumn, rightColumn, builders...))
	return w
}

// RightJoinOn 右连接（自定义 ON 条件）
func (w *UpdateWrapper[T]) RightJoinOn(table string, leftColumn string, rightColumn string, builders ...func(*JoinOnWrapper)) *UpdateWrapper[T] {
	w.hasJoin = true
	w.scopes = append(w.scopes, buildJoinScope("RIGHT JOIN", table, leftColumn, rightColumn, builders...))
	return w
}

// InnerJoinOn 内连接（自定义 ON 条件）
func (w *UpdateWrapper[T]) InnerJoinOn(table string, leftColumn string, rightColumn string, builders ...func(*JoinOnWrapper)) *UpdateWrapper[T] {
	w.hasJoin = true
	w.scopes = append(w.scopes, buildJoinScope("INNER JOIN", table, leftColumn, rightColumn, builders...))
	return w
}

// Apply 应用条件到 GORM DB
// Table 先于 scopes 设置，确保 GORM 能正确关联表名与 JOIN
func (w *UpdateWrapper[T]) Apply(db *gorm.DB) *gorm.DB {
	if w.tableName != "" {
		db = db.Table(w.tableName)
	}
	return w.applyScopes(db)
}

// hasWhereConditions 检查是否有 WHERE 条件
func (w *UpdateWrapper[T]) hasWhereConditions() bool {
	return w.hasConditions()
}

// HasValues 检查是否有要更新的值
func (w *UpdateWrapper[T]) HasValues() bool {
	return len(w.values) > 0
}

// IsEmpty 检查 wrapper 是否为空（没有值也没有条件）
func (w *UpdateWrapper[T]) IsEmpty() bool {
	return len(w.values) == 0 && !w.hasConditions()
}

// GetTableName 获取表名
func (w *UpdateWrapper[T]) GetTableName() string {
	return w.tableName
}

// HasJoin 检查是否包含 JOIN
func (w *UpdateWrapper[T]) HasJoin() bool {
	return w.hasJoin
}
