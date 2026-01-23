# GOMP (GORM Plus)

A powerful and simple wrapper for GORM, inspired by MyBatis-Plus, providing convenient CRUD interfaces and query construction capabilities.

## Features

- **QueryWrapper**: Chainable query condition builder (Eq, Like, Gt, Lt, etc.)
- **Service**: Generic Service interface for standard CRUD operations
- **Page**: Built-in pagination support
- **Hooks**: Support for `BeforeCreate`, `AfterCreate` etc. via GORM hooks

## Installation

```bash
go get github.com/yourusername/gomp
```

## Usage

### 1. Define Model

```go
type User struct {
    ID   int64  `gorm:"primaryKey"`
    Name string
    Age  int
}
```

### 2. Create Service

```go
type UserService struct {
    *gomp.ServiceImpl[User]
}

func NewUserService(db *gorm.DB) *UserService {
    return &UserService{
        ServiceImpl: gomp.NewServiceImpl[User](db),
    }
}
```

### 3. CRUD Operations

```go
// Query
w := gomp.NewQueryWrapper[User]()
w.Eq("name", "Tom").Gt("age", 18)
users, err := userService.List(ctx, w)

// Pagination
page := gomp.NewPage[User](1, 10)
result, err := userService.Page(ctx, page, w)

// Update
u := gomp.NewUpdateWrapper[User]()
u.Set("age", 20).Eq("name", "Tom")
err := userService.Update(ctx, u)
```

## Requirements

- Go 1.18+
- GORM v1.20+
