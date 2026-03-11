package gomp

// Page 分页对象
type Page[T any] struct {
	Current int64 `json:"current"` // 当前页（从 1 开始）
	Size    int64 `json:"size"`    // 每页条数
	Total   int64 `json:"total"`   // 总记录数
	Records []*T  `json:"records"` // 当前页数据
}

// NewPage 创建分页对象，自动修正非法参数
// current: 小于 1 时自动置为 1
// size:    小于 1 时自动置为 10，超过配置的 PageMaxSize 时按上限截断
func NewPage[T any](current, size int64) *Page[T] {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 10
	} else {
		maxSize := int64(getConfig().Gomp.PageMaxSize)
		if maxSize <= 0 {
			maxSize = 1000
		}
		if size > maxSize {
			size = maxSize
		}
	}
	return &Page[T]{
		Current: current,
		Size:    size,
		Records: make([]*T, 0),
	}
}

// Offset 计算偏移量（OFFSET）
func (p *Page[T]) Offset() int {
	return int((p.Current - 1) * p.Size)
}

// Limit 获取每页条数（LIMIT）
func (p *Page[T]) Limit() int {
	return int(p.Size)
}

// Pages 总页数
func (p *Page[T]) Pages() int64 {
	if p.Size <= 0 {
		return 0
	}
	pages := p.Total / p.Size
	if p.Total%p.Size != 0 {
		pages++
	}
	return pages
}

// HasNext 是否有下一页
func (p *Page[T]) HasNext() bool {
	return p.Current < p.Pages()
}

// HasPrev 是否有上一页
func (p *Page[T]) HasPrev() bool {
	return p.Current > 1
}
