# GOMP (GO MyBatis-Plus)

GOMP 是一个基于 [GORM](https://gorm.io/) 的增强库，灵感来源于 MyBatis-Plus。它旨在简化 GORM 的开发流程，提供类似于 MyBatis-Plus 的链式查询构造器（Wrapper）和通用的 Service 层 CRUD 接口。

## ✨ 特性

- **链式构造器**: 提供 `QueryWrapper`、`UpdateWrapper`、`DeleteWrapper`，支持流式构建查询条件。
- **通用 Service**: 提供基于泛型的 `IService` 接口和 `ServiceImpl` 实现，开箱即用的 CRUD 方法。
- **内置分页**: 封装 `Page` 对象，轻松实现分页查询。
- **动态条件**: 所有 Wrapper 方法均支持可选的布尔参数，用于根据业务逻辑动态拼接条件。
- **非侵入式**: 完全兼容 GORM 原生用法，可随时获取 `*gorm.DB` 进行原生操作。

## 📦 安装

```bash
go get github.com/shelbeii/gomp
```

## 🚀 快速开始

### 1. 定义模型 (Model)

定义标准的 GORM 模型结构体。

```go
package model

import "time"

type User struct {
    ID        int64     `gorm:"primaryKey"`
    Username  string    `gorm:"size:32;unique"`
    Password  string    `gorm:"size:64"`
    Age       int
    Email     string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 2. 定义 Service

创建一个 Service 结构体，并嵌入 `gomp.ServiceImpl[T]`。

```go
package service

import (
    "github.com/lustfulCap/gomp"
    "your_project/model"
    "gorm.io/gorm"
)

// 定义接口 (可选，推荐)
type IUserService interface {
    gomp.IService[model.User]
    // 在此定义其他自定义业务方法
}

// 实现结构体
type UserService struct {
    *gomp.ServiceImpl[model.User]
}

// 构造函数
func NewUserService(db *gorm.DB) *UserService {
    return &UserService{
        ServiceImpl: gomp.NewServiceImpl[model.User](db),
    }
}
```

### 3. 使用示例

```go
package main

import (
    "context"
    "fmt"
    "github.com/lustfulCap/gomp"
    "gorm.io/driver/sqlite" // 或其他驱动
    "gorm.io/gorm"
)

func main() {
    // 1. 初始化 DB
    db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
    db.AutoMigrate(&model.User{})

    // 2. 创建 Service
    userService := NewUserService(db)
    ctx := context.Background()

    // --- 新增 (Create) ---
    user := &model.User{Username: "tom", Age: 18, Email: "tom@example.com"}
    userService.Save(ctx, user)

    // --- 查询 (Read) ---
    
    // 根据 ID 查询
    u, _ := userService.GetById(ctx, user.ID)
    
    // 复杂条件查询: 名字是 tom 且 (年龄 > 20 或 邮箱不为空)
    w := gomp.NewQueryWrapper[model.User]()
    w.Eq("username", "tom").
      And(func(sub *gomp.QueryWrapper[model.User]) {
          sub.Gt("age", 20).Or().IsNotNull("email")
      })
    
    list, _ := userService.List(ctx, w)

    // --- 分页查询 (Page) ---
    page := gomp.NewPage[model.User](1, 10) // 第1页，每页10条
    query := gomp.NewQueryWrapper[model.User]().Like("username", "t")
    
    resultPage, _ := userService.Page(ctx, page, query)
    fmt.Printf("Total: %d, Records: %d\n", resultPage.Total, len(resultPage.Records))

    // --- 更新 (Update) ---
    
    // 方式1: 根据 ID 更新实体 (只更新非零值)
    u.Age = 25
    userService.UpdateById(ctx, u)

    // 方式2: 使用 UpdateWrapper 指定更新字段和条件
    updater := gomp.NewUpdateWrapper[model.User]()
    updater.Set("age", 30).Set("email", "new@example.com"). // 设置更新的值
            Eq("username", "tom")                           // 设置条件
    userService.Update(ctx, updater)

    // --- 删除 (Delete) ---
    
    // 根据 ID 删除
    userService.RemoveById(ctx, user.ID)
    
    // 根据条件删除
    deleter := gomp.NewDeleteWrapper[model.User]()
    deleter.Le("age", 10) // 删除年龄 <= 10 的
    userService.Delete(ctx, deleter)
}
```

## 🛠️ Wrapper 方法概览

### QueryWrapper 方法详解

`QueryWrapper` 支持大部分常用的 SQL 操作符，以下是详细的使用说明与 SQL 映射关系：

| 方法 | 说明 | 示例代码 | 对应 SQL 结构 (示例) |
| :--- | :--- | :--- | :--- |
| `Eq` | 等于 = | `w.Eq("name", "Tom")` | `name = 'Tom'` |
| `Ne` | 不等于 <> | `w.Ne("status", 1)` | `status <> 1` |
| `Gt` | 大于 > | `w.Gt("age", 18)` | `age > 18` |
| `Ge` | 大于等于 >= | `w.Ge("age", 18)` | `age >= 18` |
| `Lt` | 小于 < | `w.Lt("price", 100)` | `price < 100` |
| `Le` | 小于等于 <= | `w.Le("price", 100)` | `price <= 100` |
| `Like` | 模糊查询 | `w.Like("name", "k")` | `name LIKE '%k%'` |
| `LikeLeft` | 左模糊 | `w.LikeLeft("name", "k")` | `name LIKE '%k'` |
| `LikeRight` | 右模糊 | `w.LikeRight("name", "k")` | `name LIKE 'k%'` |
| `In` | IN 查询 | `w.In("id", []int{1, 2, 3})` | `id IN (1, 2, 3)` |
| `NotIn` | NOT IN 查询 | `w.NotIn("id", []int{1, 2})` | `id NOT IN (1, 2)` |
| `IsNull` | IS NULL | `w.IsNull("deleted_at")` | `deleted_at IS NULL` |
| `IsNotNull` | IS NOT NULL | `w.IsNotNull("email")` | `email IS NOT NULL` |
| `Between` | 区间查询 | `w.Between("age", 18, 30)` | `age BETWEEN 18 AND 30` |
| `NotBetween` | NOT 区间 | `w.NotBetween("age", 18, 30)` | `age NOT BETWEEN 18 AND 30` |
| `Or` | OR 连接 | `w.Eq("a", 1).Or().Eq("b", 2)` | `a = 1 OR b = 2` |
| `Or` (嵌套) | OR 嵌套 | `w.Or(func(sw){ sw.Eq("a", 1).Eq("b", 2) })` | `OR (a = 1 AND b = 2)` |
| `And` | AND 嵌套 | `w.And(func(sw){ sw.Eq("a", 1).Or().Eq("b", 2) })` | `AND (a = 1 OR b = 2)` |
| `Select` | 指定字段 | `w.Select("id", "name", "age")` | `SELECT id, name, age` |
| `Distinct` | 去重 | `w.Distinct("age")` | `SELECT DISTINCT age` |
| `OrderByAsc` | 升序 | `w.OrderByAsc("created_at")` | `ORDER BY created_at ASC` |
| `OrderByDesc` | 降序 | `w.OrderByDesc("score")` | `ORDER BY score DESC` |
| `GroupBy` | 分组 | `w.GroupBy("dept_id")` | `GROUP BY dept_id` |
| `Having` | 分组筛选 | `w.GroupBy("dept").Having("count(*) > ?", 5)` | `GROUP BY dept HAVING count(*) > 5` |
| `LeftJoin` | 左连接 | `w.LeftJoin("user u", "u.id = order.uid")` | `LEFT JOIN user u ON u.id = order.uid` |
| `RightJoin` | 右连接 | `w.RightJoin("user u", "u.id = order.uid")` | `RIGHT JOIN user u ON u.id = order.uid` |
| `InnerJoin` | 内连接 | `w.InnerJoin("user u", "u.id = order.uid")` | `INNER JOIN user u ON u.id = order.uid` |
| `LeftJoinOn` | 左连接(条件构造器) | `w.LeftJoinOn("user u", "u.id", "order.uid", func(on *gomp.JoinOnWrapper){ on.Gt("order.amount", 100) })` | `LEFT JOIN user u ON u.id = order.uid AND order.amount > 100` |
| `RightJoinOn` | 右连接(条件构造器) | `w.RightJoinOn("user u", "u.id", "order.uid", func(on *gomp.JoinOnWrapper){ on.Or().IsNull("order.deleted_at") })` | `RIGHT JOIN user u ON u.id = order.uid OR order.deleted_at IS NULL` |
| `InnerJoinOn` | 内连接(条件构造器) | `w.InnerJoinOn("user u", "u.id", "order.uid", func(on *gomp.JoinOnWrapper){ on.And(func(sw *gomp.JoinOnWrapper){ sw.Gt("order.amount", 100).Or().Gt("order.discount", 0) }) })` | `INNER JOIN user u ON u.id = order.uid AND (order.amount > 100 OR order.discount > 0)` |
| `Table` | 指定表名 | `w.Table("users as u")` | `FROM users as u` |

**Join 条件构造器（JoinOnWrapper）**

`JoinOnWrapper` 用于拼接 JOIN 的 ON 条件，支持 AND / OR 混合与分组，减少手写 SQL 拼接错误。与 `LeftJoinOn` / `RightJoinOn` / `InnerJoinOn` 搭配使用。

**基础用法**（生成的 SQL）

```go
w.LeftJoinOn(
    "t_purchase_contract_component pcc",
    "pcc.id",
    "ind.purchase_contract_component_id",
    func(on *gomp.JoinOnWrapper) {
        on.Gt("ind.purchase_contract_component_id", 0).
            And(func(sw *gomp.JoinOnWrapper) {
                sw.Gt("pcc.id", 0).Or().IsNull("pcc.deleted_at")
            })
    },
)
```
对应 SQL：
```
LEFT JOIN t_purchase_contract_component pcc
  ON pcc.id = ind.purchase_contract_component_id
 AND ind.purchase_contract_component_id > 0
 AND (pcc.id > 0 OR pcc.deleted_at IS NULL)
```

**OR 分组示例**（生成的 SQL）

```go
w.InnerJoinOn(
    "t_integration_notice_detail ind",
    "ind.id",
    "pdod.integration_notice_detail_id",
    func(on *gomp.JoinOnWrapper) {
        on.And(func(sw *gomp.JoinOnWrapper) {
            sw.Eq("ind.notice_status", "4").Or().Eq("ind.notice_status", "5")
        })
    },
)
```
对应 SQL：
```
INNER JOIN t_integration_notice_detail ind
  ON ind.id = pdod.integration_notice_detail_id
 AND (ind.notice_status = '4' OR ind.notice_status = '5')
```

**多条件混合示例**（生成的 SQL）

```go
w.RightJoinOn(
    "t_order o",
    "o.user_id",
    "u.id",
    func(on *gomp.JoinOnWrapper) {
        on.IsNull("o.deleted_at").
           And(func(sw *gomp.JoinOnWrapper){ sw.Gt("o.amount", 100).Or().Gt("o.discount", 0) }).
           And(func(sw *gomp.JoinOnWrapper){ sw.Raw("o.status IN ('paid','shipped')") })
    },
)
```
对应 SQL：
```
RIGHT JOIN t_order o
  ON o.user_id = u.id
 AND o.deleted_at IS NULL
 AND (o.amount > 100 OR o.discount > 0)
 AND (o.status IN ('paid','shipped'))
```
**JoinOnWrapper 常用方法**

| 方法 | 说明 | 示例代码 | 对应 SQL 结构 (示例) |
| :--- | :--- | :--- | :--- |
| `Eq` | 等于 = | `on.Eq("a.id", 1)` | `a.id = 1` |
| `EqColumn` | 列等于列 | `on.EqColumn("a.id", "b.a_id")` | `a.id = b.a_id` |
| `Ne` | 不等于 <> | `on.Ne("a.status", 1)` | `a.status <> 1` |
| `Gt` | 大于 > | `on.Gt("a.amount", 10)` | `a.amount > 10` |
| `Ge` | 大于等于 >= | `on.Ge("a.amount", 10)` | `a.amount >= 10` |
| `Lt` | 小于 < | `on.Lt("a.amount", 10)` | `a.amount < 10` |
| `Le` | 小于等于 <= | `on.Le("a.amount", 10)` | `a.amount <= 10` |
| `Like` | 模糊查询 | `on.Like("a.name", "k")` | `a.name LIKE '%k%'` |
| `LikeLeft` | 左模糊 | `on.LikeLeft("a.name", "k")` | `a.name LIKE '%k'` |
| `LikeRight` | 右模糊 | `on.LikeRight("a.name", "k")` | `a.name LIKE 'k%'` |
| `In` | IN 查询 | `on.In("a.id", []int{1,2})` | `a.id IN (1,2)` |
| `NotIn` | NOT IN 查询 | `on.NotIn("a.id", []int{1,2})` | `a.id NOT IN (1,2)` |
| `IsNull` | IS NULL | `on.IsNull("a.deleted_at")` | `a.deleted_at IS NULL` |
| `IsNotNull` | IS NOT NULL | `on.IsNotNull("a.deleted_at")` | `a.deleted_at IS NOT NULL` |
| `Between` | 区间查询 | `on.Between("a.score", 1, 10)` | `a.score BETWEEN 1 AND 10` |
| `NotBetween` | NOT 区间 | `on.NotBetween("a.score", 1, 10)` | `a.score NOT BETWEEN 1 AND 10` |
| `Or` | OR 连接 | `on.Eq("a.type", 1).Or().Eq("a.type", 2)` | `a.type = 1 OR a.type = 2` |
| `And` | AND 分组 | `on.And(func(sw *gomp.JoinOnWrapper){...})` | `AND (...)` |
| `Raw` | 原始条件 | `on.Raw("a.flag = 1")` | `a.flag = 1` |

### UpdateWrapper 方法详解

`UpdateWrapper` 用于构建更新语句，支持设置更新字段 (`Set`) 以及各种 `WHERE` 条件。

| 方法 | 说明 | 示例代码 | 对应 SQL 结构 (示例) |
| :--- | :--- | :--- | :--- |
| `Set` | 设置更新值 | `w.Set("age", 20)` | `SET age = 20` |
| `SetIncrBy` | 字段自增 | `w.SetIncrBy("count", 1)` | `SET count = count + 1` |
| `SetDecrBy` | 字段自减 | `w.SetDecrBy("stock", 1)` | `SET stock = stock - 1` |
| `Eq` | 等于 = | `w.Eq("name", "Tom")` | `WHERE name = 'Tom'` |
| `Ne` | 不等于 <> | `w.Ne("status", 1)` | `WHERE status <> 1` |
| `Gt` | 大于 > | `w.Gt("age", 18)` | `WHERE age > 18` |
| `Ge` | 大于等于 >= | `w.Ge("age", 18)` | `WHERE age >= 18` |
| `Lt` | 小于 < | `w.Lt("price", 100)` | `WHERE price < 100` |
| `Le` | 小于等于 <= | `w.Le("price", 100)` | `WHERE price <= 100` |
| `Like` | 模糊查询 | `w.Like("name", "k")` | `WHERE name LIKE '%k%'` |
| `LikeLeft` | 左模糊 | `w.LikeLeft("name", "k")` | `WHERE name LIKE '%k'` |
| `LikeRight` | 右模糊 | `w.LikeRight("name", "k")` | `WHERE name LIKE 'k%'` |
| `In` | IN 查询 | `w.In("id", []int{1, 2})` | `WHERE id IN (1, 2)` |
| `NotIn` | NOT IN 查询 | `w.NotIn("id", []int{1, 2})` | `WHERE id NOT IN (1, 2)` |
| `IsNull` | IS NULL | `w.IsNull("deleted_at")` | `WHERE deleted_at IS NULL` |
| `IsNotNull` | IS NOT NULL | `w.IsNotNull("email")` | `WHERE email IS NOT NULL` |
| `Between` | 区间查询 | `w.Between("age", 18, 30)` | `WHERE age BETWEEN 18 AND 30` |
| `NotBetween` | NOT 区间 | `w.NotBetween("age", 18, 30)` | `WHERE age NOT BETWEEN 18 AND 30` |
| `Or` | OR 连接 | `w.Eq("a", 1).Or().Eq("b", 2)` | `WHERE a = 1 OR b = 2` |
| `And` | AND 嵌套 | `w.And(func(sw){...})` | `WHERE ... AND (...)` |
| `Table` | 指定表名 | `w.Table("users u")` | `FROM users u` |

#### 联表更新示例

`UpdateWrapper` 支持 `Join` 语法，可实现多表关联更新。

**简单关联更新**

```go
// UPDATE user u LEFT JOIN order o ON o.user_id = u.id SET u.email = 'vip@example.com' WHERE o.amount > 1000
updater := gomp.NewUpdateWrapper[model.User]()
updater.Table("user u"). // 显式指定别名 u
        LeftJoin("order o", "o.user_id", "u.id").
        Set("u.email", "vip@example.com").
        Gt("o.amount", 1000)
userService.Update(ctx, updater)
```

**复杂条件关联更新**

```go
// 使用 LeftJoinOn 自定义 ON 条件
updater := gomp.NewUpdateWrapper[model.User]()
updater.Table("user u").
        LeftJoinOn("order o", "o.user_id", "u.id", func(on *gomp.JoinOnWrapper) {
            on.Gt("o.amount", 1000).Or().Eq("o.status", "paid")
        }).Set("u.vip_level", 2)

userService.Update(ctx, updater)
```

### DeleteWrapper 方法详解

`DeleteWrapper` 用于构建删除语句，支持各种 `WHERE` 条件。

| 方法 | 说明 | 示例代码 | 对应 SQL 结构 (示例) |
| :--- | :--- | :--- | :--- |
| `Eq` | 等于 = | `w.Eq("name", "Tom")` | `WHERE name = 'Tom'` |
| `Ne` | 不等于 <> | `w.Ne("status", 1)` | `WHERE status <> 1` |
| `Gt` | 大于 > | `w.Gt("age", 18)` | `WHERE age > 18` |
| `Ge` | 大于等于 >= | `w.Ge("age", 18)` | `WHERE age >= 18` |
| `Lt` | 小于 < | `w.Lt("price", 100)` | `WHERE price < 100` |
| `Le` | 小于等于 <= | `w.Le("price", 100)` | `WHERE price <= 100` |
| `Like` | 模糊查询 | `w.Like("name", "k")` | `WHERE name LIKE '%k%'` |
| `LikeLeft` | 左模糊 | `w.LikeLeft("name", "k")` | `WHERE name LIKE '%k'` |
| `LikeRight` | 右模糊 | `w.LikeRight("name", "k")` | `WHERE name LIKE 'k%'` |
| `In` | IN 查询 | `w.In("id", []int{1, 2})` | `WHERE id IN (1, 2)` |
| `NotIn` | NOT IN 查询 | `w.NotIn("id", []int{1, 2})` | `WHERE id NOT IN (1, 2)` |
| `IsNull` | IS NULL | `w.IsNull("deleted_at")` | `WHERE deleted_at IS NULL` |
| `IsNotNull` | IS NOT NULL | `w.IsNotNull("email")` | `WHERE email IS NOT NULL` |
| `Between` | 区间查询 | `w.Between("age", 18, 30)` | `WHERE age BETWEEN 18 AND 30` |
| `NotBetween` | NOT 区间 | `w.NotBetween("age", 18, 30)` | `WHERE age NOT BETWEEN 18 AND 30` |
| `Or` | OR 连接 | `w.Eq("a", 1).Or().Eq("b", 2)` | `WHERE a = 1 OR b = 2` |
| `And` | AND 嵌套 | `w.And(func(sw){...})` | `WHERE ... AND (...)` |

#### 联表删除示例

`DeleteWrapper` 支持 `Join` 语法，可实现多表关联删除。

**简单关联删除**

```go
// DELETE u FROM user u LEFT JOIN order o ON o.user_id = u.id WHERE o.status = 'cancelled'
deleter := gomp.NewDeleteWrapper[model.User]()
deleter.Table("user u"). // 显式指定别名 u
        LeftJoin("order o", "o.user_id", "u.id").
        Eq("o.status", "cancelled")
userService.Delete(ctx, deleter)
```

**复杂条件关联删除**

```go
// 使用 LeftJoinOn 自定义 ON 条件
deleter := gomp.NewDeleteWrapper[model.User]()
deleter.Table("user u").
        LeftJoinOn("login_log l", "l.user_id", "u.id", func(on *gomp.JoinOnWrapper) {
            on.Lt("l.login_time", "2023-01-01")
        }).IsNull("u.active_at") // 删除很久没登录且未激活的用户

userService.Delete(ctx, deleter)
```

### InsertWrapper 方法详解

`InsertWrapper` 用于构建插入语句，主要用于指定插入的字段和值。

| 方法 | 说明 | 示例代码 | 对应 SQL 结构 (示例) |
| :--- | :--- | :--- | :--- |
| `Set` | 设置插入值 | `w.Set("name", "Tom")` | `INSERT INTO ... (name) VALUES ('Tom')` |

> **提示**: 所有方法最后一个参数支持传入 `bool` 类型条件。例如：`w.Eq("name", name, name != "")`，只有当 `name != ""` 为 true 时，该条件才会生效。

## 📋 要求

- Go 1.18+ (泛型支持)
- GORM v1.20+
