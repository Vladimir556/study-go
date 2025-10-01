package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"auth-app/internal/config"
	"auth-app/internal/models"

	"github.com/segmentio/kafka-go"
)

type KafkaService struct {
	config  *config.Config
	writer  *kafka.Writer
	readers map[string]*kafka.Reader
	enabled bool
}

func NewKafkaService(cfg *config.Config) *KafkaService {
	k := &KafkaService{
		config:  cfg,
		readers: make(map[string]*kafka.Reader),
		enabled: cfg.KafkaEnabled,
	}

	if !k.enabled {
		log.Println("Kafka is disabled")
		return k
	}

	// Инициализация Kafka writer
	k.writer = &kafka.Writer{
		Addr:                   kafka.TCP(strings.Split(cfg.KafkaBrokers, ",")...),
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireOne,
		AllowAutoTopicCreation: true,
		Async:                  true, // Асинхронная отправка для производительности
	}

	// Инициализация readers для каждого топика
	topics := strings.Split(cfg.KafkaTopics, ",")
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		k.readers[topic] = kafka.NewReader(kafka.ReaderConfig{
			Brokers:  strings.Split(cfg.KafkaBrokers, ","),
			GroupID:  cfg.KafkaGroupID,
			Topic:    topic,
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		})
	}

	log.Printf("Kafka service initialized with brokers: %s", cfg.KafkaBrokers)
	return k
}

func (k *KafkaService) ProduceEvent(ctx context.Context, eventType models.EventType, data interface{}) error {
	if !k.enabled {
		return nil // Игнорируем если Kafka отключен
	}

	event := models.Event{
		ID:        generateID(),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
		Source:    "auth-app",
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	topic := getTopicForEventType(eventType)

	err = k.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.ID),
		Value: eventBytes,
		Time:  time.Now(),
	})

	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	log.Printf("Produced event to topic '%s': %s", topic, event.Type)
	return nil
}

func (k *KafkaService) ConsumeEvents(ctx context.Context, topic string, handler func(event models.Event) error) error {
	if !k.enabled {
		return nil
	}

	reader, exists := k.readers[topic]
	if !exists {
		return fmt.Errorf("no reader for topic: %s", topic)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					log.Printf("Error reading from kafka: %v", err)
					continue
				}

				var event models.Event
				if err := json.Unmarshal(msg.Value, &event); err != nil {
					log.Printf("Error unmarshaling event: %v", err)
					continue
				}

				if err := handler(event); err != nil {
					log.Printf("Error handling event: %v", err)
				}
			}
		}
	}()

	log.Printf("Started consuming from topic: %s", topic)
	return nil
}

func (k *KafkaService) Close() {
	if k.writer != nil {
		k.writer.Close()
	}

	for _, reader := range k.readers {
		reader.Close()
	}

	log.Println("Kafka service closed")
}

// Вспомогательные функции
func generateID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

func getTopicForEventType(eventType models.EventType) string {
	switch eventType {
	case models.UserRegistered:
		return "user-registered"
	case models.UserLoggedIn:
		return "user-logged-in"
	case models.UserProfileViewed:
		return "user-profile-viewed"
	default:
		return "general-events"
	}
}
