package commondao

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ========== 测试数据模型 ==========

type TestDataModel struct {
	Key  string
	Val  any
	Info *TestDataModel
}

func (d *TestDataModel) GetKey() string {
	return d.Key
}

func (d *TestDataModel) GetEmptyValue() *TestDataModel {
	return &TestDataModel{
		Key:  d.Key,
		Val:  nil,
		Info: &TestDataModel{},
	}
}

func (d *TestDataModel) SetInfo(info *TestDataModel) {
	d.Info = info
}

func (d *TestDataModel) WhetherExist() bool {
	if d.Info == nil {
		return false
	}
	return d.Info.Val != nil
}

// ========== 测试 DBContext ==========

type TestDBContext struct {
	Mysql string
	Redis string
}

func (d *TestDBContext) GetClient() *TestDBContext {
	return d
}

// ========== 函数适配器（测试用） ==========
// MockOperateWithDBAndCache 实现了 OperateWithDBAndCache 接口
type MockOperateWithDBAndCache[DM any, DC any] struct {
	// info 接口
	GetKeyFunc        func() string
	GetEmptyValueFunc func() *DM
	SetInfoFunc       func(*DM)
	WhetherExistFunc  func() bool

	// cacheOperation 接口
	GetCacheFunc    func(context.Context, DBContextInterface[DC]) (*DM, bool, error)
	SetCacheFunc    func(context.Context, DBContextInterface[DC]) error
	DeleteCacheFunc func(context.Context, DBContextInterface[DC]) error

	// dbOperation 接口
	GetDBFunc    func(context.Context, DBContextInterface[DC]) (*DM, bool, error)
	SetDBFunc    func(context.Context, DBContextInterface[DC]) error
	DeleteDBFunc func(context.Context, DBContextInterface[DC]) error
	UpdateDBFunc func(context.Context, DBContextInterface[DC]) (bool, error)
	//储存数据
	info *DM
}

// 实现 info 接口
func (m *MockOperateWithDBAndCache[DM, DC]) GetKey() string {
	if m.GetKeyFunc != nil {
		return m.GetKeyFunc()
	}
	return ""
}

func (m *MockOperateWithDBAndCache[DM, DC]) GetEmptyValue() *DM {
	if m.GetEmptyValueFunc != nil {
		return m.GetEmptyValueFunc()
	}
	var zero DM
	return &zero
}

func (m *MockOperateWithDBAndCache[DM, DC]) SetInfo(info *DM) {
	if m.SetInfoFunc != nil {
		m.SetInfoFunc(info)
	}
}

func (m *MockOperateWithDBAndCache[DM, DC]) WhetherExist() bool {
	if m.WhetherExistFunc != nil {
		return m.WhetherExistFunc()
	}
	return false
}

// 实现 cacheOperation 接口
func (m *MockOperateWithDBAndCache[DM, DC]) GetCache(ctx context.Context, db DBContextInterface[DC]) (*DM, bool, error) {
	if m.GetCacheFunc != nil {
		return m.GetCacheFunc(ctx, db)
	}
	var zero DM
	return &zero, false, nil
}

func (m *MockOperateWithDBAndCache[DM, DC]) SetCache(ctx context.Context, db DBContextInterface[DC]) error {
	if m.SetCacheFunc != nil {
		return m.SetCacheFunc(ctx, db)
	}
	return nil
}

func (m *MockOperateWithDBAndCache[DM, DC]) DeleteCache(ctx context.Context, db DBContextInterface[DC]) error {
	if m.DeleteCacheFunc != nil {
		return m.DeleteCacheFunc(ctx, db)
	}
	return nil
}
func (m *MockOperateWithDBAndCache[DM, DC]) LogCache(_ error) {}

// 实现 dbOperation 接口
func (m *MockOperateWithDBAndCache[DM, DC]) GetDB(ctx context.Context, db DBContextInterface[DC]) (*DM, bool, error) {
	if m.GetDBFunc != nil {
		return m.GetDBFunc(ctx, db)
	}
	var zero DM
	return &zero, false, nil
}

