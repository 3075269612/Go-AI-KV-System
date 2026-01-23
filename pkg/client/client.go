package client

import (
	"Go-AI-KV-System/pkg/protocol"
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

// Client 结构体
type Client struct {
	addr string 	// 服务端地址 "localhost:8080"
	conn net.Conn	// 当前持有的长连接
	mu sync.Mutex	// 防止并发写入导致粘包混乱
}

// NewClient 初始化客户端并建立连接
func NewClient(addr string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 3 * time.Second)
	if err != nil {
		return nil, err
	}
	return &Client{
		addr: addr,
		conn: conn,
	}, nil
}

// 核心接口
// Set 发送 SET 命令
func (c *Client) Set(key, value string) error {
	command := fmt.Sprintf("SET %s %s", key, value)
	_, err := c.sendRequest(command)
	return err
}

// Get 发送 GET 命令
func (c *Client) Get(key string) (string, error) {
	command := fmt.Sprintf("GET %s", key)

	resp, err := c.sendRequest(command)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// sendRequest 封装底层的封包和拆包逻辑
// 这是 SDK 最核心的部分：屏蔽网络细节
func (c *Client) sendRequest(msg string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. 检查连接状态（简单的重连机制）
	if c.conn == nil {
		var err error
		c.conn, err = net.DialTimeout("tcp", c.addr, 3 * time.Second)
		if err != nil {
			return "", err
		}
	}

	// 2. 封包（Encode）
	data, err := protocol.Encode(msg)
	if err != nil {
		return "", err
	}

	// 3. 发送
	_, err = c.conn.Write(data)
	if err != nil {
		c.conn.Close()
		c.conn = nil
		return "", err
	}

	// 4. 接受响应（Decode）
	reader := bufio.NewReader(c.conn)
	responseMsg, err := protocol.Decode(reader)
	if err != nil {
		c.conn.Close()
		c.conn = nil
		return "", err
	}
	return responseMsg, nil
}

// 关闭资源
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}