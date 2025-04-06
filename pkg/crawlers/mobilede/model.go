package mobilede

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

type ListParse struct {
}

type AllJson struct {
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
