package gomp

import (
	"fmt"

	"gorm.io/gorm"
)

// QueryWrapper 查询条件构造器
type QueryWrapper[T any] struct {
	scopes  []func(*gorm.DB) *gorm.DB
	selects []string // 存储需要查询的字段
	or      bool     // 下一个条件是否使用 OR 连接
}

// NewQueryWrapper 创建查询条件构造器
func NewQueryWrapper[T any]() *QueryWrapper[T] {
	return &QueryWrapper[T]{
		scopes:  make([]func(*gorm.DB) *gorm.DB, 0),
		selects: make([]string, 0),
		or:      false,
	}
}

// addCondition 添加条件 (内部辅助方法)
func (w *QueryWrapper[T]) addCondition(query any, args ...any) {
	isOr := w.or
	w.or = false
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		if isOr {
			return db.Or(query, args...)
		}
		return db.Where(query, args...)
	})
}

// Or 设置下一个条件为 OR 连接，或者添加嵌套 OR 条件
// Or() -> 下一个条件使用 OR
// Or(func(w *QueryWrapper[T])) -> OR ( ... )
func (w *QueryWrapper[T]) Or(conditions ...func(*QueryWrapper[T])) *QueryWrapper[T] {
	if len(conditions) > 0 {
		f := conditions[0]
		isOr := w.or // 捕获当前连接符
		w.or = false
		w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
			subWrapper := NewQueryWrapper[T]()
			f(subWrapper)

			subDB := subWrapper.Apply(db.Session(&gorm.Session{NewDB: true}))

			if isOr {
				return db.Or(subDB)
			}
			return db.Or(subDB)
		})
		return w
	}
	w.or = true
	return w
}

// And 添加嵌套 AND 条件
// And(func(w *QueryWrapper[T])) -> AND ( ... )
func (w *QueryWrapper[T]) And(conditions ...func(*QueryWrapper[T])) *QueryWrapper[T] {
	if len(conditions) > 0 {
		f := conditions[0]
		isOr := w.or
		w.or = false
		w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
			subWrapper := NewQueryWrapper[T]()
			f(subWrapper)

			subDB := subWrapper.Apply(db.Session(&gorm.Session{NewDB: true}))

			if isOr {
				return db.Or(subDB)
			}
			return db.Where(subDB)
		})
	}
	// 如果没有参数，重置为 AND (默认就是 AND，所以其实不做操作，或者强制 w.or = false)
	w.or = false
	return w
}

// Eq 等于 =
func (w *QueryWrapper[T]) Eq(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s = ?", column), val)
	return w
}

// Ne 不等于 <>
func (w *QueryWrapper[T]) Ne(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s <> ?", column), val)
	return w
}

// Gt 大于 >
func (w *QueryWrapper[T]) Gt(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s > ?", column), val)
	return w
}

// Ge 大于等于 >=
func (w *QueryWrapper[T]) Ge(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s >= ?", column), val)
	return w
}

// Lt 小于 <
func (w *QueryWrapper[T]) Lt(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s < ?", column), val)
	return w
}

// Le 小于等于 <=
func (w *QueryWrapper[T]) Le(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s <= ?", column), val)
	return w
}

// Like 模糊查询 LIKE '%值%'
func (w *QueryWrapper[T]) Like(column string, val string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s LIKE ?", column), "%"+val+"%")
	return w
}

// LikeLeft 左模糊 LIKE '%值'
func (w *QueryWrapper[T]) LikeLeft(column string, val string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s LIKE ?", column), "%"+val)
	return w
}

// LikeRight 右模糊 LIKE '值%'
func (w *QueryWrapper[T]) LikeRight(column string, val string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s LIKE ?", column), val+"%")
	return w
}

// In IN 查询
func (w *QueryWrapper[T]) In(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s IN (?)", column), val)
	return w
}

// NotIn NOT IN 查询
func (w *QueryWrapper[T]) NotIn(column string, val any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s NOT IN (?)", column), val)
	return w
}

// IsNull IS NULL
func (w *QueryWrapper[T]) IsNull(column string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s IS NULL", column))
	return w
}

// IsNotNull IS NOT NULL
func (w *QueryWrapper[T]) IsNotNull(column string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s IS NOT NULL", column))
	return w
}

// Between BETWEEN AND
func (w *QueryWrapper[T]) Between(column string, val1, val2 any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s BETWEEN ? AND ?", column), val1, val2)
	return w
}

// NotBetween NOT BETWEEN AND
func (w *QueryWrapper[T]) NotBetween(column string, val1, val2 any, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s NOT BETWEEN ? AND ?", column), val1, val2)
	return w
}

// Table 指定表名/别名
func (w *QueryWrapper[T]) Table(name string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Table(name)
	})
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

// GroupBy 分组 GROUP BY
func (w *QueryWrapper[T]) GroupBy(columns ...string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		for _, column := range columns {
			db = db.Group(column)
		}
		return db
	})
	return w
}

// Having 分组后筛选 HAVING
func (w *QueryWrapper[T]) Having(query string, args ...any) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Having(query, args...)
	})
	return w
}

// Distinct 去重 DISTINCT
func (w *QueryWrapper[T]) Distinct(args ...any) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Distinct(args...)
	})
	return w
}

// Select 指定查询字段
func (w *QueryWrapper[T]) Select(columns ...string) *QueryWrapper[T] {
	w.selects = append(w.selects, columns...)
	return w
}

// LeftJoin 左连接
func (w *QueryWrapper[T]) LeftJoin(table string, on string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins(fmt.Sprintf("LEFT JOIN %s ON %s", table, on))
	})
	return w
}

// RightJoin 右连接
func (w *QueryWrapper[T]) RightJoin(table string, on string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins(fmt.Sprintf("RIGHT JOIN %s ON %s", table, on))
	})
	return w
}

// InnerJoin 内连接
func (w *QueryWrapper[T]) InnerJoin(table string, on string) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins(fmt.Sprintf("INNER JOIN %s ON %s", table, on))
	})
	return w
}

// Apply 应用条件到 GORM DB
func (w *QueryWrapper[T]) Apply(db *gorm.DB) *gorm.DB {
	if len(w.selects) > 0 {
		db = db.Select(w.selects)
	}
	for _, scope := range w.scopes {
		db = scope(db)
	}
	return db
}
