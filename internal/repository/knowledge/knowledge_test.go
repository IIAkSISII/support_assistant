package knowledge

import (
	"path/filepath"
	"testing"
)

func newTestRepository(t *testing.T) *JsonRepository {
	path := filepath.Join("testdata", "knowledge_base.json")

	repo, err := NewJsonRepository(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	return repo
}

func TestJsonRepository_LoadKnowledge(t *testing.T) {
	repo := newTestRepository(t)

	if len(repo.entries) == 0 {
		t.Fatal("expected knowledge base to contain entries")
	}
}

func TestJsonRepository_FindAnswerByCategory(t *testing.T) {
	repo := newTestRepository(t)

	answer, found := repo.FindAnswer("auth", nil)
	if !found {
		t.Fatal("expected answer to be found")
	}

	expected := "Попробуйте восстановить пароль через страницу входа. Если письмо не пришло, проверьте папку Спам."

	if answer != expected {
		t.Errorf("expected %q, got %q", expected, answer)
	}
}

func TestJSONRepository_FindAnswerByCategoryAndKeyword(t *testing.T) {
	repo := newTestRepository(t)

	answer, found := repo.FindAnswer("payment", []string{"подписка"})
	if !found {
		t.Fatal("expected answer to be found")
	}

	expected := "Если оплата прошла, но доступ не появился, пришлите email аккаунта и номер платежа."

	if answer != expected {
		t.Errorf("expected %q, got %q", expected, answer)
	}
}

func TestJSONRepository_FindAnswerByKeyword(t *testing.T) {
	repo := newTestRepository(t)

	answer, found := repo.FindAnswer("", []string{"логин"})
	if !found {
		t.Fatal("expected answer to be found")
	}

	expected := "Проверьте правильность логина и пароля. Если войти не получается, попробуйте восстановить пароль или напишите email аккаунта."

	if answer != expected {
		t.Errorf("expected %q, got %q", expected, answer)
	}
}

func TestJSONRepository_FindAnswerReturnsFalseWhenNotFound(t *testing.T) {
	repo := newTestRepository(t)

	answer, found := repo.FindAnswer("unknown", []string{"something"})
	if found {
		t.Fatal("expected answer not to be found")
	}

	if answer != "" {
		t.Errorf("expected empty answer, got %q", answer)
	}
}