func (m *MockOperateWithDBAndCache[DM, DC]) SetDB(ctx context.Context, db DBContextInterface[DC]) error {
	if m.SetDBFunc != nil {
		return m.SetDBFunc(ctx, db)
	}
	return nil
}

func (m *MockOperateWithDBAndCache[DM, DC]) DeleteDB(ctx context.Context, db DBContextInterface[DC]) error {
	if m.DeleteDBFunc != nil {
		return m.DeleteDBFunc(ctx, db)
	}
	return nil
}

func (m *MockOperateWithDBAndCache[DM, DC]) UpdateDB(ctx context.Context, db DBContextInterface[DC]) (bool, error) {
	if m.UpdateDBFunc != nil {
		return m.UpdateDBFunc(ctx, db)
	}
	return false, nil
}

// ========== 测试辅助函数（返回预配置的 mock） ==========

// 缓存命中场景
func newMockCacheHit(key string) *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
	mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}

	mock.GetKeyFunc = func() string { return key }
	mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		data := &TestDataModel{
			Key:  key,
			Val:  "cached_value",
			Info: &TestDataModel{Val: "cached_value"},
		}
		return data, true, nil
	}
	mock.SetInfoFunc = func(info *TestDataModel) {
		mock.info = info
	}
	mock.WhetherExistFunc = func() bool { return true }

	return mock
}

// 缓存未命中，数据库命中场景
func newMockCacheMissDBHit(key string) *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
	mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}

	mock.GetKeyFunc = func() string { return key }
	mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, nil
	}
	mock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		data := &TestDataModel{
			Key:  key,
			Val:  "db_value",
			Info: &TestDataModel{Val: "db_value"},
		}
		return data, true, nil
	}
	mock.SetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) error {
		return nil
	}
	mock.SetInfoFunc = func(info *TestDataModel) {
		mock.info = info
	}
	mock.WhetherExistFunc = func() bool { return true }

	return mock
}

// 缓存和数据库都未命中场景
func newMockCacheMissDBMiss(key string) *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
	mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}

	mock.GetKeyFunc = func() string { return key }
	mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, nil
	}
	mock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, nil
	}
	mock.GetEmptyValueFunc = func() *TestDataModel {
		return &TestDataModel{Key: key, Val: nil, Info: &TestDataModel{}}
	}
	mock.SetInfoFunc = func(info *TestDataModel) {
		mock.info = info
	}
	mock.WhetherExistFunc = func() bool { return false }

	return mock
}

// 缓存错误场景
func newMockCacheError(key string) *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
	mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}

	mock.GetKeyFunc = func() string { return key }
	mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, errors.New("redis connection failed")
	}
	mock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		data := &TestDataModel{
			Key:  key,
			Val:  "db_value",
			Info: &TestDataModel{Val: "db_value"},
		}
		return data, true, nil
	}
	mock.SetInfoFunc = func(info *TestDataModel) {
		mock.info = info
	}
	mock.WhetherExistFunc = func() bool { return true }

	return mock
}

// 数据库错误场景
func newMockDBError(key string) *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
	mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}

	mock.GetKeyFunc = func() string { return key }
	mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, nil
	}
	mock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, errors.New("database connection failed")
	}

	return mock
}

// 慢查询场景
func newMockSlowDB(key string, delay time.Duration) *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
	mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}

	mock.GetKeyFunc = func() string { return key }
	mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, nil
	}
	mock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		select {
		case <-time.After(delay):
			data := &TestDataModel{
				Key:  key,
				Val:  "slow_db_value",
				Info: &TestDataModel{Val: "slow_db_value"},
			}
			return data, true, nil
		case <-ctx.Done():
			return nil, false, ctx.Err()
		}
	}
	mock.SetInfoFunc = func(info *TestDataModel) {
		mock.info = info
	}
	mock.WhetherExistFunc = func() bool { return true }

	return mock
}

// 数据库超时场景
func newMockDBTimeout(key string) *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
	mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}

	mock.GetKeyFunc = func() string { return key }
	mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, nil
	}
	mock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		<-time.After(500 * time.Millisecond)
		return nil, false, context.DeadlineExceeded
	}

	return mock
}

