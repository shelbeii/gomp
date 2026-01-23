package gomp

// Page 分页对象
type Page[T any] struct {
	Current int64 `json:"current"` // 当前页
	Size    int64 `json:"size"`    // 每页显示条数
	Total   int64 `json:"total"`   // 总数
	Records []*T  `json:"records"` // 查询数据列表
}

// NewPage 创建分页对象
func NewPage[T any](current, size int64) *Page[T] {
	return &Page[T]{
		Current: current,
		Size:    size,
		Records: make([]*T, 0),
	}
}

// Offset 计算偏移量
func (p *Page[T]) Offset() int {
	if p.Current > 0 {
		return int((p.Current - 1) * p.Size)
	}
	return 0
}

// Limit 获取每页数量
func (p *Page[T]) Limit() int {
	return int(p.Size)
}
