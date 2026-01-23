# GOMP (GORM Plus)

GOMP 是一个基于 [GORM](https://gorm.io/) 的增强库，灵感来源于 MyBatis-Plus。它旨在简化 GORM 的开发流程，提供类似于 MyBatis-Plus 的链式查询构造器（Wrapper）和通用的 Service 层 CRUD 接口。

## ✨ 特性

- **链式构造器**: 提供 `QueryWrapper`、`UpdateWrapper`、`DeleteWrapper`，支持流式构建查询条件。
- **通用 Service**: 提供基于泛型的 `IService` 接口和 `ServiceImpl` 实现，开箱即用的 CRUD 方法。
- **内置分页**: 封装 `Page` 对象，轻松实现分页查询。
- **动态条件**: 所有 Wrapper 方法均支持可选的布尔参数，用于根据业务逻辑动态拼接条件。
- **非侵入式**: 完全兼容 GORM 原生用法，可随时获取 `*gorm.DB` 进行原生操作。

## 📦 安装

```bash
go get github.com/lustfulCap/gomp
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

`QueryWrapper`、`UpdateWrapper`、`DeleteWrapper` 支持大部分常用的 SQL 操作符：

| 方法 | 说明 | 示例 |
| --- | --- | --- |
| `Eq` | 等于 = | `w.Eq("name", "Tom")` |
| `Ne` | 不等于 <> | `w.Ne("status", 1)` |
| `Gt` / `Ge` | 大于 / 大于等于 | `w.Gt("age", 18)` |
| `Lt` / `Le` | 小于 / 小于等于 | `w.Lt("score", 60)` |
| `Like` | 模糊查询 | `w.Like("name", "To")` |
| `LikeLeft` / `LikeRight` | 左/右模糊 | `w.LikeRight("name", "To")` |
| `In` / `NotIn` | IN 查询 | `w.In("id", []int{1, 2, 3})` |
| `Between` / `NotBetween` | 区间查询 | `w.Between("age", 18, 30)` |
| `IsNull` / `IsNotNull` | NULL 值判断 | `w.IsNull("deleted_at")` |
| `And` | 嵌套 AND | `w.And(func(sw){...})` |
| `Or` | OR 连接 | `w.Or()` 或 `w.Or(func(sw){...})` |
| `OrderByAsc` / `OrderByDesc` | 排序 | `w.OrderByDesc("created_at")` |
| `Select` | 指定查询字段 | `w.Select("id", "name")` |

> **提示**: 所有方法最后一个参数支持传入 `bool` 类型条件。例如：`w.Eq("name", name, name != "")`，只有当 `name != ""` 为 true 时，该条件才会生效。

## 📋 要求

- Go 1.18+ (泛型支持)
- GORM v1.20+
