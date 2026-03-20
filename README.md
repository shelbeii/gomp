# GOMP — Go MyBatis-Plus

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

GOMP 是一个基于 [GORM](https://gorm.io/) 的增强库，灵感来源于 Java 的 MyBatis-Plus。
它通过泛型提供链式条件构造器（Wrapper）和通用 Service 层，让 GORM 开发更简洁、更安全。

---

## 特性

- **链式构造器** — `QueryWrapper`、`UpdateWrapper`、`DeleteWrapper`、`InsertWrapper`，流式构建复杂 SQL 条件
- **通用 Service** — 泛型 `IService[T]` 接口 + `ServiceImpl[T]` 实现，开箱即用的 CRUD
- **内置分页** — `Page[T]` 对象 + `SelectPage` / `Page` 方法，一行代码完成分页
- **动态条件** — 所有条件方法支持末尾可选 `bool` 参数，轻松实现动态拼接
- **嵌套逻辑** — `And(func)` / `Or(func)` / `AndOr(func...)` 支持任意深度嵌套括号
- **联表查询** — `LeftJoinOn` / `RightJoinOn` / `InnerJoinOn` 配合 `JoinOnWrapper` 构建复杂 ON 条件
- **悲观锁** — `ForUpdate()` / `ForShare()` 一键添加行锁
- **Upsert** — `InsertWrapper` 支持 `OnConflictDoNothing` 和 `OnConflictDoUpdate`
- **并发安全配置** — `atomic.Pointer` 实现无锁读，支持运行时热更新
- **非侵入式** — 完全兼容 GORM 原生用法，随时可通过 `GetDB()` 降级

---

## 安装

```bash
go get github.com/shelbeii/gomp
```

> 要求 Go 1.21+，GORM v1.20+

---

## 快速开始

### 1. 定义模型

```go
type User struct {
    ID        int64     `gorm:"primaryKey"`
    Username  string    `gorm:"size:32;unique"`
    Age       int
    Email     string
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"` // 软删除
}
```

### 2. 初始化配置（可选）

```go
// 方式一：通过代码设置
gomp.SetConfig(gomp.GompConfig{
    EnableSQLPrint:    true,   // 打印 SQL 日志
    AllowGlobalUpdate: false,  // 禁止无 WHERE 的全表更新
    AllowGlobalDelete: false,  // 禁止无 WHERE 的全表删除
    SaveBatchSize:     200,    // 批量插入每批大小（默认 100）
    PageMaxSize:       500,    // 分页最大条数上限（默认 1000）
})

// 方式二：从 YAML 文件读取
gomp.InitConfig("config/app.yaml")
```

YAML 示例：
```yaml
gomp:
  enableSqlPrint: false
  allowGlobalUpdate: false
  allowGlobalDelete: false
  saveBatchSize: 100
  pageMaxSize: 1000
```

### 3. 创建 Service

```go
type IUserService interface {
    gomp.IService[User]
    // 自定义业务方法
    FindActiveUsers(ctx context.Context) ([]*User, error)
}

type UserService struct {
    *gomp.ServiceImpl[User]
}

func NewUserService(db *gorm.DB) *UserService {
    return &UserService{ServiceImpl: gomp.NewServiceImpl[User](db)}
}

// 自定义方法示例
func (s *UserService) FindActiveUsers(ctx context.Context) ([]*User, error) {
    return s.List(ctx, gomp.NewQueryWrapper[User]().IsNotNull("email"))
}
```

### 4. 基础 CRUD

```go
ctx := context.Background()
userSvc := NewUserService(db)

// 新增
user := &User{Username: "tom", Age: 18, Email: "tom@example.com"}
userSvc.Save(ctx, user)

// 批量新增（自动按 SaveBatchSize 分批）
userSvc.SaveBatch(ctx, users)
// 也可临时指定批次大小
userSvc.SaveBatch(ctx, users, 50)

// 根据 ID 查询
u, _ := userSvc.GetById(ctx, 1)

// 根据 ID 更新（只更新非零字段）
u.Age = 25
userSvc.UpdateById(ctx, u)

// 根据 ID 删除
userSvc.RemoveById(ctx, 1)