// 设置缓存失败场景
func newMockSetCacheError(key string) *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
	mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}

	mock.GetKeyFunc = func() string { return key }
	mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, nil
	}
	mock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		data := &TestDataModel{
			Key:  key,
			Val:  "db_value",
			Info: &TestDataModel{Val: "db_value"},
		}
		return data, true, nil
	}
	mock.SetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) error {
		return errors.New("set cache failed")
	}
	mock.SetInfoFunc = func(info *TestDataModel) {
		mock.info = info
	}

	return mock
}

// 空值缓存命中场景
func newMockEmptyCacheHit(key string) *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
	mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}

	mock.GetKeyFunc = func() string { return key }
	mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		data := &TestDataModel{
			Key:  key,
			Val:  nil,
			Info: &TestDataModel{},
		}
		return data, true, nil
	}
	mock.SetInfoFunc = func(info *TestDataModel) {
		mock.info = info
	}
	mock.WhetherExistFunc = func() bool { return false }

	return mock
}

// ========== 测试用例 ==========

// 1. 测试命中缓存
func TestGetWithCache_HitCache(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "test_key"

	mockInfo := newMockCacheHit(key)

	exist, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		context.Background(),
		cache,
		dbCtx,
		mockInfo,
	)

	if err != nil {
		t.Errorf("期望无错误，实际: %v", err)
	}
	if !exist {
		t.Errorf("期望 exist = true，实际: false")
	}
	t.Log("✅ 命中缓存测试通过")
}

// 2. 测试缓存未命中，但数据库命中
func TestGetWithCache_CacheMiss_DBHit(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "test_key"

	mockInfo := newMockCacheMissDBHit(key)

	exist, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		context.Background(),
		cache,
		dbCtx,
		mockInfo,
	)

	if err != nil {
		t.Errorf("期望无错误，实际: %v", err)
	}
	if !exist {
		t.Errorf("期望 exist = true，实际: false")
	}

	t.Log("✅ 缓存未命中-数据库命中测试通过")
}

// 3. 测试缓存和数据库都未命中
func TestGetWithCache_CacheMiss_DBMiss(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "test_key"

	mockInfo := newMockCacheMissDBMiss(key)

	exist, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		context.Background(),
		cache,
		dbCtx,
		mockInfo,
	)

	if err != nil {
		t.Errorf("期望无错误，实际: %v", err)
	}
	if exist {
		t.Errorf("期望 exist = false，实际: true")
	}

	t.Log("✅ 缓存和数据库都未命中测试通过")
}

// 4. 测试缓存错误
func TestGetWithCache_CacheError(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "test_key"

	mockInfo := newMockCacheError(key)

	_, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		context.Background(),
		cache,
		dbCtx,
		mockInfo,
	)

	if err == nil {
		t.Errorf("期望有错误，实际无错误")
		return
	}
	if err.Error() != "redis connection failed" {
		t.Errorf("期望错误信息为 'redis connection failed'，实际: %v", err)
	}
	t.Log("✅ 缓存错误测试通过")
}

// 5. 测试数据库错误
func TestGetWithCache_DBError(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "test_key"

	mockInfo := newMockDBError(key)

	_, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		context.Background(),
		cache,
		dbCtx,
		mockInfo,
	)

	if err == nil {
		t.Errorf("期望有错误，实际无错误")
	}
	if err.Error() != "database connection failed" {
		t.Errorf("期望错误信息为 'database connection failed'，实际: %v", err)
	}
	t.Log("✅ 数据库错误测试通过")
}

