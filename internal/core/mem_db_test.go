package core

import (
	"testing"
	"time"
)

func TestMemDB_TTL(t *testing.T) {
	db := NewMemDB()

	// 1. 测试基本的 Set/Get
	db.Set("key1", "value1", 2*time.Second)
	if v, ok := db.Get("key1"); !ok || v != "value1" {
		t.Fatalf("Expeceted value1, got %v", v)
	}

	// 2. 测试过期（惰性删除验证）
	time.Sleep(2100 * time.Millisecond)
	if _, ok := db.Get("key1"); ok {
		t.Fatal("Key should be expired (Lazy Delete failed)")
	}
}

func TestMemDB_GC(t *testing.T) {
	db := NewMemDB()

	db.StartGC(500 * time.Millisecond)

	db.Set("key_gc", "value_gc", 1*time.Second)

	if _, ok := db.Get("key_gc"); !ok {
		t.Fatal("Key should exist")
	}

	time.Sleep(1500 * time.Millisecond)

	db.mu.RLock()
	_, ok := db.data["key_gc"]
	db.mu.RUnlock()

	if ok {
		t.Fatal("Key should be removed by background GC")
	}
}
