package translate

import (
	gt "github.com/bas24/googletranslatefree"
)

func Translate(text string) string {
	result, err := gt.Translate(text, "de", "en")
	if err != nil {
		return ""
	}
	return result
}
