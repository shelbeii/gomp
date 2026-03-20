package gomp

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// withCtx 内部辅助：按配置附加 ctx 和 Debug
func withCtx(db *gorm.DB, ctx context.Context) *gorm.DB {
	if getConfig().Gomp.EnableSQLPrint {
		return db.WithContext(ctx).Debug()
	}
	return db.WithContext(ctx)
}

// SelectPage 快捷分页查询
func SelectPage[T any](ctx context.Context, db *gorm.DB, current, size int64, wrapper *QueryWrapper[T]) (*Page[T], error) {
	entities := make([]*T, 0)
	d := withCtx(db, ctx)

	// 先应用 wrapper 的条件
	if wrapper != nil {
		d = wrapper.Apply(d)
	}

	// 若 wrapper 未指定 Table，则尝试用 Model(new(T)) 推断表名（普通 Model 场景）
	// 注意：Model 必须在 wrapper.Apply 之后设置，这样才能正确应用软删除等条件
	if d.Statement == nil || d.Statement.Table == "" {
		d = d.Model(new(T))
	}

	page := NewPage[T](current, size)
	var total int64
	// 用 Session 克隆当前 db 做 COUNT，保留 Table/JOIN/WHERE 和 Model 的软删除条件
	// 注意：必须使用 Model(new(T)) 来确保 COUNT 也应用软删除条件
	countDB := d.Session(&gorm.Session{})
	// 如果 wrapper 指定了 Table（联表查询），需要重新设置 Model 以应用软删除
	if wrapper != nil && d.Statement != nil && d.Statement.Table != "" {
		countDB = countDB.Model(new(T))
	}
	if err := countDB.Count(&total).Error; err != nil {
		return nil, err
	}
	page.Total = total
	if total == 0 {
		return page, nil
	}
	if page.Size > 0 {
		d = d.Offset(page.Offset()).Limit(page.Limit())
	}
	if err := d.Find(&entities).Error; err != nil {
		return nil, err
	}
	page.Records = entities
	return page, nil
}

// SelectList 快捷列表查询
func SelectList[T any](ctx context.Context, db *gorm.DB, wrapper *QueryWrapper[T]) ([]*T, error) {
	entities := make([]*T, 0)
	d := withCtx(db, ctx)
	if wrapper != nil {
		d = wrapper.Apply(d)
	}
	err := d.Find(&entities).Error
	return entities, err
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
	err := d.Count(&total).Error
	return total, err
}

// Insert 快捷插入
func Insert[T any](ctx context.Context, db *gorm.DB, wrapper *InsertWrapper[T]) error {
	if wrapper == nil {
		return errors.New("insert wrapper cannot be nil")
	}
	if wrapper.IsEmpty() {
		return errors.New("insert wrapper has no fields set")
	}
	d := withCtx(db, ctx).Model(new(T))
	if wrapper.conflictAction != 0 {
		oc, err := wrapper.buildClause()
		if err != nil {
			return err
		}
		d = d.Clauses(oc)
	}
	return d.Create(wrapper.values).Error
}

// Delete 快捷删除
func Delete[T any](ctx context.Context, db *gorm.DB, wrapper *DeleteWrapper[T]) error {
	if !getConfig().Gomp.AllowGlobalDelete {
		if wrapper == nil || !wrapper.hasWhereConditions() {
			return errors.New("global delete is not allowed without WHERE clause; set AllowGlobalDelete=true to override")
		}
	}
	d := withCtx(db, ctx)
	useSoftDelete := true
	if wrapper != nil {
		useSoftDelete = wrapper.useSoftDelete
		d = wrapper.Apply(d)
	}
	if !useSoftDelete {
		d = d.Unscoped()
	}
	return d.Delete(new(T)).Error
}