// 批量 ID 删除
userSvc.RemoveByIds(ctx, []int64{1, 2, 3})
```

---

## QueryWrapper 详解

### 基础条件

```go
w := gomp.NewQueryWrapper[User]().
    Eq("username", "tom").          // username = 'tom'
    Ge("age", 18).                  // AND age >= 18
    Like("email", "gmail").         // AND email LIKE '%gmail%'
    In("status", []int{1, 2}).      // AND status IN (1, 2)
    IsNotNull("deleted_at")         // AND deleted_at IS NOT NULL

list, _ := userSvc.List(ctx, w)
```

### 动态条件（末尾 bool 参数）

```go
keyword := ""  // 来自前端，可能为空
minAge  := 0

w := gomp.NewQueryWrapper[User]().
    Like("username", keyword, keyword != "").  // keyword 为空时忽略此条件
    Ge("age", minAge, minAge > 0)             // minAge <= 0 时忽略

list, _ := userSvc.List(ctx, w)
```

### OR 条件

```go
// 简单 OR：a = 1 OR b = 2
w := gomp.NewQueryWrapper[User]().
    Eq("status", 1).Or().Eq("status", 2)

// 嵌套 OR：AND (a = 1 OR b = 2)
w := gomp.NewQueryWrapper[User]().
    Eq("role", "admin").
    Or(func(sw *gomp.QueryWrapper[User]) {
        sw.Eq("status", 1).Eq("verified", true)
    })
// => WHERE role = 'admin' OR (status = 1 AND verified = true)

// 多分支 OR 合并后 AND：AND ((c1) OR (c2) OR (c3))
w := gomp.NewQueryWrapper[User]().
    Eq("role", "user").
    And(func(sw *gomp.QueryWrapper[User]) {
        sw.Eq("status", 1)
    }).
    AndOr(
        func(sw *gomp.QueryWrapper[User]) { sw.Between("age", 18, 25) },
        func(sw *gomp.QueryWrapper[User]) { sw.Between("age", 60, 70) },
    )
// => WHERE role = 'user' AND status = 1
//    AND ((age BETWEEN 18 AND 25) OR (age BETWEEN 60 AND 70))
```

### 分页查询

```go
// 方式一：使用 Page 对象
page := gomp.NewPage[User](1, 10) // 第 1 页，每页 10 条
w := gomp.NewQueryWrapper[User]().Like("username", "t").OrderByDesc("created_at")
result, _ := userSvc.Page(ctx, page, w)
fmt.Printf("总数: %d, 总页数: %d, 当前页记录数: %d\n",
    result.Total, result.Pages(), len(result.Records))
fmt.Println("有下一页:", result.HasNext())

// 方式二：直接传页码和条数
result, _ := userSvc.SelectPage(ctx, 2, 20, w)
```

### 排序、分组、去重

```go
w := gomp.NewQueryWrapper[User]().
    Select("dept_id", "count(*) as cnt"). // 指定查询字段
    GroupBy("dept_id").                   // GROUP BY
    Having("count(*) > ?", 5).            // HAVING
    OrderByDesc("cnt").                   // ORDER BY cnt DESC
    Distinct("dept_id")                   // DISTINCT
```

### 手动分页（Limit / Offset）

```go
w := gomp.NewQueryWrapper[User]().
    OrderByAsc("id").
    Limit(20).
    Offset(40) // 第 3 页（每页 20 条）

list, _ := userSvc.List(ctx, w)
```

### 悲观锁

```go
// SELECT ... FOR UPDATE（需在事务中使用）
db.Transaction(func(tx *gorm.DB) error {
    svc := gomp.NewServiceImpl[User](tx)
    w := gomp.NewQueryWrapper[User]().Eq("id", 1).ForUpdate()
    u, _ := svc.GetOne(ctx, w)
    // ... 修改并保存
    return svc.UpdateById(ctx, u)
})

// SELECT ... FOR SHARE
w := gomp.NewQueryWrapper[User]().Eq("id", 1).ForShare()
```

### 联表查询

```go
// 简单 JOIN
w := gomp.NewQueryWrapper[UserVO]().
    Table("user u").
    Select("u.id", "u.username", "o.amount").
    LeftJoin("order o", "o.user_id", "u.id").
    Gt("o.amount", 100)

