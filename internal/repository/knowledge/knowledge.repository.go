package knowledge

import (
	"encoding/json"
	"errors"
	"github.com/IIAkSISII/support-assistant/internal/model"
	"os"
	"strings"
)

type KnowledgeRepository interface {
	FindAnswer(category string, keywords []string) (string, bool)
}

type JsonRepository struct {
	entries []model.Entry
}

func NewJsonRepository(path string) (*JsonRepository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var entries []model.Entry

	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, errors.New("knowledge base is empty")
	}

	return &JsonRepository{
		entries: entries,
	}, nil
}

func (j *JsonRepository) FindAnswer(category string, keywords []string) (string, bool) {
	normalizedCategory := normalize(category)
	normalizedKeywords := normalizeKeywords(keywords)

	if normalizedCategory != "" && len(normalizedKeywords) > 0 {
		return j.findByCategoryAndKeywords(normalizedCategory, normalizedKeywords)
	}
	if normalizedCategory != "" {
		return j.findByCategory(normalizedCategory)
	}
	if len(normalizedKeywords) > 0 {
		return j.findByKeywords(normalizedKeywords)
	}
	return "", false
}

func (j *JsonRepository) findByCategoryAndKeywords(category string, keyword map[string]struct{}) (string, bool) {
	for _, entry := range j.entries {
		if normalize(entry.Category) != category {
			continue
		}
		if hasKeywordMatch(entry.Keywords, keyword) {
			return entry.Answer, true
		}
	}
	return "", false
}

func (j *JsonRepository) findByCategory(category string) (string, bool) {
	for _, entry := range j.entries {
		if normalize(entry.Category) == category {
			return entry.Answer, true
		}
	}
	return "", false
}

func (j *JsonRepository) findByKeywords(keywords map[string]struct{}) (string, bool) {
	for _, entry := range j.entries {
		if hasKeywordMatch(entry.Keywords, keywords) {
			return entry.Answer, true
		}
	}
	return "", false
}

func hasKeywordMatch(entryKeywords []string, requestedKeywords map[string]struct{}) bool {
	for _, k := range entryKeywords {
		normalizedKeyword := normalize(k)
		if _, ok := requestedKeywords[normalizedKeyword]; ok {
			return true
		}
	}
	return false
}

func normalizeKeywords(keyword []string) map[string]struct{} {
	result := make(map[string]struct{}, len(keyword))

	for _, k := range keyword {
		normalizedKeyword := normalize(k)

		if normalizedKeyword == "" {
			continue
		}
		result[normalizedKeyword] = struct{}{}
	}
	return result
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
