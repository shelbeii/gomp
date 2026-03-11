package gomp

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QueryWrapper 查询条件构造器
type QueryWrapper[T any] struct {
	conditionMixin
	selects []string
}

// NewQueryWrapper 创建查询条件构造器
func NewQueryWrapper[T any]() *QueryWrapper[T] {
	return &QueryWrapper[T]{
		conditionMixin: newConditionMixin(),
		selects:        make([]string, 0, 4),
	}
}

// Raw 添加原生 SQL 条件片段
func (w *QueryWrapper[T]) Raw(query string, args ...any) *QueryWrapper[T] {
	condRaw(&w.conditionMixin, query, args...)
	return w
}

// Or 设置下一个条件为 OR，或将多个子条件以 OR 连接后追加
// Or()                    -> 下一个条件用 OR 连接
// Or(f1, f2, ...)         -> (f1) OR (f2) OR ...，整体以当前连接符（AND/OR）追加
func (w *QueryWrapper[T]) Or(conditions ...func(*QueryWrapper[T])) *QueryWrapper[T] {
	if len(conditions) == 0 {
		w.or = true
		return w
	}
	isOr := w.or
	w.or = false
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		firstSub := NewQueryWrapper[T]()
		conditions[0](firstSub)
		subDB := firstSub.Apply(db.Session(&gorm.Session{NewDB: true}))
		for _, f := range conditions[1:] {
			nextSub := NewQueryWrapper[T]()
			f(nextSub)
			nextDB := nextSub.Apply(db.Session(&gorm.Session{NewDB: true}))
			subDB = subDB.Or(nextDB)
		}
		if isOr {
			return db.Or(subDB)
		}
		return db.Or(subDB)
	})
	return w
}

// And 将多个子条件以 AND 连接后追加（整体以当前连接符追加）
// And()                    -> 重置为 AND 连接（清除上一个 Or() 标记）
// And(f1, f2, ...)         -> (f1) AND (f2) AND ...，整体以当前连接符追加
func (w *QueryWrapper[T]) And(conditions ...func(*QueryWrapper[T])) *QueryWrapper[T] {
	if len(conditions) == 0 {
		w.or = false
		return w
	}
	isOr := w.or
	w.or = false
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		firstSub := NewQueryWrapper[T]()
		conditions[0](firstSub)
		subDB := firstSub.Apply(db.Session(&gorm.Session{NewDB: true}))
		for _, f := range conditions[1:] {
			nextSub := NewQueryWrapper[T]()
			f(nextSub)
			nextDB := nextSub.Apply(db.Session(&gorm.Session{NewDB: true}))
			subDB = subDB.Where(nextDB)
		}
		if isOr {
			return db.Or(subDB)
		}
		return db.Where(subDB)
	})
	return w
}

// AndOr 将多个子条件以 OR 连接，整体以 AND 加入查询
// 等价于: AND ( (子条件1) OR (子条件2) OR ... )
func (w *QueryWrapper[T]) AndOr(conditions ...func(*QueryWrapper[T])) *QueryWrapper[T] {
	if len(conditions) == 0 {
		return w
	}
	isOr := w.or
	w.or = false
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		firstSub := NewQueryWrapper[T]()
		conditions[0](firstSub)
		subDB := firstSub.Apply(db.Session(&gorm.Session{NewDB: true}))
		for _, f := range conditions[1:] {
			nextSub := NewQueryWrapper[T]()
			f(nextSub)
			nextDB := nextSub.Apply(db.Session(&gorm.Session{NewDB: true}))
			subDB = subDB.Or(nextDB)
		}
		if isOr {
			return db.Or(subDB)
		}
		return db.Where(subDB)
	})
	return w
}

// Eq 等于 =
func (w *QueryWrapper[T]) Eq(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condEq(&w.conditionMixin, column, val)
	return w
}

