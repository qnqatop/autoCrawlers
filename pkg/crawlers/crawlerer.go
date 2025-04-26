package crawlers

import (
	"context"
)

type (
	// Crawler осовной интерфейс для краулеров
	Crawler interface {
		PageParse(ctx context.Context, task Task) error
		ListParse(ctx context.Context, task Task) error
		ModelParse(ctx context.Context) error
		BrandParse(ctx context.Context) error
	}
	// Task - основной интерфейс для тасок краулера
	Task interface {
		Type() string
	}
	// TaskDeserializer интерфейс для десериализации задач
	TaskDeserializer interface {
		Deserialize(data []byte, taskType string) (Task, error)
	}
)
