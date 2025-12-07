package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

	"github.com/gin-gonic/gin"
)

// mockCache 模拟缓存实现
type mockCache struct {
	data  map[string]string
	locks map[string]bool
}

func newMockCache() *mockCache {
	return &mockCache{
		data:  make(map[string]string),
		locks: make(map[string]bool),
	}
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	if v, ok := m.data[key]; ok {
		return v, nil
	}
	return "", nil
}

func (m *mockCache) Set(ctx context.Context, key string, value string, expiration *time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

func (m *mockCache) Incr(ctx context.Context, key string) (int64, error) {
	return 1, nil
}

func (m *mockCache) Decr(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

func (m *mockCache) Lock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	lockKey := "lock:" + key
	if m.locks[lockKey] {
		return false, nil
	}
	m.locks[lockKey] = true
	return true, nil
}

func (m *mockCache) Unlock(ctx context.Context, key string) error {
	lockKey := "lock:" + key
	delete(m.locks, lockKey)
	return nil
}

func (m *mockCache) SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	if _, ok := m.data[key]; ok {
		return false, nil
	}
	m.data[key] = value
	return true, nil
}

func (m *mockCache) Close() error {
	return nil
}

var _ cache.Cache = (*mockCache)(nil)

func setupRouter(ca cache.Cache, strict bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())

	// 设置全局缓存
	cache.GlobalCache = ca

	r.POST("/test", CreateIdempotencyMiddleware(ca, strict), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": "test data"})
	})

	return r
}

func TestIdempotencyMiddleware_WithKey_FirstRequest(t *testing.T) {
	mockCa := newMockCache()
	router := setupRouter(mockCa, false)

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test": "data"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(constant.IdempotencyKey, "test-key-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		return
	}
	if resp["message"] != "success" {
		t.Errorf("Expected message 'success', got %v", resp["message"])
	}
}

func TestIdempotencyMiddleware_WithKey_DuplicateRequest(t *testing.T) {
	mockCa := newMockCache()
	router := setupRouter(mockCa, false)

	// 第一次请求
	req1, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test": "data"}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(constant.IdempotencyKey, "duplicate-key-123")

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request: Expected status 200, got %d", w1.Code)
	}

	// 第二次请求（重复）
	req2, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test": "data"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(constant.IdempotencyKey, "duplicate-key-123")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request: Expected status 200, got %d", w2.Code)
	}

	// 检查是否标记为重放响应
	if w2.Header().Get("X-Idempotency-Replayed") != "true" {
		t.Error("Expected X-Idempotency-Replayed header to be 'true'")
	}
}

func TestIdempotencyMiddleware_WithoutKey_LooseMode(t *testing.T) {
	mockCa := newMockCache()
	router := setupRouter(mockCa, false) // 宽松模式

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test": "data"}`))
	req.Header.Set("Content-Type", "application/json")
	// 不设置 X-Idempotency-Key

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 宽松模式下应该正常处理请求
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestIdempotencyMiddleware_WithoutKey_StrictMode(t *testing.T) {
	mockCa := newMockCache()
	router := setupRouter(mockCa, true) // 严格模式

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test": "data"}`))
	req.Header.Set("Content-Type", "application/json")
	// 不设置 X-Idempotency-Key

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 严格模式下应该拒绝请求
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestIdempotencyMiddleware_GetRequest_SkipCheck(t *testing.T) {
	mockCa := newMockCache()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cache.GlobalCache = mockCa

	r.GET("/test", CreateIdempotencyMiddleware(mockCa, true), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	// GET请求不需要幂等性Key

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// GET请求应该跳过幂等性检查
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestIdempotencyMiddleware_DifferentKeys_BothSucceed(t *testing.T) {
	mockCa := newMockCache()
	router := setupRouter(mockCa, false)

	// 第一个请求
	req1, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test": "data1"}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(constant.IdempotencyKey, "key-1")

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request: Expected status 200, got %d", w1.Code)
	}

	// 第二个请求（不同的key）
	req2, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test": "data2"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(constant.IdempotencyKey, "key-2")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request: Expected status 200, got %d", w2.Code)
	}

	// 两个请求都不应该被标记为重放
	if w1.Header().Get("X-Idempotency-Replayed") == "true" {
		t.Error("First request should not be marked as replayed")
	}
	if w2.Header().Get("X-Idempotency-Replayed") == "true" {
		t.Error("Second request should not be marked as replayed")
	}
}

func TestIdempotencyMiddleware_NilCache_SkipCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())

	// 设置全局缓存为nil
	cache.GlobalCache = nil

	r.POST("/test", IdempotencyRecommended(nil), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(`{"test": "data"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(constant.IdempotencyKey, "test-key")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 缓存不可用时应该跳过检查，正常处理请求
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
