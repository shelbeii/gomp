package gomp

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// IService 定义通用 Service 接口
type IService[T any] interface {
	Save(ctx context.Context, entity *T) error
	SaveBatch(ctx context.Context, entities []*T, batchSize ...int) error
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

// NewServiceImpl 创建 ServiceImpl
func NewServiceImpl[T any](db *gorm.DB) *ServiceImpl[T] {
	return &ServiceImpl[T]{DB: db}
}

// GetDB 获取原始 DB
func (s *ServiceImpl[T]) GetDB() *gorm.DB {
	return s.DB
}

// getDB 按配置返回带 ctx 的 DB
func (s *ServiceImpl[T]) getDB(ctx context.Context) *gorm.DB {
	if getConfig().Gomp.EnableSQLPrint {
		return s.DB.WithContext(ctx).Debug()
	}
	return s.DB.WithContext(ctx)
}

// Save 保存单条记录
func (s *ServiceImpl[T]) Save(ctx context.Context, entity *T) error {
	return s.getDB(ctx).Create(entity).Error
}

// SaveBatch 批量保存，空切片直接返回
func (s *ServiceImpl[T]) SaveBatch(ctx context.Context, entities []*T, batchSize ...int) error {
	if len(entities) == 0 {
		return nil
	}
	size := getConfig().Gomp.SaveBatchSize
	if len(batchSize) > 0 && batchSize[0] > 0 {
		size = batchSize[0]
	}
	return s.getDB(ctx).CreateInBatches(entities, size).Error
}

// RemoveById 根据主键删除
func (s *ServiceImpl[T]) RemoveById(ctx context.Context, id any) error {
	var entity T
	return s.getDB(ctx).Delete(&entity, id).Error
}

// RemoveByIds 根据主键批量删除
func (s *ServiceImpl[T]) RemoveByIds(ctx context.Context, ids any) error {
	var entity T
	return s.getDB(ctx).Delete(&entity, ids).Error
}

// UpdateById 根据主键更新（只更新非零字段）
func (s *ServiceImpl[T]) UpdateById(ctx context.Context, entity *T) error {
	return s.getDB(ctx).Updates(entity).Error
}

// GetById 根据主键查询，不存在返回 (nil, nil)
func (s *ServiceImpl[T]) GetById(ctx context.Context, id any) (*T, error) {
	var entity T
	err := s.getDB(ctx).First(&entity, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// GetOne 根据条件查询单条，不存在返回 (nil, nil)
func (s *ServiceImpl[T]) GetOne(ctx context.Context, wrapper *QueryWrapper[T]) (*T, error) {
	var entity T
	db := s.getDB(ctx)
	if wrapper != nil {
		db = wrapper.Apply(db)
	}
	err := db.Take(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// List 根据条件查询列表，空结果返回空切片（非 nil）
func (s *ServiceImpl[T]) List(ctx context.Context, wrapper *QueryWrapper[T]) ([]*T, error) {
	entities := make([]*T, 0)
	db := s.getDB(ctx)
	if wrapper != nil {
		db = wrapper.Apply(db)
	}
	err := db.Find(&entities).Error
	return entities, err
}

// Page 分页查询
func (s *ServiceImpl[T]) Page(ctx context.Context, page *Page[T], wrapper *QueryWrapper[T]) (*Page[T], error) {
	entities := make([]*T, 0)
	db := s.getDB(ctx)
	if wrapper != nil {
		db = wrapper.Apply(db)
	}
	// 若 wrapper 未指定 Table，则尝试用 Model(new(T)) 推断表名（普通 Model 场景）
	if db.Statement == nil || db.Statement.Table == "" {
		db = db.Model(new(T))
	}
	var total int64
	// 用 Session(SkipHooks+SkipDefaultTransaction) 克隆当前 db 做 COUNT，保留 Table/JOIN/WHERE
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	page.Total = total
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

// SelectPage 快捷分页查询
func (s *ServiceImpl[T]) SelectPage(ctx context.Context, current, size int64, wrapper *QueryWrapper[T]) (*Page[T], error) {
	return s.Page(ctx, NewPage[T](current, size), wrapper)
}

// Count 统计记录数
func (s *ServiceImpl[T]) Count(ctx context.Context, wrapper *QueryWrapper[T]) (int64, error) {
	var total int64
	db := s.getDB(ctx)
	if wrapper != nil {
		db = wrapper.Apply(db)
	}
	if db.Statement == nil || db.Statement.Table == "" {
		db = db.Model(new(T))
	}
	err := db.Count(&total).Error
	return total, err
}

// Insert 根据 InsertWrapper 插入，支持 OnConflict
func (s *ServiceImpl[T]) Insert(ctx context.Context, wrapper *InsertWrapper[T]) error {
	if wrapper == nil {
		return errors.New("insert wrapper cannot be nil")
	}
	if wrapper.IsEmpty() {
		return errors.New("insert wrapper has no fields set")
	}
	db := s.getDB(ctx).Model(new(T))
	if wrapper.conflictAction != 0 {
		oc, err := wrapper.buildClause()
		if err != nil {
			return err
		}
		db = db.Clauses(oc)
	}
	return db.Create(wrapper.values).Error
}

// Delete 根据 DeleteWrapper 删除
func (s *ServiceImpl[T]) Delete(ctx context.Context, wrapper *DeleteWrapper[T]) error {
	if !getConfig().Gomp.AllowGlobalDelete {
		if wrapper == nil || !wrapper.hasWhereConditions() {
			return errors.New("global delete is not allowed without WHERE clause; set AllowGlobalDelete=true to override")
		}
	}
	db := s.getDB(ctx)
	useSoftDelete := true
	if wrapper != nil {
		useSoftDelete = wrapper.useSoftDelete
		db = wrapper.Apply(db)
	}
	if !useSoftDelete {
		db = db.Unscoped()
	}
	return db.Delete(new(T)).Error
}

// Update 根据 UpdateWrapper 更新
func (s *ServiceImpl[T]) Update(ctx context.Context, wrapper *UpdateWrapper[T]) error {
	if wrapper == nil {
		return errors.New("update wrapper cannot be nil")
	}
	if !getConfig().Gomp.AllowGlobalUpdate && !wrapper.hasWhereConditions() {
		return errors.New("global update is not allowed without WHERE clause; set AllowGlobalUpdate=true to override")
	}
	db := wrapper.Apply(s.getDB(ctx))
	// 联表更新（指定了 Table）时直接用当前 db.Updates，避免 Model(new(T)) 覆盖表名并丢失 JOIN
	// 未指定 Table 时用 Model(new(T)) 让 GORM 推断表名及软删除字段
	if wrapper.tableName != "" {
		return db.Updates(wrapper.values).Error
	}
	return db.Model(new(T)).Updates(wrapper.values).Error
}

// ─────────────────────────────────────────────────────
// 包级快捷函数：直接内联操作 db，避免 NewServiceImpl 堆分配
// ─────────────────────────────────────────────────────

// withCtx 内部辅助：按配置附加 ctx 和 Debug
func withCtx(db *gorm.DB, ctx context.Context) *gorm.DB {
	if getConfig().Gomp.EnableSQLPrint {
		return db.WithContext(ctx).Debug()
	}
	return db.WithContext(ctx)
}

// SelectPage 快捷分页查询
func SelectPage[T any](ctx context.Context, db *gorm.DB, current, size int64, wrapper *QueryWrapper[T]) (*Page[T], error) {
	return NewServiceImpl[T](db).SelectPage(ctx, current, size, wrapper)
}

// SelectList 快捷列表查询
func SelectList[T any](ctx context.Context, db *gorm.DB, wrapper *QueryWrapper[T]) ([]*T, error) {
	entities := make([]*T, 0)
	d := withCtx(db, ctx)
	if wrapper != nil {
		d = wrapper.Apply(d)
	}
	return entities, d.Find(&entities).Error
}

// SelectOne 快捷单条查询
func SelectOne[T any](ctx context.Context, db *gorm.DB, wrapper *QueryWrapper[T]) (*T, error) {
	var entity T
	d := withCtx(db, ctx)
	if wrapper != nil {
		d = wrapper.Apply(d)
	}
	err := d.Take(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// Save 快捷保存
func Save[T any](ctx context.Context, db *gorm.DB, entity *T) error {
	return withCtx(db, ctx).Create(entity).Error
}

// SaveBatch 快捷批量保存
func SaveBatch[T any](ctx context.Context, db *gorm.DB, entities []*T, batchSize ...int) error {
	if len(entities) == 0 {
		return nil
	}
	size := getConfig().Gomp.SaveBatchSize
	if len(batchSize) > 0 && batchSize[0] > 0 {
		size = batchSize[0]
	}
	return withCtx(db, ctx).CreateInBatches(entities, size).Error
}

// RemoveById 快捷根据 ID 删除
func RemoveById[T any](ctx context.Context, db *gorm.DB, id any) error {
	var entity T
	return withCtx(db, ctx).Delete(&entity, id).Error
}

// RemoveByIds 快捷根据 ID 批量删除
func RemoveByIds[T any](ctx context.Context, db *gorm.DB, ids any) error {
	var entity T
	return withCtx(db, ctx).Delete(&entity, ids).Error
}

// UpdateById 快捷根据 ID 更新
func UpdateById[T any](ctx context.Context, db *gorm.DB, entity *T) error {
	return withCtx(db, ctx).Updates(entity).Error
}

// GetById 快捷根据 ID 查询
func GetById[T any](ctx context.Context, db *gorm.DB, id any) (*T, error) {
	var entity T
	err := withCtx(db, ctx).First(&entity, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// Count 快捷统计
func Count[T any](ctx context.Context, db *gorm.DB, wrapper *QueryWrapper[T]) (int64, error) {
	var total int64
	d := withCtx(db, ctx)
	if wrapper != nil {
		d = wrapper.Apply(d)
	}
	if d.Statement == nil || d.Statement.Table == "" {
		d = d.Model(new(T))
	}
	return total, d.Count(&total).Error
}

// Insert 快捷插入
func Insert[T any](ctx context.Context, db *gorm.DB, wrapper *InsertWrapper[T]) error {
	return NewServiceImpl[T](db).Insert(ctx, wrapper)
}

// Delete 快捷删除
func Delete[T any](ctx context.Context, db *gorm.DB, wrapper *DeleteWrapper[T]) error {
	return NewServiceImpl[T](db).Delete(ctx, wrapper)
}

// Update 快捷更新
func Update[T any](ctx context.Context, db *gorm.DB, wrapper *UpdateWrapper[T]) error {
	return NewServiceImpl[T](db).Update(ctx, wrapper)
}

// Paginate 快捷分页
func Paginate[T any](ctx context.Context, db *gorm.DB, page *Page[T], wrapper *QueryWrapper[T]) (*Page[T], error) {
	return NewServiceImpl[T](db).Page(ctx, page, wrapper)
}
