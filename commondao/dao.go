package commondao

import (
	"context"
	"sync"
	"sync/atomic"
)

type entry[dataModel any] struct {
	mu      sync.RWMutex
	loading bool
	ready   chan struct{}
	value   *dataModel
	epoch   int64
	dbErr   error
}

type Cache[dataModel any] struct {
	items []map[string]*entry[dataModel]
	mapMu []sync.RWMutex
	epoch atomic.Int64
}

func NewCache[dataModel any]() *Cache[dataModel] {
	newCache := &Cache[dataModel]{
		items: make([]map[string]*entry[dataModel], 64),
		mapMu: make([]sync.RWMutex, 64),
	}
	for i := range newCache.items {
		newCache.items[i] = make(map[string]*entry[dataModel])
	}
	return newCache
}

type DBContextInterface[Context any] interface {
	GetClient() *Context
}
type DataInterface[dataModel any] interface {
	SetInfo(...*dataModel)
}
type DataCacheInterface[dataModel any] interface {
	GetKey() string
	GetEmptyValue() *dataModel
	WhetherExist(...*dataModel) bool
}
type cacheOperation[dataModel any, dbContextModel any] interface {
	GetCache(context.Context, DBContextInterface[dbContextModel]) (*dataModel, bool, error)
	SetCache(context.Context, DBContextInterface[dbContextModel]) error
	DeleteCache(context.Context, DBContextInterface[dbContextModel]) error
	LogCache(error)
}

type dbOperation[dataModel any, dbContextModel any] interface {
	GetDB(context.Context, DBContextInterface[dbContextModel]) (*dataModel, bool, error)
	SetDB(context.Context, DBContextInterface[dbContextModel]) error
	DeleteDB(context.Context, DBContextInterface[dbContextModel]) error
	UpdateDB(context.Context, DBContextInterface[dbContextModel]) (bool, error)
}

type OperateWithDBAndCache[dataModel any, dbContextModel any] interface {
	DataCacheInterface[dataModel]
	DataInterface[dataModel]
	dbOperation[dataModel, dbContextModel]
	cacheOperation[dataModel, dbContextModel]
}
type OperateWithDB[dataModel any, dbContextModel any] interface {
	DataInterface[dataModel]
	dbOperation[dataModel, dbContextModel]
}

type Setter[dbContextModel any] interface {
	Set(context.Context, *dbContextModel) error
}

func Set[dbContextModel any](ctx context.Context, DBContext DBContextInterface[dbContextModel], Info Setter[dbContextModel]) error {
	return Info.Set(ctx, DBContext.GetClient())
}

type Getter[dataModel any, dbContextModel any] interface {
	Get(context.Context, *dbContextModel) (*dataModel, bool, error)
}

func Get[dataModel any, dbContextModel any, getter Getter[dataModel, dbContextModel]](ctx context.Context, DBContext DBContextInterface[dbContextModel], Info getter) (*dataModel, bool, error) {
	return Info.Get(ctx, DBContext.GetClient())
}

type Updater[dbContextModel any] interface {
	Update(context.Context, *dbContextModel) (bool, error)
}

func Update[dbContextModel any](ctx context.Context, DBContext DBContextInterface[dbContextModel], Info Updater[dbContextModel]) (bool, error) {
	return Info.Update(ctx, DBContext.GetClient())
}

type Deleter[dbContextModel any] interface {
	Delete(context.Context, *dbContextModel) error
}

func Delete[dbContextModel any](ctx context.Context, DBContext DBContextInterface[dbContextModel], Info Deleter[dbContextModel]) error {
	return Info.Delete(ctx, DBContext.GetClient())
}
