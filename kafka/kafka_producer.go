package kafka

import (
	"log"
	"time"

	"github.com/IBM/sarama"
)

const (
	kafkaBrokers = "localhost:9092"
	kafkaTopic   = "deribitOrderbook"
	maxRetry     = 5
)

// InitKafkaProducer initializes a new synchronous Kafka producer.
func InitKafkaProducer() (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForLocal
	producer, err := sarama.NewSyncProducer([]string{kafkaBrokers}, config)
	if err != nil {
		return nil, err
	}
	return producer, nil
}

// ProduceToKafka sends a provided message to Kafka.
func ProduceToKafka(producer sarama.SyncProducer, msg []byte) error {
	_, _, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic: kafkaTopic,
		Value: sarama.ByteEncoder(msg),
	})
	return err
}

// RetryProduceToKafka retries sending a message to Kafka up to 5 times,
// with exponential backoff for handling transient failures.
func RetryProduceToKafka(producer sarama.SyncProducer, msg []byte) {
	for attempt := 1; attempt <= maxRetry; attempt++ {
		err := ProduceToKafka(producer, msg)
		if err == nil {
			log.Printf("[RetryProduceToKafka] Successfully sent message to Kafka on attempt %d", attempt)
			return
		}
		log.Printf("[RetryProduceToKafka] Failed to send to Kafka, attempt %d: %s", attempt, err)
		time.Sleep(time.Duration(attempt*2) * time.Second)
	}
	log.Println("[RetryProduceToKafka] Failed to send message to Kafka after 5 attempts")
}