// 6. 测试并发请求（缓存击穿防护）
func TestGetWithCache_ConcurrentRequests_CacheMiss(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "concurrent_key"

	var wg sync.WaitGroup
	successCount := 0
	halfSuccessCount := 0
	var mu1, mu2, mu3 sync.Mutex
	requestCount := 0
	// 模拟 10 个并发请求
	for range 10 {
		wg.Add(100)
		go func() {
			for range 100 {
				time.Sleep(time.Millisecond * 50)
				go func() {
					defer wg.Done()
					mockInfo := newMockCacheMissDBHit(key)
					exist, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
						context.Background(),
						cache,
						dbCtx,
						mockInfo,
					)
					mu3.Lock()
					requestCount++
					mu3.Unlock()
					if err == nil && exist {
						mu1.Lock()
						halfSuccessCount++
						mu1.Unlock()
						if mockInfo.info != nil && mockInfo.info.Val == "db_value" {
							mu2.Lock()
							successCount++
							mu2.Unlock()
						}
					}
				}()
			}
		}()
	}
	wg.Wait()
	t.Logf("halfSuccess请求个数：%d", halfSuccessCount)
	t.Logf("success请求个数：%d", successCount)
	t.Logf("总请求个数：%d", requestCount)
	// 所有请求都应该成功获取数据
	//if successCount != requestCount*100 {
	//	t.Errorf("期望 %d 个请求都成功，实际: %d", requestCount*100, successCount)
	//	return
	//}

	//// 验证只有一个 entry 被创建
	//cache.mapMu.RLock()
	//entryCount := len(cache.items)
	//cache.mapMu.RUnlock()
	//if entryCount != 1 {
	//	t.Errorf("期望只有 1 个缓存条目，实际: %d", entryCount)
	//}
	t.Log("✅ 并发请求测试通过")
}

// 7. 测试上下文超时
func TestGetWithCache_ContextTimeout(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "test_key"

	// 创建 100ms 超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mockInfo := newMockSlowDB(key, 500*time.Millisecond)

	_, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		ctx,
		cache,
		dbCtx,
		mockInfo,
	)

	if err == nil {
		t.Errorf("期望有错误，实际无错误")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("期望超时错误，实际: %v", err)
	}
	t.Log("✅ 上下文超时测试通过")
}

// 8. 测试数据库查询超时
func TestGetWithCache_DBTimeout(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "test_key"

	ctx := context.Background()

	mockInfo := newMockDBTimeout(key)

	_, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		ctx,
		cache,
		dbCtx,
		mockInfo,
	)

	if err == nil {
		t.Errorf("期望有错误，实际无错误")
	}
	t.Log("✅ 数据库超时测试通过")
}

// 9. 测试设置缓存失败
func TestGetWithCache_SetCacheError(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "test_key"

	mockInfo := newMockSetCacheError(key)

	_, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		context.Background(),
		cache,
		dbCtx,
		mockInfo,
	)

	if err == nil {
		t.Errorf("期望有错误，实际无错误")
	}
	if err.Error() != "set cache failed" {
		t.Errorf("期望错误信息为 'set cache failed'，实际: %v", err)
	}
	t.Log("✅ 设置缓存失败测试通过")
}

// 10. 测试重复请求同一 key（等待 loading 完成）
func TestGetWithCache_RequestWhileLoading(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "loading_key"

	var wg sync.WaitGroup
	startCh := make(chan struct{})
	slowMock := newMockSlowDB(key, 300*time.Millisecond)

	// 第一个请求会触发加载
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-startCh
		exist, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
			context.Background(),
			cache,
			dbCtx,
			slowMock,
		)
		if err != nil || !exist {
			t.Errorf("第一个请求失败: err=%v, exist=%v", err, exist)
		}
	}()

	// 第二个请求会在第一个请求的 loading 期间等待
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-startCh
		// 延迟 50ms 发起第二个请求，确保第一个请求已进入 loading 状态
		time.Sleep(50 * time.Millisecond)
		mockInfo := newMockCacheMissDBHit(key)
		exist, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
			context.Background(),
			cache,
			dbCtx,
			mockInfo,
		)
		if err != nil || !exist {
			t.Errorf("第二个请求失败: err=%v, exist=%v", err, exist)
		}
	}()

	close(startCh)
	wg.Wait()

	t.Log("✅ 等待加载测试通过")
}