// UpdateBatch 批量更新，在单次数据库连接中依次执行多个 UpdateWrapper
// useTransaction: 可选参数，传 true 时开启事务（任意失败则全部回滚）；
// 默认为 false，适用于调用方已自行管理事务的场景
func UpdateBatch[T any](ctx context.Context, db *gorm.DB, wrappers []*UpdateWrapper[T], useTransaction ...bool) error {
	if len(wrappers) == 0 {
		return nil
	}

	// 预校验所有 wrapper，避免执行中途才发现错误
	for i, wrapper := range wrappers {
		if wrapper == nil {
			return errors.New("update wrapper at index " + itoa(i) + " cannot be nil")
		}
		if !wrapper.HasValues() {
			return errors.New("update wrapper at index " + itoa(i) + " has no values to update")
		}
		if !getConfig().Gomp.AllowGlobalUpdate && !wrapper.hasWhereConditions() {
			return errors.New("update wrapper at index " + itoa(i) + ": global update is not allowed without WHERE clause; set AllowGlobalUpdate=true to override")
		}
	}

	execUpdates := func(tx *gorm.DB) error {
		for _, wrapper := range wrappers {
			d := wrapper.Apply(tx)
			var err error
			if wrapper.hasJoin {
				err = execJoinUpdate(d, wrapper.values)
			} else if wrapper.tableName != "" {
				err = d.Updates(wrapper.values).Error
			} else {
				err = d.Model(new(T)).Updates(wrapper.values).Error
			}
			if err != nil {
				return err
			}
		}
		return nil
	}

	d := withCtx(db, ctx)
	if len(useTransaction) > 0 && useTransaction[0] {
		return d.Transaction(execUpdates)
	}
	return execUpdates(d)
}

// itoa 将整数转换为字符串（内部工具函数）
func itoa(i int) string {
	return strconv.Itoa(i)
}

// Update 快捷更新
func Update[T any](ctx context.Context, db *gorm.DB, wrapper *UpdateWrapper[T]) error {
	if wrapper == nil {
		return errors.New("update wrapper cannot be nil")
	}

	// 检查是否有要更新的值
	if !wrapper.HasValues() {
		return errors.New("no values to update")
	}

	// 检查全局更新保护
	if !getConfig().Gomp.AllowGlobalUpdate && !wrapper.hasWhereConditions() {
		return errors.New("global update is not allowed without WHERE clause; set AllowGlobalUpdate=true to override")
	}

	d := wrapper.Apply(withCtx(db, ctx))

	// 联表 UPDATE：使用原生 SQL 执行
	if wrapper.hasJoin {
		return execJoinUpdate(d, wrapper.values)
	}

	// 普通 UPDATE：使用 GORM 的 Updates 方法（会触发钩子和自动更新）
	if wrapper.tableName != "" {
		return d.Updates(wrapper.values).Error
	}
	return d.Model(new(T)).Updates(wrapper.values).Error
}

