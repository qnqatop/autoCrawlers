package mobilede

type ModelsJSON struct {
	Data []Items `json:"data"`
}

type Items struct {
	Value         string `json:"value"`
	Label         string `json:"label"`
	OptgroupLabel string `json:"optgroupLabel"`
	Items         []InternalItem
}
type InternalItem struct {
	Value   string `json:"value"`
	Label   string `json:"label"`
	isGroup bool   `json:"isGroup"`
}
