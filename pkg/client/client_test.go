package client

import (
	"testing"
	"time"

	// 👇 引入我们需要的基础模块
	"Go-AI-KV-System/internal/config"
	"Go-AI-KV-System/internal/core"
	"Go-AI-KV-System/pkg/protocol"
)

func TestClient_Integration(t *testing.T) {
	// ==========================================
	// 1. 模拟启动服务端 (Server Setup)
	// ==========================================
	
	// 使用 localhost 的一个不常用端口，防止冲突
	addr := "localhost:9999"

	// 初始化内存数据库 (这是 Server 需要的依赖)
	memDB := core.NewMemDB(&config.Config{})
	
	// 初始化服务端
	server := protocol.NewServer(addr, memDB)

	// ⚠️ 关键点：在一个新的 Goroutine 中启动 Server
	// 如果不加 'go'，代码会卡在这里死循环，永远不会执行下面的 Client 逻辑
	go func() {
		if err := server.Start(); err != nil {
			t.Errorf("Server failed to start: %v", err)
		}
	}()

	// 稍微睡 100 毫秒，确保 Server 已经准备好监听了
	time.Sleep(100 * time.Millisecond)

	// ==========================================
	// 2. 启动客户端 (Client Action)
	// ==========================================
	
	// 连接刚刚启动的本地服务端
	cli, err := NewClient(addr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer cli.Close()

	// ==========================================
	// 3. 执行测试用例 (Assert)
	// ==========================================

	// 测试 SET
	key := "my_name"
	val := "Naato"
	t.Logf("Testing SET %s = %s", key, val)
	
	err = cli.Set(key, val)
	if err != nil {
		t.Fatalf("❌ SET command failed: %v", err)
	}

	// 测试 GET
	t.Logf("Testing GET %s", key)
	got, err := cli.Get(key)
	if err != nil {
		t.Fatalf("❌ GET command failed: %v", err)
	}

	// 验证结果
	if got != val {
		t.Errorf("❌ Verification Failed! Expected '%s', but got '%s'", val, got)
	} else {
		t.Logf("✅ Success! Got expected value: %s", got)
	}
}