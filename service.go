package gomp

import (
	"context"

	"gorm.io/gorm"
)

// IService 定义类似 MyBatis-Plus 的通用 Service 接口
type IService[T any] interface {
	Save(ctx context.Context, entity *T) error
	SaveBatch(ctx context.Context, entities []*T) error
	RemoveById(ctx context.Context, id any) error
	RemoveByIds(ctx context.Context, ids any) error
	UpdateById(ctx context.Context, entity *T) error
	GetById(ctx context.Context, id any) (*T, error)
	GetOne(ctx context.Context, wrapper *QueryWrapper[T]) (*T, error)
	List(ctx context.Context, wrapper *QueryWrapper[T]) ([]*T, error)
	Page(ctx context.Context, page *Page[T], wrapper *QueryWrapper[T]) (*Page[T], error)
	SelectPage(ctx context.Context, current, size int64, wrapper *QueryWrapper[T]) (*Page[T], error)
	Count(ctx context.Context, wrapper *QueryWrapper[T]) (int64, error)
	Insert(ctx context.Context, wrapper *InsertWrapper[T]) error
	Delete(ctx context.Context, wrapper *DeleteWrapper[T]) error
	Update(ctx context.Context, wrapper *UpdateWrapper[T]) error
	GetDB() *gorm.DB
}

// ServiceImpl 通用 Service 实现
type ServiceImpl[T any] struct {
	DB *gorm.DB
}

func NewServiceImpl[T any](db *gorm.DB) *ServiceImpl[T] {
	return &ServiceImpl[T]{DB: db}
}

func (s *ServiceImpl[T]) GetDB() *gorm.DB {
	return s.DB
}

func (s *ServiceImpl[T]) getDB(ctx context.Context) *gorm.DB {
	if config.Gomp.EnableSQLPrint {
		return s.DB.WithContext(ctx).Debug()
	}
	return s.DB.WithContext(ctx)
}

func (s *ServiceImpl[T]) Save(ctx context.Context, entity *T) error {
	return s.getDB(ctx).Create(entity).Error
}

func (s *ServiceImpl[T]) SaveBatch(ctx context.Context, entities []*T) error {
	return s.getDB(ctx).CreateInBatches(entities, 100).Error
}

func (s *ServiceImpl[T]) RemoveById(ctx context.Context, id any) error {
	var entity T
	return s.getDB(ctx).Delete(&entity, id).Error
}

func (s *ServiceImpl[T]) RemoveByIds(ctx context.Context, ids any) error {
	var entity T
	return s.getDB(ctx).Delete(&entity, ids).Error
}

func (s *ServiceImpl[T]) UpdateById(ctx context.Context, entity *T) error {
	return s.getDB(ctx).Updates(entity).Error
}

func (s *ServiceImpl[T]) GetById(ctx context.Context, id any) (*T, error) {
	var entity T
	err := s.getDB(ctx).First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (s *ServiceImpl[T]) GetOne(ctx context.Context, wrapper *QueryWrapper[T]) (*T, error) {
	var entity T
	db := s.getDB(ctx)
	if wrapper != nil {
		db = wrapper.Apply(db)
	}
	err := db.First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (s *ServiceImpl[T]) List(ctx context.Context, wrapper *QueryWrapper[T]) ([]*T, error) {
	var entities []*T
	db := s.getDB(ctx)
	if wrapper != nil {
		db = wrapper.Apply(db)
	}
	err := db.Find(&entities).Error
	return entities, err
}

func (s *ServiceImpl[T]) Page(ctx context.Context, page *Page[T], wrapper *QueryWrapper[T]) (*Page[T], error) {
	var entities []*T
	db := s.getDB(ctx).Model(new(T))
	if wrapper != nil {
		db = wrapper.Apply(db)
	}

	var total int64
	// 使用 Session 拷贝进行 Count，避免污染后续查询状态
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	page.Total = total

	// 如果没有数据，直接返回
	if total == 0 {
		return page, nil
	}

	if page.Size > 0 {
		db = db.Offset(page.Offset()).Limit(page.Limit())
	}

	if err := db.Find(&entities).Error; err != nil {
		return nil, err
	}
	page.Records = entities
	return page, nil
}

func (s *ServiceImpl[T]) SelectPage(ctx context.Context, current, size int64, wrapper *QueryWrapper[T]) (*Page[T], error) {
	page := NewPage[T](current, size)
	return s.Page(ctx, page, wrapper)
}

func (s *ServiceImpl[T]) Count(ctx context.Context, wrapper *QueryWrapper[T]) (int64, error) {
	var total int64
	db := s.getDB(ctx).Model(new(T))
	if wrapper != nil {
		db = wrapper.Apply(db)
	}
	err := db.Count(&total).Error
	return total, err
}

func (s *ServiceImpl[T]) Insert(ctx context.Context, wrapper *InsertWrapper[T]) error {
	return s.getDB(ctx).Model(new(T)).Create(wrapper.values).Error
}

func (s *ServiceImpl[T]) Delete(ctx context.Context, wrapper *DeleteWrapper[T]) error {
	db := s.getDB(ctx)
	if wrapper != nil {
		db = wrapper.Apply(db)
	}
	return db.Delete(new(T)).Error
}

func (s *ServiceImpl[T]) Update(ctx context.Context, wrapper *UpdateWrapper[T]) error {
	db := s.getDB(ctx)
	if wrapper != nil {
		db = wrapper.Apply(db)
	}
	return db.Model(new(T)).Updates(wrapper.values).Error
}

// SelectPage 快捷分页查询
func SelectPage[T any](ctx context.Context, db *gorm.DB, current, size int64, wrapper *QueryWrapper[T]) (*Page[T], error) {
	return NewServiceImpl[T](db).SelectPage(ctx, current, size, wrapper)
}

// SelectList 快捷列表查询
func SelectList[T any](ctx context.Context, db *gorm.DB, wrapper *QueryWrapper[T]) ([]*T, error) {
	return NewServiceImpl[T](db).List(ctx, wrapper)
}

// SelectOne 快捷单条查询
func SelectOne[T any](ctx context.Context, db *gorm.DB, wrapper *QueryWrapper[T]) (*T, error) {
	return NewServiceImpl[T](db).GetOne(ctx, wrapper)
}
