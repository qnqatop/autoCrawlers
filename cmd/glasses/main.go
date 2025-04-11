package main

import (
	"context"
	"fmt"
	"qnqa-auto-crawlers/pkg/crawlers/glasses/dita"
	"qnqa-auto-crawlers/pkg/logger"
)

func main() {
	cr := dita.NewCrawler(logger.NewLogger(true))
	err := cr.Start(context.Background(), nil)
	if err != nil {
		fmt.Println(err)
	}
}
