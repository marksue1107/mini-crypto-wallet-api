package kafkaclient

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
)

type TxCreatedMessage struct {
	Hash       string  `json:"hash"`
	FromUserID uint    `json:"from_user_id"`
	ToUserID   uint    `json:"to_user_id"`
	Amount     float64 `json:"amount"`
	Timestamp  string  `json:"timestamp"`
}

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokerAddr string, topic string) *KafkaProducer {
	// 嘗試建立 topic（如不存在）
	if err := createTopic(brokerAddr, topic, 1, 1); err != nil {
		log.Printf("⚠️ Kafka topic create failed: %v", err)
	}

	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokerAddr),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (kp *KafkaProducer) SendTxCreated(msg TxCreatedMessage) error {
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err = kp.writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(msg.Hash),
		Value: bytes,
	}); err != nil {
		log.Println("Kafka write error:", err)
		return err
	}

	log.Println("✅ Kafka: tx.created sent successfully")

	return nil
}

func (kp *KafkaProducer) Close() {
	if err := kp.writer.Close(); err != nil {
		log.Println("Failed to close Kafka writer:", err)
	}
}

func createTopic(broker, topic string, numPartitions, replicationFactor int) error {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 檢查是否已存在
	partitions, err := conn.ReadPartitions()
	if err != nil {
		return err
	}
	for _, p := range partitions {
		if p.Topic == topic {
			return nil // 已存在
		}
	}

	topicConfig := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
	}
	return conn.CreateTopics(topicConfig)
}
