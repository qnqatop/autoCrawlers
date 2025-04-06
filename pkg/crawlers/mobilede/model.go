package mobilede

import "encoding/json"

type ModelsJSON struct {
	Data []DataItem `json:"data"`
}

type DataItem struct {
	Value         string `json:"value"`
	Label         string `json:"label"`
	OptgroupLabel string `json:"optgroupLabel"`
	Items         []InternalItem
}
type InternalItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ListParseResponse struct {
	HasNextPage bool   `json:"hasNextPage"`
	Items       []Item `json:"items"`
}

type Item struct {
	IsEyeCatcher bool   `json:"isEyeCatcher"`
	NumImages    int    `json:"numImages"`
	RelativePath string `json:"relativeUrl"`
	Id           int    `json:"id"`
}

// Response представляет стандартный ответ API
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ListParseTask struct {
	Url string `json:"url"`
}

func (lpt *ListParseTask) Model(data interface{}) error {
	b, err := json.Marshal(lpt)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &data)
	return err
}

func (lpt *ListParseTask) Byte() []byte {
	b, err := json.Marshal(lpt)
	if err != nil {
		return nil
	}
	return b
}

type CarParseTask struct {
	RelativePath string `json:"relativePath"`
	ExternalId   int    `json:"externalId"`
}

func (cpt *CarParseTask) Model(data interface{}) error {
	b, err := json.Marshal(cpt)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &data)
	return err
}

func (cpt *CarParseTask) Byte() []byte {
	b, err := json.Marshal(cpt)
	if err != nil {
		return nil
	}
	return b
}
