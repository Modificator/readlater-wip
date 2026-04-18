package service

import (
	"testing"
	"time"

	"github.com/Modificator/readlater-wip/internal/model"
	"github.com/Modificator/readlater-wip/internal/store"
)

func TestMatchRule(t *testing.T) {
	repo := store.NewMemoryRepository()
	svc := NewRuleService(repo)
	r1 := &model.ExtractionRule{ID: "r1", Name: "prefix", MatchType: model.MatchTypePrefix, MatchExpression: "https://example.com/blog/", Priority: 1, Enabled: true, CreatedAt: time.Now().UTC()}
	r2 := &model.ExtractionRule{ID: "r2", Name: "regex", MatchType: model.MatchTypeRegex, MatchExpression: `^https://example\.com/blog/.*$`, Priority: 10, Enabled: true, CreatedAt: time.Now().UTC()}
	repo.SaveRule(r1)
	repo.SaveRule(r2)

	r, err := svc.MatchRule("https://example.com/blog/post", "")
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "r2" {
		t.Fatalf("expected regex rule r2, got %s", r.ID)
	}
}