// 带自定义 ON 条件的 JOIN
w := gomp.NewQueryWrapper[UserVO]().
    Table("user u").
    LeftJoinOn("order o", "o.user_id", "u.id", func(on *gomp.JoinOnWrapper) {
        on.IsNull("o.deleted_at").
           And(func(sw *gomp.JoinOnWrapper) {
               sw.Eq("o.status", "paid").Or().Gt("o.amount", 500)
           })
    }).
    Gt("o.amount", 100)
// LEFT JOIN order o ON o.user_id = u.id
//   AND o.deleted_at IS NULL
//   AND (o.status = 'paid' OR o.amount > 500)
```

### 列间比较 EqColumn

```go
// WHERE a.ref_id = b.id
w := gomp.NewQueryWrapper[MyVO]().
    Table("table_a a").
    InnerJoin("table_b b", "b.id", "a.ref_id").
    EqColumn("a.created_by", "b.owner_id") // WHERE a.created_by = b.owner_id
```

### 原生 SQL 片段

```go
w := gomp.NewQueryWrapper[User]().
    Raw("DATE(created_at) = ?", "2024-01-01").
    Gt("age", 18)
```

### 全部方法速查

| 方法 | 说明 | SQL 片段示例 |
|------|------|--------------|
| `Eq(col, val)` | 等于 | `col = ?` |
| `Ne(col, val)` | 不等于 | `col <> ?` |
| `Gt(col, val)` | 大于 | `col > ?` |
| `Ge(col, val)` | 大于等于 | `col >= ?` |
| `Lt(col, val)` | 小于 | `col < ?` |
| `Le(col, val)` | 小于等于 | `col <= ?` |
| `Like(col, val)` | 模糊 | `col LIKE '%val%'` |
| `LikeLeft(col, val)` | 左模糊 | `col LIKE '%val'` |
| `LikeRight(col, val)` | 右模糊 | `col LIKE 'val%'` |
| `In(col, slice)` | IN | `col IN (...)` |
| `NotIn(col, slice)` | NOT IN | `col NOT IN (...)` |
| `IsNull(col)` | NULL | `col IS NULL` |
| `IsNotNull(col)` | NOT NULL | `col IS NOT NULL` |
| `Between(col, v1, v2)` | 区间 | `col BETWEEN ? AND ?` |
| `NotBetween(col, v1, v2)` | NOT 区间 | `col NOT BETWEEN ? AND ?` |
| `EqColumn(left, right)` | 列间比较 | `left = right` |
| `Raw(sql, args...)` | 原生片段 | 原样插入 |
| `Or()` | 下一个条件用 OR | — |
| `Or(f1, f2, ...)` | 多子条件 OR 分组 | `OR (f1 OR f2 ...)` |
| `And(f1, f2, ...)` | 多子条件 AND 分组 | `AND (f1 AND f2 ...)` |
| `AndOr(f1, f2, ...)` | 多子条件 OR 后整体 AND | `AND (f1 OR f2 ...)` |
| `Select(cols...)` | 指定字段 | `SELECT ...` |
| `Distinct(args...)` | 去重 | `DISTINCT` |
| `Table(name)` | 指定表名/别名 | `FROM name` |
| `OrderByAsc(col)` | 升序 | `ORDER BY col ASC` |
| `OrderByDesc(col)` | 降序 | `ORDER BY col DESC` |
| `GroupBy(cols...)` | 分组 | `GROUP BY ...` |
| `Having(sql, args...)` | 分组筛选 | `HAVING ...` |
| `Limit(n)` | 限制条数 | `LIMIT n` |
| `Offset(n)` | 偏移量 | `OFFSET n` |
| `ForUpdate()` | 悲观排他锁 | `FOR UPDATE` |
| `ForShare()` | 悲观共享锁 | `FOR SHARE` |
| `LeftJoin(table, left, right)` | 左连接 | `LEFT JOIN ... ON ...` |
| `RightJoin(table, left, right)` | 右连接 | `RIGHT JOIN ... ON ...` |
| `InnerJoin(table, left, right)` | 内连接 | `INNER JOIN ... ON ...` |
| `LeftJoinOn(table, left, right, fn)` | 自定义 ON 左连接 | `LEFT JOIN ... ON ... AND ...` |
| `RightJoinOn(table, left, right, fn)` | 自定义 ON 右连接 | `RIGHT JOIN ... ON ... AND ...` |
| `InnerJoinOn(table, left, right, fn)` | 自定义 ON 内连接 | `INNER JOIN ... ON ... AND ...` |

---

## UpdateWrapper 详解

```go
// 基础更新
w := gomp.NewUpdateWrapper[User]().
    Set("age", 30).
    Set("email", "new@example.com").
    Eq("username", "tom")
