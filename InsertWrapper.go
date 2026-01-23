package gomp

// InsertWrapper 插入构造器
type InsertWrapper[T any] struct {
	values map[string]any
}

// NewInsertWrapper 创建插入构造器
func NewInsertWrapper[T any]() *InsertWrapper[T] {
	return &InsertWrapper[T]{
		values: make(map[string]any),
	}
}

// Set 设置插入字段
func (w *InsertWrapper[T]) Set(column string, val any, condition ...bool) *InsertWrapper[T] {
	if len(condition) > 0 && !condition[0] {
		return w
	}
	w.values[column] = val
	return w
}
