package handler

import (
	"Go-AI-KV-System/pkg/client"
	"net/http"

	"github.com/gin-gonic/gin"
)

// KVHandler 持有 gRPC 客户端的指针
type KVHandler struct {
	cli *client.Client
}

func NewKVHandler(cli *client.Client) *KVHandler {
	return &KVHandler{
		cli: cli,
	}
}

// HandleSet 处理 SET 请求
// POST /api/v1/kv
// Body: {"key": "name", "value": "naato"}
func (h *KVHandler) HandleSet(c *gin.Context) {
	// 定义请求体结构
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	// 1. 解析 JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 2. 调用 gRPC 客户端
	err := h.cli.Set(req.Key, req.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "存储失败: " + err.Error()})
		return
	}

	// 3. 返回成功
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"key":     req.Key,
		"value":   req.Value,
	})
}

// HandlerGet 处理 GET 请求
// GET /api/v1/kv?key=name
func (h *KVHandler) HandleGet(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 key 参数"})
		return
	}

	// 调用 gRPC
	val, err := h.cli.Get(key)
	if err != nil {
		// 简单处理，后续可优化
		c.JSON(http.StatusNotFound, gin.H{"error": "查询失败或 Key 不存在: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key": key,
		"value": val,
	})
}

// HandleDel 处理 DEL 请求
// DELETE /api/v1/kv?key=name
func (h *KVHandler) HandleDel(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 key 参数"})
		return
	}

	err := h.cli.Del(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted", "key": key})
}