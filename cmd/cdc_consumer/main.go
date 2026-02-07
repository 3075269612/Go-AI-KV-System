package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventType int

const (
	EventSet EventType = iota
	EventDel
)

type Event struct {
	Type  EventType   `json:"type"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

const (
	ExchangeName = "flux_kv_events"
	QueueName = "flux_cdc_file_logger"
	LogFileName = "flux_cdc.log"
	AmqpURL = "amqp://guest:guest@localhost:5672/"
)

func main() {
	// 1. 连接 RabbitMQ (建立连接 + 打开通道)
	conn, err := amqp.Dial(AmqpURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// 2. 声明交换机
	err = ch.ExchangeDeclare(
		ExchangeName,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare exchange")

	// 3. 声明队列
	q, err := ch.QueueDeclare(
		QueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare queue")

	// 4. 绑定队列到交换机
	err = ch.QueueBind(
		q.Name, "", ExchangeName, false, nil,
	)
	failOnError(err, "Failed to bind queue")

	// 5. 注册消费者
	msgs, err := ch.Consume(
		q.Name,
		"",
		true,	// 自动确认消息
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register consumer")

	// 6. 打开日志文件
	logFile, err := os.OpenFile(LogFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	failOnError(err, "Failed to open log file")
	defer logFile.Close()

	log.Printf("[*] Waiting for CDC events. To exit press CTRL+C")

	// 7. 处理消息循环
	go func() {
		for d := range msgs {
			var event Event
			// 反序列化 JSON 消息体
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("Error decoding JSON: %s", err)
				continue
			}

			// 从 AMQP 消息头获取时间戳
			eventTime := d.Timestamp
			if eventTime.IsZero() {
				eventTime = time.Now()
			}

			processEvent(logFile, event, eventTime)
		}
	}()

	// 优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan 	// 阻塞等待信号
	log.Println("Shutting down CDC consumer")
}

func processEvent(f *os.File, e Event, t time.Time) {
	timeStr := t.Format(time.RFC3339)
	var logLine string

	// 还原操作类型字符串
	op := "UNKNOWN"
	if e.Type == EventSet {
		op = "SET"
	} else if e.Type == EventDel {
		op = "DEL"
	}

	// 构造不同操作类型的日志行
	if e.Type == EventSet {
		valStr := fmt.Sprintf("%v", e.Value)
		logLine = fmt.Sprintf("[%s] [CDC_SYNC] %s key='%s' value_len=%d >> Persisted\n", timeStr, op, e.Key, len(valStr))
	} else {
		logLine = fmt.Sprintf("[%s] [CDC_SYNC] %s key='%s' >> Deleted\n", timeStr, op, e.Key)
	}

	// 写入日志文件
	if _, err := f.WriteString(logLine); err != nil {
		log.Printf("Error writing to file: %v", err)
	}
	fmt.Print(logLine)	// 同时打印到控制台
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}