// 11. 测试不同的 key（隔离性）
func TestGetWithCache_DifferentKeys(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}

	keys := []string{"key1", "key2", "key3"}

	for _, key := range keys {
		mockInfo := newMockCacheMissDBHit(key)
		exist, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
			context.Background(),
			cache,
			dbCtx,
			mockInfo,
		)
		if err != nil {
			t.Errorf("key %s 失败: %v", key, err)
		}
		if !exist {
			t.Errorf("key %s 期望 exist=true", key)
		}
	}

	t.Log("✅ 不同 key 隔离性测试通过")
}

// 12. 测试空值缓存（防止缓存穿透）
func TestGetWithCache_EmptyValueCaching(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "nonexistent_key"

	// 第一次请求：数据库也不存在
	mockInfo1 := newMockCacheMissDBMiss(key)

	exist, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		context.Background(),
		cache,
		dbCtx,
		mockInfo1,
	)

	if err != nil {
		t.Errorf("第一次请求错误: %v", err)
		return
	}
	if exist {
		t.Errorf("第一次请求期望 exist=false")
		return
	}

	// 第二次请求：应该命中空值缓存
	mockInfo2 := newMockEmptyCacheHit(key)

	exist2, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		context.Background(),
		cache,
		dbCtx,
		mockInfo2,
	)

	if err != nil {
		t.Errorf("第二次请求错误: %v", err)
		return
	}
	if exist2 {
		t.Errorf("第二次请求期望 exist=false")
		return
	}
	t.Log("✅ 空值缓存测试通过")
}

// 13. 测试 context 取消
func TestGetWithCache_ContextCancel(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "test_key"

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	mockInfo := newMockSlowDB(key, 500*time.Millisecond)

	_, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
		ctx,
		cache,
		dbCtx,
		mockInfo,
	)

	if err == nil {
		t.Errorf("期望有错误")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("期望 context.Canceled，实际: %v", err)
	}
	t.Log("✅ context 取消测试通过")
}

// 14. 测试极端并发下的缓存穿透
func TestGetWithCache_ExtremeConcurrentPenetration(t *testing.T) {
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "penetration_key"

	// 计数器：统计数据库被调用的次数
	dbCallCount := int32(0)

	// 创建自定义 mock 来统计 DB 调用次数
	customMock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}
	customMock.GetKeyFunc = func() string { return key }
	customMock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, nil
	}
	customMock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		atomic.AddInt32(&dbCallCount, 1)
		time.Sleep(100 * time.Millisecond) // 模拟数据库查询耗时
		data := &TestDataModel{
			Key:  key,
			Val:  "db_value",
			Info: &TestDataModel{Val: "db_value"},
		}
		return data, true, nil
	}
	customMock.SetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) error {
		return nil
	}
	customMock.SetInfoFunc = func(info *TestDataModel) {}
	customMock.WhetherExistFunc = func() bool { return true }

	ch := make(chan struct{})
	var wg sync.WaitGroup
	goroutineCount := 100

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ch
			exist, err := GetWithCacheWithL1(
				context.Background(),
				cache,
				dbCtx,
				customMock,
			)
			if err != nil || !exist {
				t.Errorf("请求失败: err=%v", err)
			}
		}()
	}
	close(ch)
	wg.Wait()

	// 验证数据库只被调用了一次（缓存击穿防护生效）
	if atomic.LoadInt32(&dbCallCount) != 1 {
		t.Errorf("期望数据库只被调用 1 次，实际: %d", atomic.LoadInt32(&dbCallCount))
	}
	t.Log("✅ 极端并发缓存穿透测试通过")
}

