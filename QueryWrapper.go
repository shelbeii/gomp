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
//
// 用法1：Or() - 设置下一个条件为 OR 连接
//
//	wrapper.Eq("status", 1).Or().Eq("status", 2)  // status = 1 OR status = 2
//
// 用法2：Or(func1, func2, ...) - 创建 OR 子句，整体以当前连接符追加
//
//	wrapper.Eq("type", "A").Or(
//	  func(sub *QueryWrapper[T]) { sub.Eq("status", 1) },
//	  func(sub *QueryWrapper[T]) { sub.Eq("status", 2) },
//	)  // type = 'A' AND (status = 1 OR status = 2)
func (w *QueryWrapper[T]) Or(conditions ...func(*QueryWrapper[T])) *QueryWrapper[T] {
	if len(conditions) == 0 {
		w.or = true
		return w
	}
	w.or = false
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		// 使用空 DB session 避免继承外部条件
		sess := db.Session(&gorm.Session{NewDB: true})
		firstSub := NewQueryWrapper[T]()
		conditions[0](firstSub)

		// 如果第一个子条件为空，检查是否有其他非空子条件
		if !firstSub.hasConditions() {
			for _, f := range conditions[1:] {
				nextSub := NewQueryWrapper[T]()
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
			nextSub := NewQueryWrapper[T]()
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

// And 将多个子条件以 AND 连接后追加（整体以当前连接符追加）
//
// 用法1：And() - 重置为 AND 连接（清除上一个 Or() 标记）
//
//	wrapper.Eq("status", 1).Or().Eq("status", 2).And().Eq("type", "A")
//	// (status = 1 OR status = 2) AND type = 'A'
//
// 用法2：And(func1, func2, ...) - 创建 AND 子句
//
//	wrapper.Eq("type", "A").And(
//	  func(sub *QueryWrapper[T]) { sub.Eq("status", 1) },
//	  func(sub *QueryWrapper[T]) { sub.Eq("level", 2) },
//	)  // type = 'A' AND (status = 1 AND level = 2)
func (w *QueryWrapper[T]) And(conditions ...func(*QueryWrapper[T])) *QueryWrapper[T] {
	if len(conditions) == 0 {
		w.or = false
		return w
	}
	isOr := w.or
	w.or = false
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		// 使用空 DB session 避免继承外部条件
		sess := db.Session(&gorm.Session{NewDB: true})
		firstSub := NewQueryWrapper[T]()
		conditions[0](firstSub)

		// 如果第一个子条件为空，检查是否有其他非空子条件
		if !firstSub.hasConditions() {
			for _, f := range conditions[1:] {
				nextSub := NewQueryWrapper[T]()
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
			nextSub := NewQueryWrapper[T]()
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

// AndOr 将多个子条件以 OR 连接，整体以 AND 加入查询
// 等价于: AND ( (子条件1) OR (子条件2) OR ... )
func (w *QueryWrapper[T]) AndOr(conditions ...func(*QueryWrapper[T])) *QueryWrapper[T] {
	if len(conditions) == 0 {
		return w
	}
	isOr := w.or
	w.or = false
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		// 使用空 DB session 避免继承外部条件
		sess := db.Session(&gorm.Session{NewDB: true})
		firstSub := NewQueryWrapper[T]()
		conditions[0](firstSub)

		// 如果第一个子条件为空，检查是否有其他非空子条件
		if !firstSub.hasConditions() {
			for _, f := range conditions[1:] {
				nextSub := NewQueryWrapper[T]()
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
			nextSub := NewQueryWrapper[T]()
			f(nextSub)
			if nextSub.hasConditions() {
				nextDB := nextSub.applyScopes(sess)
				subDB = subDB.Or(nextDB)
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

// NeColumn 列与列比较 left <> right
func (w *QueryWrapper[T]) NeColumn(leftColumn, rightColumn string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condNeColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// GtColumn 列与列比较 left > right
func (w *QueryWrapper[T]) GtColumn(leftColumn, rightColumn string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condGtColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// GeColumn 列与列比较 left >= right
func (w *QueryWrapper[T]) GeColumn(leftColumn, rightColumn string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condGeColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// LtColumn 列与列比较 left < right
func (w *QueryWrapper[T]) LtColumn(leftColumn, rightColumn string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLtColumn(&w.conditionMixin, leftColumn, rightColumn)
	return w
}

// LeColumn 列与列比较 left <= right
func (w *QueryWrapper[T]) LeColumn(leftColumn, rightColumn string, condition ...bool) *QueryWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	condLeColumn(&w.conditionMixin, leftColumn, rightColumn)
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

// SelectAs 指定查询字段并取别名
// expr:  字段或表达式，如 "opt.id"、"COALESCE(agg.rate, 0)"
// alias: 别名，如 "task_id"、"rate"
//
// 示例：
//
//	wrapper.SelectAs("COALESCE(agg.product_rate, 0)", "product_rate")
//	// 生成：COALESCE(agg.product_rate, 0) AS product_rate
//
//	wrapper.SelectAs("opt.id", "task_id")
//	// 生成：opt.id AS task_id
func (w *QueryWrapper[T]) SelectAs(expr string, alias string) *QueryWrapper[T] {
	w.selects = append(w.selects, expr+" AS "+alias)
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

// LeftJoinSubQuery 左连接子查询
// subQuery: 子查询 SQL，如 "SELECT id, AVG(score) AS avg_score FROM t GROUP BY id"
// alias:    子查询别名，如 "agg"
// leftCol:  主表关联列，如 "opt.id"
// rightCol: 子查询关联列，如 "agg.task_id"
// args:     子查询中的参数（对应 subQuery 中的 ?）
//
// 示例：
//
//	wrapper.LeftJoinSubQuery(
//	    "SELECT task_id, AVG(score) AS avg_score FROM t_scores WHERE status = ? GROUP BY task_id",
//	    "agg", "opt.id", "agg.task_id", 1,
//	)
func (w *QueryWrapper[T]) LeftJoinSubQuery(subQuery string, alias string, leftCol string, rightCol string, args ...any) *QueryWrapper[T] {
	w.scopes = append(w.scopes, buildSubQueryJoinScope("LEFT JOIN", subQuery, alias, leftCol, rightCol, args...))
	return w
}

// RightJoinSubQuery 右连接子查询
// 参数含义同 LeftJoinSubQuery
func (w *QueryWrapper[T]) RightJoinSubQuery(subQuery string, alias string, leftCol string, rightCol string, args ...any) *QueryWrapper[T] {
	w.scopes = append(w.scopes, buildSubQueryJoinScope("RIGHT JOIN", subQuery, alias, leftCol, rightCol, args...))
	return w
}

// InnerJoinSubQuery 内连接子查询
// 参数含义同 LeftJoinSubQuery
func (w *QueryWrapper[T]) InnerJoinSubQuery(subQuery string, alias string, leftCol string, rightCol string, args ...any) *QueryWrapper[T] {
	w.scopes = append(w.scopes, buildSubQueryJoinScope("INNER JOIN", subQuery, alias, leftCol, rightCol, args...))
	return w
}

// JoinRaw 原生 JOIN，完整传入 JOIN 语句，支持任意复杂写法
// joinSQL: 完整 JOIN 语句，如 "LEFT JOIN t_user u ON u.id = t.user_id AND u.status = ?"
// args:    joinSQL 中的参数
//
// 示例：
//
//	wrapper.JoinRaw("LEFT JOIN t_user u ON u.id = t.user_id AND u.status = ?", 1)
func (w *QueryWrapper[T]) JoinRaw(joinSQL string, args ...any) *QueryWrapper[T] {
	w.scopes = append(w.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins(joinSQL, args...)
	})
	return w
}

// Apply 应用所有条件到 GORM DB
func (w *QueryWrapper[T]) Apply(db *gorm.DB) *gorm.DB {
	if len(w.selects) > 0 {
		db = db.Select(w.selects)
	}
	return w.applyScopes(db)
}
