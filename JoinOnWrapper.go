package gomp

import "strings"

// JoinOnWrapper JOIN ON 条件构造器
type JoinOnWrapper struct {
	conditions []joinCondition
	or         bool
}

type joinCondition struct {
	query string
	args  []any
	isOr  bool
}

// NewJoinOnWrapper 创建 JoinOnWrapper
func NewJoinOnWrapper() *JoinOnWrapper {
	return &JoinOnWrapper{
		conditions: make([]joinCondition, 0, 4),
	}
}

func (w *JoinOnWrapper) addCondition(query string, args ...any) {
	if strings.TrimSpace(query) == "" {
		w.or = false
		return
	}
	isOr := w.or
	w.or = false
	w.conditions = append(w.conditions, joinCondition{query: query, args: args, isOr: isOr})
}

// Or 设置下一个条件为 OR，或添加嵌套 OR 子句
func (w *JoinOnWrapper) Or(conditions ...func(*JoinOnWrapper)) *JoinOnWrapper {
	if len(conditions) == 0 {
		w.or = true
		return w
	}
	// 处理所有条件，用 OR 连接
	for i, cond := range conditions {
		sub := NewJoinOnWrapper()
		cond(sub)
		clause, args := sub.Build()
		if strings.TrimSpace(clause) != "" {
			if i == 0 {
				w.or = true
			}
			w.addCondition("("+clause+")", args...)
		}
	}
	return w
}

// And 设置下一个条件为 AND，或添加嵌套 AND 子句
func (w *JoinOnWrapper) And(conditions ...func(*JoinOnWrapper)) *JoinOnWrapper {
	if len(conditions) == 0 {
		w.or = false
		return w
	}
	// 处理所有条件，用 AND 连接
	for _, cond := range conditions {
		sub := NewJoinOnWrapper()
		cond(sub)
		clause, args := sub.Build()
		if strings.TrimSpace(clause) != "" {
			w.addCondition("("+clause+")", args...)
		}
	}
	return w
}

// Raw 添加原生条件片段
func (w *JoinOnWrapper) Raw(query string, args ...any) *JoinOnWrapper {
	w.addCondition(query, args...)
	return w
}

func (w *JoinOnWrapper) Eq(column string, val any, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" = ?", val)
	return w
}

func (w *JoinOnWrapper) EqColumn(leftColumn string, rightColumn string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(leftColumn + " = " + rightColumn)
	return w
}

func (w *JoinOnWrapper) NeColumn(leftColumn string, rightColumn string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(leftColumn + " <> " + rightColumn)
	return w
}

func (w *JoinOnWrapper) GtColumn(leftColumn string, rightColumn string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(leftColumn + " > " + rightColumn)
	return w
}

func (w *JoinOnWrapper) GeColumn(leftColumn string, rightColumn string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(leftColumn + " >= " + rightColumn)
	return w
}

func (w *JoinOnWrapper) LtColumn(leftColumn string, rightColumn string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(leftColumn + " < " + rightColumn)
	return w
}

func (w *JoinOnWrapper) LeColumn(leftColumn string, rightColumn string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(leftColumn + " <= " + rightColumn)
	return w
}

func (w *JoinOnWrapper) Ne(column string, val any, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" <> ?", val)
	return w
}

func (w *JoinOnWrapper) Gt(column string, val any, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" > ?", val)
	return w
}

func (w *JoinOnWrapper) Ge(column string, val any, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" >= ?", val)
	return w
}

func (w *JoinOnWrapper) Lt(column string, val any, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" < ?", val)
	return w
}

func (w *JoinOnWrapper) Le(column string, val any, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" <= ?", val)
	return w
}

func (w *JoinOnWrapper) Like(column string, val string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" LIKE ?", "%"+val+"%")
	return w
}

func (w *JoinOnWrapper) LikeLeft(column string, val string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" LIKE ?", "%"+val)
	return w
}

func (w *JoinOnWrapper) LikeRight(column string, val string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" LIKE ?", val+"%")
	return w
}

func (w *JoinOnWrapper) In(column string, val any, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" IN (?)", val)
	return w
}

func (w *JoinOnWrapper) NotIn(column string, val any, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" NOT IN (?)", val)
	return w
}

func (w *JoinOnWrapper) IsNull(column string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column + " IS NULL")
	return w
}

func (w *JoinOnWrapper) IsNotNull(column string, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column + " IS NOT NULL")
	return w
}

func (w *JoinOnWrapper) Between(column string, val1, val2 any, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" BETWEEN ? AND ?", val1, val2)
	return w
}

func (w *JoinOnWrapper) NotBetween(column string, val1, val2 any, condition ...bool) *JoinOnWrapper {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.addCondition(column+" NOT BETWEEN ? AND ?", val1, val2)
	return w
}

// Build 构建 ON 子句字符串和参数
func (w *JoinOnWrapper) Build() (string, []any) {
	if len(w.conditions) == 0 {
		return "", nil
	}
	var sb strings.Builder
	args := make([]any, 0, len(w.conditions))
	for i, c := range w.conditions {
		if i > 0 {
			if c.isOr {
				sb.WriteString(" OR ")
			} else {
				sb.WriteString(" AND ")
			}
		}
		sb.WriteString(c.query)
		args = append(args, c.args...)
	}
	return sb.String(), args
}