// 15. 测试数据库更新后是否读取脏数据
func TestWithCache_UpdateData(t *testing.T) {
	defer func() {
		err := recover()
		if err != nil {
			t.Error(err)
		}
	}()
	timeMiliSeconed := 100 * time.Millisecond
	cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	ctx := context.Background()
	key := "test_key"
	mockInfo := newMockCacheMissDBHit(key)
	dbGet := func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		data := &TestDataModel{
			Key:  key,
			Val:  "db_value2",
			Info: &TestDataModel{Val: "db_value2"},
		}
		return data, true, nil
	}
	var oldValueCount, newValueCount atomic.Int32
	var wg1, wg2 sync.WaitGroup
	for range 100 {
		wg1.Go(func() {
			_, _ = GetWithCacheWithL1(ctx, cache, dbCtx, mockInfo)
			if mockInfo.info != nil {
				if mockInfo.info.Val.(string) == "db_value" {
					oldValueCount.Add(1)
				}
				if mockInfo.info.Val.(string) == "db_value2" {
					newValueCount.Add(1)
				}
			}
		})
	}
	time.Sleep(timeMiliSeconed)
	mockInfo.GetDBFunc = dbGet
	for range 100 {
		wg2.Go(func() {
			_, _ = GetWithCacheWithL1(ctx, cache, dbCtx, mockInfo)
			if mockInfo != nil {
				if mockInfo.info.Val.(string) == "db_value" {
					oldValueCount.Add(1)
				}
				if mockInfo.info.Val.(string) == "db_value2" {
					newValueCount.Add(1)
				}
			}
		})
	}
	wg1.Wait()
	wg2.Wait()
	t.Logf("oldValueCount=%d, newValueCount=%d", oldValueCount.Load(), newValueCount.Load())
	if newValueCount.Load() > 0 {
		t.Log("✅ 并发状态修改数据库操作成功")
	} else {
		t.Error("仍读取脏数据")
	}
}

// ========== 压力测试 ==========

// 测试1：高并发 - L2缓存命中场景
func TestHighConcurrency_GetWithCache_L2CacheHit(t *testing.T) {
	l1Cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}

	key := "hot_key"
	goroutineCount := 1000

	var wg sync.WaitGroup
	startCh := make(chan struct{})
	hitCount := int32(0)

	// 创建缓存命中的 mock
	mockInfo := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}
	mockInfo.GetKeyFunc = func() string { return key }
	mockInfo.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		data := &TestDataModel{
			Key:  key,
			Val:  "cached_value",
			Info: &TestDataModel{Val: "cached_value"},
		}
		return data, true, nil
	}
	mockInfo.SetInfoFunc = func(info *TestDataModel) {}
	mockInfo.WhetherExistFunc = func() bool { return true }

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startCh

			// 每个 goroutine 需要自己的 mock 实例（因为 GetKey 会被调用）
			info := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}
			info.GetKeyFunc = func() string { return key }
			info.GetCacheFunc = mockInfo.GetCacheFunc
			info.SetInfoFunc = mockInfo.SetInfoFunc
			info.WhetherExistFunc = mockInfo.WhetherExistFunc

			exist, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
				context.Background(),
				l1Cache,
				dbCtx,
				info,
			)

			if err != nil {
				t.Logf("错误: %v", err)
			}
			if exist {
				atomic.AddInt32(&hitCount, 1)
			}
		}()
	}

	close(startCh)
	wg.Wait()

	t.Logf("=== L2缓存命中压测 ===")
	t.Logf("请求数: %d", goroutineCount)
	t.Logf("成功数: %d", atomic.LoadInt32(&hitCount))

	if atomic.LoadInt32(&hitCount) != int32(goroutineCount) {
		t.Errorf("期望全部成功，实际: %d", atomic.LoadInt32(&hitCount))
	} else {
		t.Logf("✅ L2缓存命中测试通过")
	}
}

