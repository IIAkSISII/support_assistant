package model

type Entry struct {
	Category string   `json:"category"`
	Keywords []string `json:"keywords"`
	Answer   string   `json:"answer"`
}
