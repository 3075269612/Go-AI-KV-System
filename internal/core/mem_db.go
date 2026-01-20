package core

import (
	"sync"
	"time"
)

// Item 封装了值和过期时间
type Item struct {
	Val      any
	ExpireAt int64
}

// MemDB 内存数据库核心结构
type MemDB struct {
	mu   sync.RWMutex
	data map[string]*Item
}

func NewMemDB() *MemDB {
	return &MemDB{
		data: make(map[string]*Item),
	}
}

// Set 写入数据，支持过期时间(ttl: time to live)
// ttl = 0 表示永不过期
func (db *MemDB) Set(key string, val any, ttl time.Duration) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var expireAt int64 = 0
	if ttl > 0 {
		expireAt = time.Now().Add(ttl).UnixNano()
	}

	db.data[key] = &Item{val, expireAt}
}

// Get 获取数据（实现惰性删除）
func (db *MemDB) Get(key string) (any, bool) {
	db.mu.RLock()
	item, ok := db.data[key]
	db.mu.RUnlock() // 读锁先释放，因为下面可能需要加写锁进行删除

	if !ok {
		return nil, false
	}

	// 检查是否过期
	if item.ExpireAt > 0 && time.Now().UnixNano() > item.ExpireAt {
		// 发现过期，惰性删除
		db.mu.Lock()
		defer db.mu.Unlock()

		// Double Check双重检查，防止加锁间隙被其他协程处理
		newItem, exists := db.data[key]
		if !exists {
			// 已经被别人删了
			return nil, false
		}

		// 依然存在，且依然是过期状态，真删
		if newItem.ExpireAt > 0 && time.Now().UnixNano() > newItem.ExpireAt {
			delete(db.data, key)
			return nil, false
		}

		// 第一次看过期，第二次看续命
		return newItem.Val, true
	}

	return item.Val, true
}

// Del 手动删除数据
func (db *MemDB) Del(key string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	delete(db.data, key)
}

// StartGC 启动定期清理（Garbage Collection）
// interval: 清理间隔，例如1秒
func (db *MemDB) StartGC(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		for range ticker.C {
			db.activeCleanup()
		}
	}()
}

// activeCleanup 遍历 map 清理过期数据
func (db *MemDB) activeCleanup() {
	now := time.Now().UnixNano()
	var expireKeys []string

	db.mu.RLock()
	for key, item := range db.data {
		if item.ExpireAt > 0 && now > item.ExpireAt {
			expireKeys = append(expireKeys, key)
		}
	}
	db.mu.RUnlock()

	if len(expireKeys) > 0 {
		db.mu.Lock()
		defer db.mu.Unlock()

		for _, key := range expireKeys {
			// Double Check
			item, exists := db.data[key]
			if exists && item.ExpireAt > 0 && time.Now().UnixNano() > item.ExpireAt {
				delete(db.data, key)
			}
		}
	}
}
