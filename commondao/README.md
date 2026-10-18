# commondao

一个 Go 语言 DAO 层基础框架，用于解耦 DAO 实现与 Service 调用。

Service 层只需统一调用 `Set` / `Get` / `Update` / `Delete` 四个函数，DAO 对象实现对应接口即可。
框架内置了「数据库 + 缓存」的常用组合，并提供本地 L1 缓存以防止缓存穿透。

---

## 目录

- [特性](#特性)
- [快速上手](#快速上手)
- [核心概念](#核心概念)
- [接口说明](#接口说明)
- [框架内置函数](#框架内置函数)
- [如何选择函数](#如何选择函数)
- [完整示例](#完整示例)
- [注意事项](#注意事项)

---

## 特性

- 统一入口：上层只调用 `Set` / `Get` / `Update` / `Delete`。
- 逻辑解耦：DAO 只实现接口，不关心调用方式。
- 内置缓存策略：支持「仅 DB」「DB + Cache」「DB + Cache + 本地 L1」三种模式。
- 防缓存穿透：L1 缓存 + `singleflight` 思想（同 key 并发只查一次 DB）。
- 泛型支持：`Cache[dataModel]` 与各接口均为泛型，类型安全。

---

## 快速上手

### 1. 定义 DBContext

```go
type DBContext struct {
    DB *sql.DB
}

func (c *DBContext) GetClient() *DBContext { return c }
```

### 2. 定义 DAO 对象

```go
type User struct {
    ID   int64
    Name string
}

type UserDAO struct {
    user *User
}

// DataInterface
func (d *UserDAO) SetInfo(u ...*User) {
    if len(u) > 0 {
        d.user = u[0]
    }
}

// DataCacheInterface
func (d *UserDAO) GetKey() string              { return "user:" + strconv.FormatInt(d.user.ID, 10) }
func (d *UserDAO) GetEmptyValue() *User        { return &User{} }
func (d *UserDAO) WhetherExist(u ...*User) bool { return u[0] != nil && u[0].ID != 0 }

// dbOperation
func (d *UserDAO) GetDB(ctx context.Context, c DBContextInterface[*DBContext]) (*User, bool, error) {
    // 查询数据库...
}
func (d *UserDAO) SetDB(ctx context.Context, c DBContextInterface[*DBContext]) error    { /* ... */ }
func (d *UserDAO) DeleteDB(ctx context.Context, c DBContextInterface[*DBContext]) error { /* ... */ }
func (d *UserDAO) UpdateDB(ctx context.Context, c DBContextInterface[*DBContext]) (bool, error) { /* ... */ }

// cacheOperation
func (d *UserDAO) LogCache(err error)                                                  { log.Println(err) }
func (d *UserDAO) GetCache(ctx context.Context, c DBContextInterface[*DBContext]) (*User, bool, error) { /* ... */ }
func (d *UserDAO) SetCache(ctx context.Context, c DBContextInterface[*DBContext]) error    { /* ... */ }
func (d *UserDAO) DeleteCache(ctx context.Context, c DBContextInterface[*DBContext]) error { /* ... */ }

// 对外接口
func (d *UserDAO) Get(ctx context.Context, c *DBContext) (*User, bool, error) {
    return commondao.GetWithCacheWithoutL1(ctx, c, d)
}
```

### 3. 上层调用

```go
dbCtx := &DBContext{DB: db}
dao := &UserDAO{user: &User{ID: 1}}

user, exist, err := commondao.Get(ctx, dbCtx, dao)
```

---

## 核心概念

### 三种调用模式

| 模式 | 说明 | 需要实现的接口 |
|------|------|----------------|
| **WithoutCache** | 只操作数据库 | `DataInterface` + `dbOperation` |
| **WithCacheWithoutL1** | 操作 DB + 远程缓存，无本地缓存 | 上述 + `DataCacheInterface` + `cacheOperation` |
| **WithCacheWithL1** | 操作 DB + 远程缓存 + 本地 L1 缓存 | 同上，且需传入 `*Cache[dataModel]` |

### 本地 L1 缓存如何防穿透

当多个 goroutine 同时请求同一个不存在的 key 时：

1. 第一个 goroutine 发现 L1 未命中，在 `Cache` 中插入占位 `entry`（`loading = true`）。
2. 其余 goroutine 命中该 `entry`，阻塞在 `entry.ready` channel 上。
3. 第一个 goroutine 查询 DB 并写回远程缓存，写入空值缓存（`GetEmptyValue`）。
4. `close(ready)` 唤醒所有等待者，共享同一结果。
5. 100ms 后自动删除 L1 条目（`deleteL1Cache`），避免本地缓存与远程缓存长期不一致。

---

## 接口说明

### 数据库连接

#### `DBContextInterface[Context any]`

```go
type DBContextInterface[Context any] interface {
    GetClient() *Context
}
```

`Context` 一般为包含连接池的结构体指针。

---

### 框架内部接口

#### `DataInterface[dataModel]`

| 方法 | 说明 |
|------|------|
| `SetInfo(...*dataModel)` | 将查询结果写回 DAO 对象，供上层读取 |

#### `DataCacheInterface[dataModel]`

| 方法 | 说明 |
|------|------|
| `GetKey() string` | 缓存 key |
| `GetEmptyValue() *dataModel` | 空值缓存对象（防穿透） |
| `WhetherExist(...*dataModel) bool` | 判断是否为空值缓存 |

#### `dbOperation[dataModel, dbContextModel]`

| 方法 | 说明 |
|------|------|
| `GetDB(ctx, DBContext) (*dataModel, bool, error)` | 查 DB，返回数据、是否存在、错误 |
| `SetDB(ctx, DBContext) error` | 写 DB |
| `DeleteDB(ctx, DBContext) error` | 删 DB |
| `UpdateDB(ctx, DBContext) (bool, error)` | 更新 DB，返回是否存在 |

> 批量查询时，可在内部手动 `SetInfo`，框架不会覆盖。

#### `cacheOperation[dataModel, dbContextModel]`

| 方法 | 说明 |
|------|------|
| `LogCache(error)` | 缓存层内部日志（不向上返回） |
| `GetCache(ctx, DBContext) (*dataModel, bool, error)` | 读缓存 |
| `SetCache(ctx, DBContext) error` | 写缓存 |
| `DeleteCache(ctx, DBContext) error` | 删缓存 |

---

### 对外接口

| 接口 | 方法 | 用途 |
|------|------|------|
| `Setter[dbContextModel]` | `Set(ctx, *dbContextModel) error` | `Set` 入口 |
| `Getter[dataModel, dbContextModel]` | `Get(ctx, *dbContextModel) (*dataModel, bool, error)` | `Get` 入口 |
| `Updater[dbContextModel]` | `Update(ctx, *dbContextModel) (bool, error)` | `Update` 入口 |
| `Deleter[dbContextModel]` | `Delete(ctx, *dbContextModel) error` | `Delete` 入口 |

---

## 框架内置函数

### Set

| 函数 | 说明 |
|------|------|
| `SetWithoutCache` | 只写 DB |
| `SetWithCache` | 写 DB 后再写缓存（事务由上层控制） |

### Get

| 函数 | 说明 |
|------|------|
| `GetWithoutCache` | 只查 DB |
| `GetWithCacheWithoutL1` | 先查缓存，未命中再查 DB 并回写缓存 |
| `GetWithCacheWithL1` | 在上述基础上增加本地 L1 缓存 + 并发合并 |

### Update

| 函数 | 说明 |
|------|------|
| `UpdateWithoutCache` | 只更新 DB |
| `UpdateWithCache` | 更新 DB 后删除缓存（Cache-Aside） |

### Delete

| 函数 | 说明 |
|------|------|
| `DeleteWithoutCache` | 只删 DB |
| `DeleteWithCache` | 删 DB 后再删缓存 |

---

## 如何选择函数

| 场景 | 推荐函数 |
|------|----------|
| 无缓存，简单 CRUD | `*WithoutCache` |
| 有 Redis 等远程缓存，无本地缓存 | `GetWithCacheWithoutL1` + 其它 `*WithCache` |
| 高并发热点 key，需防穿透 | `GetWithCacheWithL1` |
| 强一致要求 | `*WithoutCache` 或自行处理缓存失效 |

---

## 完整示例

### Get（带 L1 缓存）

```go
var userCache = commondao.NewCache[*User]()

func (d *UserDAO) Get(ctx context.Context, c *DBContext) (*User, bool, error) {
    // 注意：WithCacheWithL1 需要传入 *Cache
    ok, err := commondao.GetWithCacheWithL1(ctx, userCache, c, d)
    if err != nil {
        return nil, false, err
    }
    return d.user, ok, nil
}
```

### Set

```go
func (d *UserDAO) Set(ctx context.Context, c *DBContext) error {
    return commondao.SetWithCache(ctx, c, d)
}
```

### Update

```go
func (d *UserDAO) Update(ctx context.Context, c *DBContext) (bool, error) {
    return commondao.UpdateWithCache(ctx, c, d)
}
```

### Delete

```go
func (d *UserDAO) Delete(ctx context.Context, c *DBContext) error {
    return commondao.DeleteWithCache(ctx, c, d)
}
```

### 上层统一调用

```go
err := commondao.Set(ctx, dbCtx, dao)
user, exist, err := commondao.Get(ctx, dbCtx, dao)
exist, err := commondao.Update(ctx, dbCtx, dao)
err := commondao.Delete(ctx, dbCtx, dao)
```

---

## 注意事项

1. **事务**：`SetWithCache` 中的 DB 与缓存写入不在同一事务内，事务回滚需由上层控制。
2. **缓存一致性**：`UpdateWithCache` 采用「先更新 DB，再删除缓存」的 Cache-Aside 策略，极端并发下仍可能短暂不一致。
3. **空值缓存**：`GetEmptyValue()` 返回的对象需能被 `WhetherExist()` 正确识别，否则空值缓存会失效。
4. **L1 缓存生命周期**：默认 100ms 后自动过期（`deleteL1Cache`），如需调整可修改源码中的 `time.Sleep`。
5. **批量查询**：`GetDB` / `GetCache` 的返回值只能承载单个对象，框架无法感知批量结果。批量场景下需在 `GetDB` / `GetCache` 内部自行将数据写入 DAO 对象；此时 `SetInfo` 仅作表面实现即可，内部无需具体逻辑。
6. **泛型实例化**：`Cache` 需按 `dataModel` 类型单独创建，例如 `NewCache[*User]()`。
7. **`GetWithCacheWithL1` 签名**：与其它内置函数不同，它需要额外传入 `*Cache[dataModel]` 参数。