package rabbitmq

import "encoding/json"

// Task представляет задачу на парсинг
type Task struct {
	BrandID string `json:"brand_id"`
	ModelID string `json:"model_id"`
	Page    int    `json:"page"`
	Url     string `json:"url"`
}

func (t *Task) Model(data interface{}) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &data)

	return err
}

func (t *Task) Byte() []byte {
	b, err := json.Marshal(t)
	if err != nil {
		return nil
	}
	return b
}
