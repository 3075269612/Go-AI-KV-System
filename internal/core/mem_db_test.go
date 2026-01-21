package core

import (
	"Go-AI-KV-System/internal/config"
	"os"
	"testing"
	"time"
)

// 辅助函数：创建一个不带 AOF 的纯内存 DB 用于旧测试
func newTestMemDB() *MemDB {
	return NewMemDB(&config.Config{})
}

func TestMemDB_TTL(t *testing.T) {
	db := newTestMemDB()

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
	db := newTestMemDB()

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

// Day3 新增：持久化测试
// 模拟：启动 DB -> 写入 -> 关闭 -> 重新启动 -> 验证数据存在
func TestMemDB_Persistence(t *testing.T) {
	// 1. 准备测试文件
	tmpAof := "test_persistence.aof"

	// 确保测试开始前和结束后都清理干净
	_ = os.Remove(tmpAof)
	defer os.Remove(tmpAof)

	// 构造开启 AOF 的配置
	cfg := &config.Config{
		AOF: config.AOFConfig{
			Filename: tmpAof,
		},
	}

	// 2. 第一阶段：启动并写入
	{
		db1 := NewMemDB(cfg)
		db1.Set("name", "naato", 0)
		db1.Set("lang", "go", 0)
		
		// 模拟数据删除
		db1.Set("temp", "delete_me", 0)
		db1.Del("temp")

		// 关闭 DB （释放文件句柄，确保 buffer 刷盘）
		db1.Close()
	}

	// 3. 第二阶段：重启
	{
		db2 := NewMemDB(cfg)

		// 验证 name 是否还在
		val, ok := db2.Get("name")
		if !ok || val != "naato" {
			t.Fatalf("Persistence failed! Expected 'naato', got %v", val)
		}

		// 验证 temp 是否被删除
		_, ok = db2.Get("temp")
		if ok {
			t.Fatal("Replay failed! 'temp' should be deleted.")
		}

		db2.Close()
	}
}