// EqColumn 列与列比较 left = right
func (w *QueryWrapper[T]) EqColumn(leftColumn, rightColumn string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condEqColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// Ne 不等于 <>
func (w *QueryWrapper[T]) Ne(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNe(&w.conditionMixin, column, val)
	return w
}

// Gt 大于 >
func (w *QueryWrapper[T]) Gt(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condGt(&w.conditionMixin, column, val)
	return w
}

// Ge 大于等于 >=
func (w *QueryWrapper[T]) Ge(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condGe(&w.conditionMixin, column, val)
	return w
}

// Lt 小于 <
func (w *QueryWrapper[T]) Lt(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLt(&w.conditionMixin, column, val)
	return w
}

// Le 小于等于 <=
func (w *QueryWrapper[T]) Le(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLe(&w.conditionMixin, column, val)
	return w
}

// Like 模糊查询 LIKE '%值%'
func (w *QueryWrapper[T]) Like(column string, val string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLike(&w.conditionMixin, column, val)
	return w
}

// LikeLeft 左模糊 LIKE '%值'
func (w *QueryWrapper[T]) LikeLeft(column string, val string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLikeLeft(&w.conditionMixin, column, val)
	return w
}

// LikeRight 右模糊 LIKE '值%'
func (w *QueryWrapper[T]) LikeRight(column string, val string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLikeRight(&w.conditionMixin, column, val)
	return w
}

// In IN 查询
func (w *QueryWrapper[T]) In(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condIn(&w.conditionMixin, column, val)
	return w
}

// NotIn NOT IN 查询
func (w *QueryWrapper[T]) NotIn(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNotIn(&w.conditionMixin, column, val)
	return w
}

// IsNull IS NULL
func (w *QueryWrapper[T]) IsNull(column string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condIsNull(&w.conditionMixin, column)
	return w
}

// IsNotNull IS NOT NULL
func (w *QueryWrapper[T]) IsNotNull(column string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condIsNotNull(&w.conditionMixin, column)
	return w
}

// Between BETWEEN AND
func (w *QueryWrapper[T]) Between(column string, val1, val2 any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condBetween(&w.conditionMixin, column, val1, val2)
	return w
}

// NotBetween NOT BETWEEN AND
func (w *QueryWrapper[T]) NotBetween(column string, val1, val2 any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNotBetween(&w.conditionMixin, column, val1, val2)
	return w
}

// Table 指定表名/别名
func (w *QueryWrapper[T]) Table(name string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Table(name)
	})
	return w
}

// Select 指定查询字段
func (w *QueryWrapper[T]) Select(columns ...string) *QueryWrapper[T] {
	w.selects = append(w.selects, columns...)
	return w
}

// OrderByDesc 降序
func (w *QueryWrapper[T]) OrderByDesc(column string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Order(column + " DESC")
	})
	return w
}

// OrderByAsc 升序
func (w *QueryWrapper[T]) OrderByAsc(column string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Order(column + " ASC")
	})
	return w
}

// GroupBy 分组 GROUP BY，多列一次性传入
func (w *QueryWrapper[T]) GroupBy(columns ...string) *QueryWrapper[T] {
	if len(columns) == 0 {
		return w
	}
	combined := strings.Join(columns, ", ")
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Group(combined)
	})
	return w
}

// Having 分组后筛选
func (w *QueryWrapper[T]) Having(query string, args ...any) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Having(query, args...)
	})
	return w
}

// Distinct 去重
func (w *QueryWrapper[T]) Distinct(args ...any) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Distinct(args...)
	})
	return w
}

// Limit 限制返回条数
func (w *QueryWrapper[T]) Limit(limit int) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit)
	})
	return w
}

// Offset 设置偏移量
func (w *QueryWrapper[T]) Offset(offset int) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Offset(offset)
	})
	return w
}

// ForUpdate 添加悲观锁 SELECT ... FOR UPDATE
func (w *QueryWrapper[T]) ForUpdate() *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	})
	return w
}

// ForShare 添加共享锁 SELECT ... FOR SHARE
func (w *QueryWrapper[T]) ForShare() *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Clauses(clause.Locking{Strength: "SHARE"})
	})
	return w
}

// LeftJoin 左连接
func (w *QueryWrapper[T]) LeftJoin(table string, leftColumn string, rightColumn string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins("LEFT JOIN " + table + " ON " + leftColumn + " = " + rightColumn)
	})
	return w
}

// RightJoin 右连接
func (w *QueryWrapper[T]) RightJoin(table string, leftColumn string, rightColumn string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins("RIGHT JOIN " + table + " ON " + leftColumn + " = " + rightColumn)
	})
	return w
}

// InnerJoin 内连接
func (w *QueryWrapper[T]) InnerJoin(table string, leftColumn string, rightColumn string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins("INNER JOIN " + table + " ON " + leftColumn + " = " + rightColumn)
	})
	return w
}

// LeftJoinOn 左连接（自定义 ON 条件）
func (w *QueryWrapper[T]) LeftJoinOn(table string, leftColumn string, rightColumn string, builders ...func(*JoinOnWrapper)) *QueryWrapper[T] {
	w.scopes = append(w.scopes, buildJoinScope("LEFT JOIN", table, leftColumn, rightColumn, builders...))
	return w
}

// RightJoinOn 右连接（自定义 ON 条件）
func (w *QueryWrapper[T]) RightJoinOn(table string, leftColumn string, rightColumn string, builders ...func(*JoinOnWrapper)) *QueryWrapper[T] {
	w.scopes = append(w.scopes, buildJoinScope("RIGHT JOIN", table, leftColumn, rightColumn, builders...))
	return w
}

// InnerJoinOn 内连接（自定义 ON 条件）
func (w *QueryWrapper[T]) InnerJoinOn(table string, leftColumn string, rightColumn string, builders ...func(*JoinOnWrapper)) *QueryWrapper[T] {
	w.scopes = append(w.scopes, buildJoinScope("INNER JOIN", table, leftColumn, rightColumn, builders...))
	return w
}

// Apply 应用所有条件到 GORM DB
func (w *QueryWrapper[T]) Apply(db *gorm.DB) *gorm.DB {
	if len(w.selects) > 0 {
		db = db.Select(w.selects)
	}
	return w.applyScopes(db)
}
