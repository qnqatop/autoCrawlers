package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"qnqa-auto-crawlers/pkg/crawlers"
	"qnqa-auto-crawlers/pkg/limitgroup"
	"qnqa-auto-crawlers/pkg/logger"

	"github.com/rabbitmq/amqp091-go"
)

// Client представляет клиент RabbitMQ
type Client struct {
	logger.Logger
	conn          *amqp091.Connection
	channel       *amqp091.Channel
	queue         map[string]amqp091.Queue
	deserializers map[string]crawlers.TaskDeserializer
	mu            sync.RWMutex
}

type taskPayload struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (p *taskPayload) Source() string {
	return strings.Split(p.Type, ".")[0]
}

// NewClient создает новый клиент RabbitMQ
func NewClient(url string, lg logger.Logger) (*Client, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		err = conn.Close()
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
		errCh := ch.Close()
		if errCh != nil {
			err = fmt.Errorf("failed to close channel: %w", err)
		}
		errConn := conn.Close()
		if errConn != nil {
			err = fmt.Errorf("failed to close connection: %w", err)
		}
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Объявляем очередь для задач
	cq, err := ch.QueueDeclare(
		"car_tasks", // имя очереди
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		errCh := ch.Close()
		if errCh != nil {
			err = fmt.Errorf("failed to close channel: %w", err)
		}
		errConn := conn.Close()
		if errConn != nil {
			err = fmt.Errorf("failed to close connection: %w", err)
		}
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	m := make(map[string]amqp091.Queue)
	m["car"] = cq
	m["list"] = q

	return &Client{
		conn:          conn,
		channel:       ch,
		queue:         m,
		Logger:        lg,
		deserializers: make(map[string]crawlers.TaskDeserializer),
		mu:            sync.RWMutex{},
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

func (c *Client) RegisterDeserializer(taskType string, deserializer crawlers.TaskDeserializer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deserializers[taskType] = deserializer
}

// PublishTask публикует задачу в очередь
func (c *Client) PublishTask(ctx context.Context, queueName string, task crawlers.Task) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payload := taskPayload{
		Type: task.Type(),
		Data: nil,
	}
	data, err := json.Marshal(task)
	if err != nil {
		c.Logger.Errorf("failed to marshal task err=%v", err)
		return err
	}
	payload.Data = data
	payloadData, err := json.Marshal(payload)
	if err != nil {
		c.Logger.Errorf("failed to marshal task err=%v", err)
		return err
	}

	err = c.channel.PublishWithContext(ctx,
		"",                      // exchange
		c.queue[queueName].Name, // routing key
		false,                   // mandatory
		false,                   // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        payloadData,
		})
	if err != nil {
		c.Logger.Errorf("failed to publish task err=%v", err)
		return err
	}
	return nil
}

// ConsumeTasks начинает потребление задач из очереди
func (c *Client) ConsumeTasks(ctx context.Context, queueName string, handler func(context.Context, crawlers.Task) error) {
	msgs, err := c.channel.Consume(
		c.queue[queueName].Name, // queue
		"",                      // consumer
		true,                    // auto-ack
		false,                   // exclusive
		false,                   // no-local
		false,                   // no-wait
		nil,                     // args
	)
	if err != nil {
		c.Logger.Errorf("failed to register a consumer: %v", err)
	}

	lg, _ := limitgroup.New(ctx, 5)
	for msg := range msgs {
		lg.Go(func() error {
			var payload taskPayload
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				return fmt.Errorf("failed to unmarshal payload: %w", err)
			}
			c.mu.RLock()
			deserializer, ok := c.deserializers[payload.Type]
			c.mu.RUnlock()
			if !ok {
				return fmt.Errorf("unknown task type: %s", payload.Type)
			}

			task, err := deserializer.Deserialize(payload.Data, payload.Source())
			if err != nil {
				return fmt.Errorf("failed to deserialize task %s: %w", payload.Type, err)
			}

			if err = handler(ctx, task); err != nil {
				c.Logger.Errorf("Failed to handle task: %v", err)
			}
			return nil
		})
	}

	err = lg.Wait()
	if err != nil {
		c.Logger.Errorf("failed to consume tasks: %v", err)
	}
}