// execJoinUpdate 执行联表 UPDATE
// 注意：联表更新使用原生 SQL，不会触发 GORM 钩子
// 如需自动更新时间字段，请手动添加：SetRaw("updated_at", "NOW()") 或 Set("updated_at", time.Now())
func execJoinUpdate(db *gorm.DB, values map[string]any) error {
	if len(values) == 0 {
		return errors.New("no values to update")
	}

	// 通过 DryRun 获取完整的 SELECT 语句（包含 JOIN）
	dryStmt := db.Session(&gorm.Session{DryRun: true}).Find(nil)
	if dryStmt.Error != nil {
		return dryStmt.Error
	}

	selectSQL := dryStmt.Statement.SQL.String()
	if selectSQL == "" {
		return errors.New("failed to generate SELECT statement")
	}
	selectVars := dryStmt.Statement.Vars

	// 解析 SQL 结构
	fromIdx := findKeywordIndex(selectSQL, "FROM")
	if fromIdx < 0 {
		return errors.New("invalid SQL: FROM not found")
	}

	fromPart := selectSQL[fromIdx+4:] // 跳过 "FROM"
	if len(fromPart) == 0 {
		return errors.New("invalid SQL: empty FROM clause")
	}

	// 查找关键字位置
	whereIdx := findKeywordIndex(fromPart, "WHERE")

	// 查找第一个 JOIN（可能是 LEFT JOIN, RIGHT JOIN, INNER JOIN 或 JOIN）
	joinIdx := findFirstJoinIndex(fromPart)
	if joinIdx < 0 {
		return errors.New("no JOIN found in query")
	}

	// 确定 JOIN 子句的结束位置
	joinEndIdx := len(fromPart)
	if whereIdx >= 0 {
		joinEndIdx = whereIdx
	}

	// 提取主表信息（表名和别名）
	mainTablePart := strings.TrimSpace(fromPart[:joinIdx])
	tableParts := strings.Fields(mainTablePart)
	if len(tableParts) < 2 {
		return errors.New("table must have an alias for JOIN UPDATE")
	}
	mainTable := tableParts[0]
	mainAlias := tableParts[1]

	// 构建 SET 子句
	setClauses := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values)+len(selectVars))

	for col, val := range values {
		if expr, ok := val.(clause.Expr); ok {
			setClauses = append(setClauses, db.Statement.Quote(col)+" = "+expr.SQL)
			args = append(args, expr.Vars...)
		} else {
			setClauses = append(setClauses, db.Statement.Quote(col)+" = ?")
			args = append(args, val)
		}
	}

	// 构建完整的 UPDATE SQL
	var sqlBuilder strings.Builder
	sqlBuilder.Grow(100 + len(fromPart) + len(values)*30)

	sqlBuilder.WriteString("UPDATE ")
	sqlBuilder.WriteString(db.Statement.Quote(mainTable))
	sqlBuilder.WriteString(" ")
	sqlBuilder.WriteString(mainAlias)
	sqlBuilder.WriteString(" ")                          // 添加空格分隔主表和 JOIN
	sqlBuilder.WriteString(fromPart[joinIdx:joinEndIdx]) // JOIN 子句
	sqlBuilder.WriteString(" SET ")
	sqlBuilder.WriteString(strings.Join(setClauses, ", "))

	// 添加 WHERE 子句
	if whereIdx >= 0 {
		sqlBuilder.WriteString(" ")
		sqlBuilder.WriteString(fromPart[whereIdx:])
		args = append(args, selectVars...)
	}

	return db.Exec(sqlBuilder.String(), args...).Error
}

// findFirstJoinIndex 查找第一个 JOIN 关键字的位置（包括 LEFT JOIN, RIGHT JOIN, INNER JOIN）
func findFirstJoinIndex(sql string) int {
	// 按优先级查找：LEFT JOIN, RIGHT JOIN, INNER JOIN, JOIN
	joinTypes := []string{"LEFT JOIN", "RIGHT JOIN", "INNER JOIN", "JOIN"}

	minIdx := -1
	for _, joinType := range joinTypes {
		idx := findKeywordIndex(sql, joinType)
		if idx >= 0 && (minIdx < 0 || idx < minIdx) {
			minIdx = idx
		}
	}

	return minIdx
}

// findKeywordIndex 使用大小写不敏感的方式查找 SQL 关键字
// 比逐字符遍历更高效，使用 strings.Index 的优化算法
func findKeywordIndex(sql, keyword string) int {
	// 避免重复转换，只转换一次
	sqlLen := len(sql)
	keywordLen := len(keyword)

	if sqlLen < keywordLen {
		return -1
	}

	// 使用 strings.EqualFold 进行大小写不敏感比较
	// 查找独立的关键字（前后有空格或在开头/结尾）
	for i := 0; i <= sqlLen-keywordLen; i++ {
		// 检查是否匹配关键字
		if strings.EqualFold(sql[i:i+keywordLen], keyword) {
			// 检查前面是否是空格或开头
			if i > 0 && sql[i-1] != ' ' {
				continue
			}
			// 检查后面是否是空格或结尾
			if i+keywordLen < sqlLen && sql[i+keywordLen] != ' ' {
				continue
			}
			return i
		}
	}

	return -1
}
