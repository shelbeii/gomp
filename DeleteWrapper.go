package gomp

import (
	"gorm.io/gorm"
)

// DeleteWrapper 删除条件构造器
type DeleteWrapper[T any] struct {
	conditionMixin
	useSoftDelete bool
	tableName     string
}

// NewDeleteWrapper 创建删除条件构造器
func NewDeleteWrapper[T any]() *DeleteWrapper[T] {
	return &DeleteWrapper[T]{
		conditionMixin: newConditionMixin(),
		useSoftDelete:  true,
	}
}

// Table 指定表名
func (w *DeleteWrapper[T]) Table(name string) *DeleteWrapper[T] {
	w.tableName = name
	return w
}

// UseSoftDelete 设置是否使用软删除（默认 true）
func (w *DeleteWrapper[T]) UseSoftDelete(enabled bool) *DeleteWrapper[T] {
	w.useSoftDelete = enabled
	return w
}

// Raw 添加原生 SQL 条件片段
func (w *DeleteWrapper[T]) Raw(query string, args ...any) *DeleteWrapper[T] {
	condRaw(&w.conditionMixin, query, args...)
	return w
}

// Or 设置下一个条件为 OR，或将多个子条件以 OR 连接后追加
func (w *DeleteWrapper[T]) Or(conditions ...func(*DeleteWrapper[T])) *DeleteWrapper[T] {
	if len(conditions) == 0 {
		w.or = true
		return w
	}
	w.or = false
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		// 使用空 DB session 避免继承外部条件
		sess := db.Session(&gorm.Session{NewDB: true})
		firstSub := NewDeleteWrapper[T]()
		conditions[0](firstSub)

		// 如果第一个子条件为空，检查是否有其他非空子条件
		if !firstSub.hasConditions() {
			for _, f := range conditions[1:] {
				nextSub := NewDeleteWrapper[T]()
				f(nextSub)
				if nextSub.hasConditions() {
					firstSub = nextSub
					break
				}
			}
			// 如果所有子条件都为空，直接返回
			if !firstSub.hasConditions() {
				return db
			}
		}

		subDB := firstSub.applyScopes(sess)
		for _, f := range conditions[1:] {
			nextSub := NewDeleteWrapper[T]()
			f(nextSub)
			if nextSub.hasConditions() {
				nextDB := nextSub.applyScopes(sess)
				subDB = subDB.Or(nextDB)
			}
		}
		return db.Or(subDB)
	})
	return w
}

// And 将多个子条件以 AND 连接后追加
func (w *DeleteWrapper[T]) And(conditions ...func(*DeleteWrapper[T])) *DeleteWrapper[T] {
	if len(conditions) == 0 {
		w.or = false
		return w
	}
	isOr := w.or
	w.or = false
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		// 使用空 DB session 避免继承外部条件
		sess := db.Session(&gorm.Session{NewDB: true})
		firstSub := NewDeleteWrapper[T]()
		conditions[0](firstSub)

		// 如果第一个子条件为空，检查是否有其他非空子条件
		if !firstSub.hasConditions() {
			for _, f := range conditions[1:] {
				nextSub := NewDeleteWrapper[T]()
				f(nextSub)
				if nextSub.hasConditions() {
					firstSub = nextSub
					break
				}
			}
			// 如果所有子条件都为空，直接返回
			if !firstSub.hasConditions() {
				return db
			}
		}

		subDB := firstSub.applyScopes(sess)
		for _, f := range conditions[1:] {
			nextSub := NewDeleteWrapper[T]()
			f(nextSub)
			if nextSub.hasConditions() {
				nextDB := nextSub.applyScopes(sess)
				subDB = subDB.Where(nextDB)
			}
		}
		if isOr {
			return db.Or(subDB)
		}
		return db.Where(subDB)
	})
	return w
}

// Eq 等于 =
func (w *DeleteWrapper[T]) Eq(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condEq(&w.conditionMixin, column, val)
	return w
}

