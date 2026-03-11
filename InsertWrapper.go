package gomp

import (
	"errors"

	"gorm.io/gorm/clause"
)

// ConflictAction 冲突处理策略
// 从 1 开始，避免与零值歧义（0 表示未设置任何冲突策略）
type ConflictAction int

const (
	// ConflictDoNothing 冲突时忽略（INSERT IGNORE / ON CONFLICT DO NOTHING）
	ConflictDoNothing ConflictAction = iota + 1
	// ConflictDoUpdate 冲突时更新指定字段（upsert）
	ConflictDoUpdate
)

// InsertWrapper 插入构造器
type InsertWrapper[T any] struct {
	values         map[string]any
	conflictAction ConflictAction  // 0 = 未设置，1 = DoNothing，2 = DoUpdate
	conflictCols   []clause.Column // ON CONFLICT 目标列
	updateCols     []string        // 冲突时需要更新的列（空表示更新全部已 Set 的列）
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

// IsEmpty 检查是否未设置任何字段
func (w *InsertWrapper[T]) IsEmpty() bool {
	return len(w.values) == 0
}

// OnConflictDoNothing 冲突时忽略
func (w *InsertWrapper[T]) OnConflictDoNothing() *InsertWrapper[T] {
	w.conflictAction = ConflictDoNothing
	return w
}

// OnConflictDoUpdate 冲突时更新指定列（upsert）
// conflictColumns: 冲突判断的唯一键列名
// updateColumns:   冲突时要更新的列名，不传则更新所有已 Set 的列
func (w *InsertWrapper[T]) OnConflictDoUpdate(conflictColumns []string, updateColumns ...string) *InsertWrapper[T] {
	w.conflictAction = ConflictDoUpdate
	w.conflictCols = make([]clause.Column, 0, len(conflictColumns))
	for _, col := range conflictColumns {
		w.conflictCols = append(w.conflictCols, clause.Column{Name: col})
	}
	w.updateCols = updateColumns
	return w
}

// buildClause 构建 GORM OnConflict clause
func (w *InsertWrapper[T]) buildClause() (clause.OnConflict, error) {
	switch w.conflictAction {
	case ConflictDoNothing:
		return clause.OnConflict{DoNothing: true}, nil
	case ConflictDoUpdate:
		if len(w.updateCols) > 0 {
			assignments := make([]clause.Assignment, 0, len(w.updateCols))
			for _, col := range w.updateCols {
				assignments = append(assignments, clause.Assignment{
					Column: clause.Column{Name: col},
					Value:  clause.Column{Table: "excluded", Name: col},
				})
			}
			return clause.OnConflict{
				Columns:   w.conflictCols,
				DoUpdates: assignments,
			}, nil
		}
		// 不指定列时更新所有列
		return clause.OnConflict{
			Columns:   w.conflictCols,
			UpdateAll: true,
		}, nil
	}
	return clause.OnConflict{}, errors.New("unknown conflict action")
}