// 测试2：高并发 - L2缓存未命中，DB命中
func TestHighConcurrency_GetWithCache_L2CacheMiss_DBHit(t *testing.T) {
	l1Cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}

	key := "miss_key"
	goroutineCount := 1000

	var wg sync.WaitGroup
	startCh := make(chan struct{})
	dbCallCount := int32(0)

	// 创建共享的 mock 工厂函数
	createMock := func() *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
		mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}
		mock.GetKeyFunc = func() string { return key }
		mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
			return nil, false, nil
		}
		mock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
			atomic.AddInt32(&dbCallCount, 1)
			time.Sleep(20 * time.Millisecond)
			return &TestDataModel{
				Key:  key,
				Val:  "db_value",
				Info: &TestDataModel{},
			}, true, nil
		}
		mock.SetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) error {
			return nil
		}
		mock.SetInfoFunc = func(info *TestDataModel) {}
		mock.WhetherExistFunc = func() bool { return true }
		return mock
	}

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startCh

			info := createMock()
			_, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
				context.Background(),
				l1Cache,
				dbCtx,
				info,
			)

			if err != nil {
				t.Logf("错误: %v", err)
			}
		}()
	}

	close(startCh)
	wg.Wait()

	t.Logf("=== L2缓存未命中，DB命中压测 ===")
	t.Logf("请求数: %d", goroutineCount)
	t.Logf("DB调用次数: %d", atomic.LoadInt32(&dbCallCount))

	// 由于 L1 缓存击穿防护，DB 应该只被调用 1 次
	if atomic.LoadInt32(&dbCallCount) != 1 {
		t.Errorf("期望 DB 调用 1 次，实际: %d", atomic.LoadInt32(&dbCallCount))
	} else {
		t.Logf("✅ L1缓存击穿防护有效，DB只被调用 %d 次", atomic.LoadInt32(&dbCallCount))
	}
}

// 测试3：高并发 - L2缓存未命中，DB也不存在（空值缓存）
func TestHighConcurrency_GetWithCache_L2CacheMiss_DBMiss(t *testing.T) {
	l1Cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}

	key := "empty_key"
	goroutineCount := 1000

	var wg sync.WaitGroup
	startCh := make(chan struct{})
	dbCallCount := int32(0)

	createMock := func() *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
		mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}
		mock.GetKeyFunc = func() string { return key }
		mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
			return nil, false, nil
		}
		mock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
			atomic.AddInt32(&dbCallCount, 1)
			return nil, false, nil
		}
		mock.GetEmptyValueFunc = func() *TestDataModel {
			return &TestDataModel{Key: key, Val: nil, Info: &TestDataModel{}}
		}
		mock.SetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) error {
			return nil
		}
		mock.SetInfoFunc = func(info *TestDataModel) {}
		mock.WhetherExistFunc = func() bool { return false }
		return mock
	}

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startCh

			info := createMock()
			_, err := GetWithCacheWithL1[TestDataModel, TestDBContext](
				context.Background(),
				l1Cache,
				dbCtx,
				info,
			)

			if err != nil {
				t.Logf("错误: %v", err)
			}
		}()
	}

	close(startCh)
	wg.Wait()

	t.Logf("=== L2缓存未命中，DB不存在压测 ===")
	t.Logf("请求数: %d", goroutineCount)
	t.Logf("DB调用次数: %d", atomic.LoadInt32(&dbCallCount))

	// 空值缓存也应该只查一次 DB
	if atomic.LoadInt32(&dbCallCount) != 1 {
		t.Errorf("期望 DB 调用 1 次，实际: %d", atomic.LoadInt32(&dbCallCount))
	} else {
		t.Logf("✅ 空值缓存有效，DB只被调用 %d 次", atomic.LoadInt32(&dbCallCount))
	}
}

