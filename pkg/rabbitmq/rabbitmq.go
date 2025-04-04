package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

// Task представляет задачу на парсинг
type Task struct {
	BrandID string `json:"brand_id"`
	ModelID string `json:"model_id"`
	Page    int    `json:"page"`
}

// Client представляет клиент RabbitMQ
type Client struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	queue   amqp091.Queue
}

// NewClient создает новый клиент RabbitMQ
func NewClient(url string) (*Client, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Объявляем очередь для задач
	q, err := ch.QueueDeclare(
		"list_tasks", // имя очереди
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &Client{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

// Close закрывает соединение с RabbitMQ
func (c *Client) Close() error {
	if err := c.channel.Close(); err != nil {
		return fmt.Errorf("failed to close channel: %w", err)
	}
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}
	return nil
}

// PublishTask публикует задачу в очередь
func (c *Client) PublishTask(ctx context.Context, task *Task) error {
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = c.channel.PublishWithContext(ctx,
		"",           // exchange
		c.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish task: %w", err)
	}

	return nil
}

// ConsumeTasks начинает потребление задач из очереди
func (c *Client) ConsumeTasks(ctx context.Context, handler func(*Task) error) error {
	msgs, err := c.channel.Consume(
		c.queue.Name, // queue
		"",           // consumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	go func() {
		for msg := range msgs {
			var task Task
			if err := json.Unmarshal(msg.Body, &task); err != nil {
				fmt.Printf("Failed to unmarshal task: %v\n", err)
				continue
			}

			if err := handler(&task); err != nil {
				fmt.Printf("Failed to handle task: %v\n", err)
			}
		}
	}()

	return nil
}
