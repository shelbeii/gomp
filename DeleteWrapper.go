package gomp

import (
	"fmt"

	"gorm.io/gorm"
)

// DeleteWrapper 删除条件构造器
type DeleteWrapper[T any] struct {
	scopes []func(*gorm.DB) *gorm.DB
	or     bool // 下一个条件是否使用 OR 连接
}

// NewDeleteWrapper 创建删除条件构造器
func NewDeleteWrapper[T any]() *DeleteWrapper[T] {
	return &DeleteWrapper[T]{
		scopes: make([]func(*gorm.DB) *gorm.DB, 0),
		or:     false,
	}
}

// addCondition 添加条件 (内部辅助方法)
func (w *DeleteWrapper[T]) addCondition(query any, args ...any) {
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
func (w *DeleteWrapper[T]) Or(conditions ...func(*DeleteWrapper[T])) *DeleteWrapper[T] {
	if len(conditions) > 0 {
		f := conditions[0]
		isOr := w.or
		w.or = false
		w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
			subWrapper := NewDeleteWrapper[T]()
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
func (w *DeleteWrapper[T]) And(conditions ...func(*DeleteWrapper[T])) *DeleteWrapper[T] {
	if len(conditions) > 0 {
		f := conditions[0]
		isOr := w.or
		w.or = false
		w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
			subWrapper := NewDeleteWrapper[T]()
			f(subWrapper)

			subDB := subWrapper.Apply(db.Session(&gorm.Session{NewDB: true}))

			if isOr {
				return db.Or(subDB)
			}
			return db.Where(subDB)
		})
	}
	w.or = false
	return w
}

// Eq 等于 =
func (w *DeleteWrapper[T]) Eq(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s = ?", column), val)
	return w
}

// Ne 不等于 <>
func (w *DeleteWrapper[T]) Ne(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s <> ?", column), val)
	return w
}

// Gt 大于 >
func (w *DeleteWrapper[T]) Gt(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s > ?", column), val)
	return w
}

// Ge 大于等于 >=
func (w *DeleteWrapper[T]) Ge(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s >= ?", column), val)
	return w
}

// Lt 小于 <
func (w *DeleteWrapper[T]) Lt(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s < ?", column), val)
	return w
}

// Le 小于等于 <=
func (w *DeleteWrapper[T]) Le(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s <= ?", column), val)
	return w
}

// Like 模糊查询 LIKE '%值%'
func (w *DeleteWrapper[T]) Like(column string, val string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s LIKE ?", column), "%"+val+"%")
	return w
}

// LikeLeft 左模糊 LIKE '%值'
func (w *DeleteWrapper[T]) LikeLeft(column string, val string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s LIKE ?", column), "%"+val)
	return w
}

// LikeRight 右模糊 LIKE '值%'
func (w *DeleteWrapper[T]) LikeRight(column string, val string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s LIKE ?", column), val+"%")
	return w
}

// In IN 查询
func (w *DeleteWrapper[T]) In(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s IN (?)", column), val)
	return w
}

// NotIn NOT IN 查询
func (w *DeleteWrapper[T]) NotIn(column string, val any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s NOT IN (?)", column), val)
	return w
}

// IsNull IS NULL
func (w *DeleteWrapper[T]) IsNull(column string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s IS NULL", column))
	return w
}

// IsNotNull IS NOT NULL
func (w *DeleteWrapper[T]) IsNotNull(column string, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s IS NOT NULL", column))
	return w
}

// Between BETWEEN AND
func (w *DeleteWrapper[T]) Between(column string, val1, val2 any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s BETWEEN ? AND ?", column), val1, val2)
	return w
}

// NotBetween NOT BETWEEN AND
func (w *DeleteWrapper[T]) NotBetween(column string, val1, val2 any, condition ...bool) *DeleteWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(fmt.Sprintf("%s NOT BETWEEN ? AND ?", column), val1, val2)
	return w
}

// Apply 应用条件到 GORM DB
func (w *DeleteWrapper[T]) Apply(db *gorm.DB) *gorm.DB {
	for _, scope := range w.scopes {
		db = scope(db)
	}
	return db
}
