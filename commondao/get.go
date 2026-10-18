package commondao

import (
	"context"
	"hash/fnv"
	"strconv"
	"time"
	"unsafe"
)

func getKeyID(rawKey any) int {
	if rawKey == nil {
		return 0
	}
	var key string
	switch rawKey.(type) {
	case string:
		key = rawKey.(string)
	case []byte:
		key = string(rawKey.([]byte))
	case int, int8, int16, int32:
		key = strconv.Itoa(rawKey.(int))
	case int64:
		key = strconv.FormatInt(rawKey.(int64), 10)
	case uint, uint8, uint16, uint32, uint64:
		key = strconv.FormatUint(rawKey.(uint64), 10)
	case float32, float64:
		key = strconv.FormatFloat(rawKey.(float64), 'f', -1, 64)
	case bool:
		if rawKey.(bool) {
			key = "true"
		} else {
			key = "false"
		}
	default:
		return 0
	}
	hash := fnv.New64a()
	_, _ = hash.Write(unsafe.Slice(unsafe.StringData(key), len(key)))
	result := int(hash.Sum64()) % 64
	if result < 0 {
		return -result
	}
	return result
}
func GetWithCacheWithL1[dataModel any, dbContextModel any, DM OperateWithDBAndCache[dataModel, dbContextModel]](ctx context.Context, c *Cache[dataModel], DBContext DBContextInterface[dbContextModel], info DM) (bool, error) {
	value, exist, err := info.GetCache(ctx, DBContext)
	if !exist || err != nil {
		if err != nil {
			info.LogCache(err)
		}
		hashKey := getKeyID(info.GetKey())
		c.mapMu[hashKey].RLock()
		k, ok := c.items[hashKey][info.GetKey()]
		c.mapMu[hashKey].RUnlock()
		if ok {
			k.mu.RLock()
			if !k.loading {
				defer k.mu.RUnlock()
				info.SetInfo(k.value)
				return info.WhetherExist(k.value), err
			} else {
				k.mu.RUnlock()
				select {
				case <-k.ready:
				case <-ctx.Done():
					return false, ctx.Err()
				}
				k.mu.RLock()
				defer k.mu.RUnlock()
				if k.value != nil {
					if info.WhetherExist(k.value) {
						info.SetInfo(k.value)
						return true, err
					} else {
						return false, err
					}
				} else {
					return false, k.dbErr
				}
			}
		} else {
			c.mapMu[hashKey].Lock()
			k, ok = c.items[hashKey][info.GetKey()]
			if ok && k != nil {
				c.mapMu[hashKey].Unlock()
				select {
				case <-k.ready:
				case <-ctx.Done():
					return false, ctx.Err()
				}
				k.mu.RLock()
				defer k.mu.RUnlock()
				if k.value != nil {
					if info.WhetherExist(k.value) {
						info.SetInfo(k.value)
						return true, err
					} else {
						return false, err
					}
				} else {
					return false, k.dbErr
				}
			}
			c.epoch.Add(1)
			epoch := c.epoch.Load()
			c.items[hashKey][info.GetKey()] = &entry[dataModel]{
				ready:   make(chan struct{}),
				loading: true,
				epoch:   epoch,
			}
			c.mapMu[hashKey].Unlock()
			c.mapMu[hashKey].RLock()
			k, _ = c.items[hashKey][info.GetKey()]
			c.mapMu[hashKey].RUnlock()
			var err2 error
			value, exist, err2 = info.GetDB(ctx, DBContext)
			k.mu.Lock()
			if err2 != nil {
				k.loading = false
				k.dbErr = err2
				k.mu.Unlock()
				c.mapMu[hashKey].Lock()
				delete(c.items[hashKey], info.GetKey())
				c.mapMu[hashKey].Unlock()
				close(k.ready)
				return false, err2
			}
			if exist {
				k.value = value
			} else {
				k.value = info.GetEmptyValue()
			}
			k.loading = false
			k.mu.Unlock()
			info.SetInfo(value)
			err2 = info.SetCache(ctx, DBContext)
			close(k.ready)
			go deleteL1Cache(c, info.GetKey(), hashKey, epoch)
			if err2 != nil {
				return exist, err2
			}
			return exist, err
		}
	}
	info.SetInfo(value)
	return info.WhetherExist(value), nil
}

func deleteL1Cache[dataModel any](c *Cache[dataModel], key string, hashKey int, epoch int64) {
	time.Sleep(100 * time.Millisecond)
	c.mapMu[hashKey].Lock()
	info, ok := c.items[hashKey][key]
	if ok && info != nil {
		if info.epoch == epoch {
			delete(c.items[hashKey], key)
		}
	}
	c.mapMu[hashKey].Unlock()
}
func GetWithCacheWithoutL1[dataModel any, dbContextModel any, operateWithDBAndCache OperateWithDBAndCache[dataModel, dbContextModel]](ctx context.Context, dbContext DBContextInterface[dbContextModel], info operateWithDBAndCache) (bool, error) {
	gotInfo, exist, err1 := info.GetCache(ctx, dbContext)
	if err1 == nil {
		if exist && info.WhetherExist(gotInfo) {
			info.SetInfo(gotInfo)
			return true, nil
		} else {
			return false, nil
		}
	}
	gotInfo, exist, err2 := info.GetDB(ctx, dbContext)
	if err2 != nil {
		return false, err2
	}
	if !exist {
		return false, nil
	}
	err1 = info.SetCache(ctx, dbContext)
	info.SetInfo(gotInfo)
	if err1 != nil {
		return true, err1
	}
	return true, nil
}
func GetWithoutCache[dataModel any, dbContextModel any, operateWithDB OperateWithDB[dataModel, dbContextModel]](ctx context.Context, DBContext DBContextInterface[dbContextModel], info operateWithDB) (bool, error) {
	data, exist, err := info.GetDB(ctx, DBContext)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, nil
	}
	info.SetInfo(data)
	return true, nil
}
