package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"qnqa-auto-crawlers/pkg/crawlers"
	"qnqa-auto-crawlers/pkg/limitgroup"
	"qnqa-auto-crawlers/pkg/logger"

	"github.com/rabbitmq/amqp091-go"
)

// Client представляет клиент RabbitMQ
type Client struct {
	logger.Logger
	conn    *amqp091.Connection
	channel *amqp091.Channel
	queue   map[string]amqp091.Queue
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
		conn:    conn,
		channel: ch,
		queue:   m,
		Logger:  lg,
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
func (c *Client) PublishTask(ctx context.Context, queueName string, task crawlers.Tasker) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	body := task.Byte()
	err := c.channel.PublishWithContext(ctx,
		"",                      // exchange
		c.queue[queueName].Name, // routing key
		false,                   // mandatory
		false,                   // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		c.Logger.Errorf("failed to publish task err=%v", err)
		return err
	}
	return nil
}

// ConsumeTasks начинает потребление задач из очереди
func (c *Client) ConsumeTasks(ctx context.Context, queueName string, handler func(context.Context, crawlers.Tasker) error) {
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
			var task Task
			if err := json.Unmarshal(msg.Body, &task); err != nil {
				c.Logger.Errorf("Failed to unmarshal task: %v", err)
			}
			if err = handler(ctx, &task); err != nil {
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