userSvc.Update(ctx, w)

// 字段自增 / 自减
w := gomp.NewUpdateWrapper[User]().
    SetIncrBy("login_count", 1).  // login_count = login_count + 1
    SetDecrBy("credits", 10).     // credits = credits - 10
    Eq("id", 1)

// 表达式更新（SetRaw）
w := gomp.NewUpdateWrapper[User]().
    SetRaw("meta", "JSON_SET(meta, '$.vip', ?)", true).
    Eq("id", 1)

// 联表更新
w := gomp.NewUpdateWrapper[User]().
    Table("user u").
    LeftJoin("order o", "o.user_id", "u.id").
    Set("u.vip", 1).
    Gt("o.amount", 1000)
userSvc.Update(ctx, w)

// 批量更新
// 在循环中构建不同类型的 wrapper，统一放入 []UpdateExecutor
executors := make([]gomp.UpdateExecutor, 0)

for _, u := range userList {
executors = append(executors,
gomp.NewUpdateWrapper[User]().Set("status", u.Status).Eq("id", u.ID),
)
}

for _, o := range orderList {
executors = append(executors,
gomp.NewUpdateWrapper[Order]().Set("amount", o.Amount).Eq("id", o.ID),
)
}

for _, p := range productList {
executors = append(executors,
gomp.NewUpdateWrapper[Product]().SetIncrBy("stock", -1).Eq("id", p.ID),
)
}

// 不使用事务
err := gomp.UpdateBatchAny(ctx, db, executors)

// 使用事务
err := gomp.UpdateBatchAny(ctx, db, executors, true)

// 复用外部事务
db.Transaction(func(tx *gorm.DB) error {
return gomp.UpdateBatchAny(ctx, tx, executors)
})

```

| 方法 | 说明 |
|------|------|
| `Set(col, val)` | 设置更新值 |
| `SetIncrBy(col, val)` | 字段自增 `col = col + val` |
| `SetDecrBy(col, val)` | 字段自减 `col = col - val` |
| `SetRaw(col, expr, args...)` | 表达式更新 |
| `Table(name)` | 指定表名/别名 |
| `EqColumn(left, right)` | 列间比较 |
| `Raw(sql, args...)` | 原生条件 |
| 所有条件方法 | 同 QueryWrapper |
| `LeftJoin` / `RightJoin` / `InnerJoin` | 联表更新 |
| `LeftJoinOn` / `RightJoinOn` / `InnerJoinOn` | 自定义 ON 联表更新 |

---

## DeleteWrapper 详解

```go
// 软删除（默认）
w := gomp.NewDeleteWrapper[User]().Le("age", 10)
userSvc.Delete(ctx, w)

// 硬删除（物理删除）
w := gomp.NewDeleteWrapper[User]().
    UseSoftDelete(false).
    Eq("id", 1)
userSvc.Delete(ctx, w)

// 联表删除
w := gomp.NewDeleteWrapper[User]().
    Table("user u").
    LeftJoinOn("login_log l", "l.user_id", "u.id", func(on *gomp.JoinOnWrapper) {
        on.Lt("l.login_time", "2023-01-01")
    }).
    IsNull("u.active_at")
userSvc.Delete(ctx, w)
```

---

## InsertWrapper 详解

```go
// 普通插入
w := gomp.NewInsertWrapper[User]().
    Set("username", "tom").
    Set("age", 18)