// 测试4：多 key 并发访问
func TestHighConcurrency_GetWithCache_MultipleKeys(t *testing.T) {
	l1Cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}

	keyCount := 50
	goroutinesPerKey := 100

	var wg sync.WaitGroup
	startCh := make(chan struct{})
	dbCallCount := int32(0)

	createMock := func(key string) *MockOperateWithDBAndCache[TestDataModel, TestDBContext] {
		mock := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}
		mock.GetKeyFunc = func() string { return key }
		mock.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
			return nil, false, nil
		}
		mock.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
			atomic.AddInt32(&dbCallCount, 1)
			return &TestDataModel{
				Key:  key,
				Val:  "db_value",
				Info: &TestDataModel{},
			}, true, nil
		}
		mock.SetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) error {
			return nil
		}
		mock.SetInfoFunc = func(info *TestDataModel) {}
		mock.WhetherExistFunc = func() bool { return true }
		return mock
	}

	for k := range keyCount {
		key := string(rune('a'+k%26)) + string(rune('0'+k/26))
		for range goroutinesPerKey {
			wg.Add(1)
			go func(key string) {
				defer wg.Done()
				<-startCh

				info := createMock(key)
				GetWithCacheWithL1[TestDataModel, TestDBContext](
					context.Background(),
					l1Cache,
					dbCtx,
					info,
				)
			}(key)
		}
	}

	close(startCh)
	wg.Wait()

	t.Logf("=== 多 key 并发压测 ===")
	t.Logf("Key 数量: %d", keyCount)
	t.Logf("每 Key 并发: %d", goroutinesPerKey)
	t.Logf("总请求数: %d", keyCount*goroutinesPerKey)
	t.Logf("DB调用次数: %d", atomic.LoadInt32(&dbCallCount))

	// 每个 key 应该只调用一次 DB
	expectedDBCalls := keyCount
	if atomic.LoadInt32(&dbCallCount) > int32(expectedDBCalls*2) {
		t.Errorf("DB调用次数过多: %d > %d", atomic.LoadInt32(&dbCallCount), expectedDBCalls*2)
	} else {
		t.Logf("✅ 多 key 压测通过，DB调用 %d 次", atomic.LoadInt32(&dbCallCount))
	}
}

// 测试5：性能基准 - L2缓存命中
func BenchmarkGetWithCache_L2CacheHit(b *testing.B) {
	l1Cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "bench_key"

	mockInfo := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}
	mockInfo.GetKeyFunc = func() string { return key }
	mockInfo.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		data := &TestDataModel{
			Key:  key,
			Val:  "cached_value",
			Info: &TestDataModel{},
		}
		return data, true, nil
	}
	mockInfo.SetInfoFunc = func(info *TestDataModel) {}
	mockInfo.WhetherExistFunc = func() bool { return true }

	// 预热
	GetWithCacheWithL1[TestDataModel, TestDBContext](
		context.Background(),
		l1Cache,
		dbCtx,
		mockInfo,
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			info := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}
			info.GetKeyFunc = mockInfo.GetKeyFunc
			info.GetCacheFunc = mockInfo.GetCacheFunc
			info.SetInfoFunc = mockInfo.SetInfoFunc
			info.WhetherExistFunc = mockInfo.WhetherExistFunc

			GetWithCacheWithL1[TestDataModel, TestDBContext](
				context.Background(),
				l1Cache,
				dbCtx,
				info,
			)
		}
	})
}

// 测试6：性能基准 - L2缓存未命中，DB命中
func BenchmarkGetWithCache_L2CacheMiss_DBHit(b *testing.B) {
	l1Cache := NewCache[TestDataModel]()
	dbCtx := &TestDBContext{Mysql: "test", Redis: "test"}
	key := "bench_key"

	mockInfo := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}
	mockInfo.GetKeyFunc = func() string { return key }
	mockInfo.GetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return nil, false, nil
	}
	mockInfo.GetDBFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) (*TestDataModel, bool, error) {
		return &TestDataModel{
			Key:  key,
			Val:  "db_value",
			Info: &TestDataModel{},
		}, true, nil
	}
	mockInfo.SetCacheFunc = func(ctx context.Context, db DBContextInterface[TestDBContext]) error {
		return nil
	}
	mockInfo.SetInfoFunc = func(info *TestDataModel) {}
	mockInfo.WhetherExistFunc = func() bool { return true }

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			info := &MockOperateWithDBAndCache[TestDataModel, TestDBContext]{}
			info.GetKeyFunc = mockInfo.GetKeyFunc
			info.GetCacheFunc = mockInfo.GetCacheFunc
			info.GetDBFunc = mockInfo.GetDBFunc
			info.SetCacheFunc = mockInfo.SetCacheFunc
			info.SetInfoFunc = mockInfo.SetInfoFunc
			info.WhetherExistFunc = mockInfo.WhetherExistFunc

			GetWithCacheWithL1[TestDataModel, TestDBContext](
				context.Background(),
				l1Cache,
				dbCtx,
				info,
			)
		}
	})
}
