package protocol

import (
	"encoding/binary"
	"io"
)

// Encode 打包
// 输入："SET a 1" -> 输出：[0,0,0,7] + "SET a 1"(二进制流)
func Encode(message string) ([]byte, error) {
	// 1. 计算原始消息的字节长度
	length := uint32(len(message))

	// 2. 申请内存：4字节长度头 + 消息内容的字节长度
	pkg := make([]byte, 4 + length)

	// 3. 将长度值写入前4字节（大端序，符合网络传输标准）
	binary.BigEndian.PutUint32(pkg[:4], length)

	// 4. 把原始消息复制到长度头之后的位置
	copy(pkg[4:], []byte(message))

	return pkg, nil
}

// Decode 拆包
func Decode(reader io.Reader) (string, error) {
	// 1. 先读取前4字节的长度头
	headerBuf := make([]byte, 4)
	// io.ReadFull 保证读满4字节，否则阻塞等待
	if _, err := io.ReadFull(reader, headerBuf); err != nil {
		return "", err
	}

	// 2. 解析长度头，得到消息内容的长度
	length := binary.BigEndian.Uint32(headerBuf)

	// 3. 根据解析出的长度，读取消息内容
	bodyBuf := make([]byte, length)
	if _, err := io.ReadFull(reader, bodyBuf); err != nil {
		return "", err
	}

	return string(bodyBuf), nil
}