userSvc.Insert(ctx, w)

// 冲突时忽略
w := gomp.NewInsertWrapper[User]().
    Set("username", "tom").Set("age", 18).
    OnConflictDoNothing()

// 冲突时更新所有字段（Upsert）
w := gomp.NewInsertWrapper[User]().
    Set("username", "tom").Set("age", 20).
    OnConflictDoUpdate([]string{"username"})

// 冲突时只更新指定字段
w := gomp.NewInsertWrapper[User]().
    Set("username", "tom").Set("age", 20).Set("email", "new@example.com").
    OnConflictDoUpdate([]string{"username"}, "age", "email")
```

---

## JoinOnWrapper 详解

```go
w.LeftJoinOn(
    "t_order o", "o.user_id", "u.id",
    func(on *gomp.JoinOnWrapper) {
        on.IsNull("o.deleted_at").
           And(func(sw *gomp.JoinOnWrapper) {
               sw.Eq("o.status", "paid").Or().Gt("o.amount", 500)
           }).
           Raw("o.flag = 1")
    },
)
// LEFT JOIN t_order o
//   ON o.user_id = u.id
//   AND o.deleted_at IS NULL
//   AND (o.status = 'paid' OR o.amount > 500)
//   AND o.flag = 1
```

| 方法 | 说明 |
|------|------|
| `Eq` / `Ne` / `Gt` / `Ge` / `Lt` / `Le` | 比较操作符 |
| `EqColumn(left, right)` | 列间等值比较 |
| `Like` / `LikeLeft` / `LikeRight` | 模糊匹配 |
| `In` / `NotIn` | IN 查询 |
| `IsNull` / `IsNotNull` | NULL 判断 |
| `Between` / `NotBetween` | 区间判断 |
| `Or()` | 下一个条件用 OR |
| `Or(fn)` | OR 嵌套子句 |
| `And(fn)` | AND 嵌套子句 |
| `Raw(sql, args...)` | 原生条件片段 |

---

## 包级快捷函数

无需创建 Service 实例，直接传入 `*gorm.DB`：

```go
// 查询
list, _  := gomp.SelectList[User](ctx, db, wrapper)
one, _   := gomp.SelectOne[User](ctx, db, wrapper)
page, _  := gomp.SelectPage[User](ctx, db, 1, 10, wrapper)
count, _ := gomp.Count[User](ctx, db, wrapper)
u, _     := gomp.GetById[User](ctx, db, 1)

// 写入
gomp.Save[User](ctx, db, &user)
gomp.SaveBatch[User](ctx, db, users)
gomp.SaveBatch[User](ctx, db, users, 50) // 指定批次大小
gomp.UpdateById[User](ctx, db, &user)
gomp.RemoveById[User](ctx, db, 1)
gomp.RemoveByIds[User](ctx, db, []int64{1, 2, 3})

// Wrapper 操作
gomp.Insert[User](ctx, db, insertWrapper)
gomp.Update[User](ctx, db, updateWrapper)
gomp.Delete[User](ctx, db, deleteWrapper)
gomp.Paginate[User](ctx, db, page, wrapper)
```

---

## Page 对象

```go
page := gomp.NewPage[User](1, 20) // current=1, size=20

result, _ := userSvc.Page(ctx, page, wrapper)
result.Total     // 总记录数
result.Records   // 当前页数据 []*User
result.Pages()   // 总页数
result.HasNext() // 是否有下一页
result.HasPrev() // 是否有上一页
result.Offset()  // 当前页偏移量
result.Limit()   // 当前页大小
```

---

## 配置说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `EnableSQLPrint` | `bool` | `false` | 打印完整 SQL 日志 |
| `AllowGlobalUpdate` | `bool` | `false` | 允许无 WHERE 全表更新 |
| `AllowGlobalDelete` | `bool` | `false` | 允许无 WHERE 全表删除 |
| `SaveBatchSize` | `int` | `100` | 批量插入每批大小，范围 [1, 5000] |
| `PageMaxSize` | `int` | `1000` | 分页最大条数上限，范围 [1, 10000] |

---

## 许可证

[MIT License](LICENSE)
 