// EqColumn 列与列比较 left = right
func (w *DeleteWrapper[T]) EqColumn(leftColumn, rightColumn string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condEqColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// NeColumn 列与列比较 left <> right
func (w *DeleteWrapper[T]) NeColumn(leftColumn, rightColumn string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNeColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// GtColumn 列与列比较 left > right
func (w *DeleteWrapper[T]) GtColumn(leftColumn, rightColumn string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condGtColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// GeColumn 列与列比较 left >= right
func (w *DeleteWrapper[T]) GeColumn(leftColumn, rightColumn string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condGeColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// LtColumn 列与列比较 left < right
func (w *DeleteWrapper[T]) LtColumn(leftColumn, rightColumn string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLtColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// LeColumn 列与列比较 left <= right
func (w *DeleteWrapper[T]) LeColumn(leftColumn, rightColumn string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLeColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// Ne 不等于 <>
func (w *DeleteWrapper[T]) Ne(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNe(&w.conditionMixin, column, val)
	return w
}

// Gt 大于 >
func (w *DeleteWrapper[T]) Gt(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condGt(&w.conditionMixin, column, val)
	return w
}

// Ge 大于等于 >=
func (w *DeleteWrapper[T]) Ge(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condGe(&w.conditionMixin, column, val)
	return w
}

// Lt 小于 <
func (w *DeleteWrapper[T]) Lt(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLt(&w.conditionMixin, column, val)
	return w
}

// Le 小于等于 <=
func (w *DeleteWrapper[T]) Le(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLe(&w.conditionMixin, column, val)
	return w
}

// Like 模糊查询 LIKE '%值%'
func (w *DeleteWrapper[T]) Like(column string, val string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLike(&w.conditionMixin, column, val)
	return w
}

// LikeLeft 左模糊 LIKE '%值'
func (w *DeleteWrapper[T]) LikeLeft(column string, val string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLikeLeft(&w.conditionMixin, column, val)
	return w
}

// LikeRight 右模糊 LIKE '值%'
func (w *DeleteWrapper[T]) LikeRight(column string, val string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLikeRight(&w.conditionMixin, column, val)
	return w
}

// In IN 查询
func (w *DeleteWrapper[T]) In(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condIn(&w.conditionMixin, column, val)
	return w
}

// NotIn NOT IN 查询
func (w *DeleteWrapper[T]) NotIn(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNotIn(&w.conditionMixin, column, val)
	return w
}

// IsNull IS NULL
func (w *DeleteWrapper[T]) IsNull(column string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condIsNull(&w.conditionMixin, column)
	return w
}

// IsNotNull IS NOT NULL
func (w *DeleteWrapper[T]) IsNotNull(column string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condIsNotNull(&w.conditionMixin, column)
	return w
}

// Between BETWEEN AND
func (w *DeleteWrapper[T]) Between(column string, val1, val2 any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condBetween(&w.conditionMixin, column, val1, val2)
	return w
}

// NotBetween NOT BETWEEN AND
func (w *DeleteWrapper[T]) NotBetween(column string, val1, val2 any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNotBetween(&w.conditionMixin, column, val1, val2)
	return w
}

// LeftJoin 左连接
func (w *DeleteWrapper[T]) LeftJoin(table string, leftColumn string, rightColumn string) *DeleteWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins("LEFT JOIN " + table + " ON " + leftColumn + " = " + rightColumn)
	})
	return w
}

// RightJoin 右连接
func (w *DeleteWrapper[T]) RightJoin(table string, leftColumn string, rightColumn string) *DeleteWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins("RIGHT JOIN " + table + " ON " + leftColumn + " = " + rightColumn)
	})
	return w
}

// InnerJoin 内连接
func (w *DeleteWrapper[T]) InnerJoin(table string, leftColumn string, rightColumn string) *DeleteWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins("INNER JOIN " + table + " ON " + leftColumn + " = " + rightColumn)
	})
	return w
}

// LeftJoinOn 左连接（自定义 ON 条件）
func (w *DeleteWrapper[T]) LeftJoinOn(table string, leftColumn string, rightColumn string, builders ...func(*JoinOnWrapper)) *DeleteWrapper[T] {
	w.scopes = append(w.scopes, buildJoinScope("LEFT JOIN", table, leftColumn, rightColumn, builders...))
	return w
}

// RightJoinOn 右连接（自定义 ON 条件）
func (w *DeleteWrapper[T]) RightJoinOn(table string, leftColumn string, rightColumn string, builders ...func(*JoinOnWrapper)) *DeleteWrapper[T] {
	w.scopes = append(w.scopes, buildJoinScope("RIGHT JOIN", table, leftColumn, rightColumn, builders...))
	return w
}

// InnerJoinOn 内连接（自定义 ON 条件）
func (w *DeleteWrapper[T]) InnerJoinOn(table string, leftColumn string, rightColumn string, builders ...func(*JoinOnWrapper)) *DeleteWrapper[T] {
	w.scopes = append(w.scopes, buildJoinScope("INNER JOIN", table, leftColumn, rightColumn, builders...))
	return w
}

// Apply 应用条件到 GORM DB
// Table 先于 scopes 设置，确保 GORM 能正确关联表名与 JOIN
func (w *DeleteWrapper[T]) Apply(db *gorm.DB) *gorm.DB {
	if w.tableName != "" {
		db = db.Table(w.tableName)
	}
	return w.applyScopes(db)
}

// hasWhereConditions 检查是否有 WHERE 条件
func (w *DeleteWrapper[T]) hasWhereConditions() bool {
	return w.hasConditions()
}
