package service

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Modificator/readlater-wip/internal/model"
	"github.com/Modificator/readlater-wip/internal/store"
)

type RuleService struct {
	repo *store.MemoryRepository
}

func NewRuleService(repo *store.MemoryRepository) *RuleService {
	return &RuleService{repo: repo}
}

func (s *RuleService) CreateRule(rule *model.ExtractionRule) error {
	if rule.Name == "" || rule.MatchExpression == "" {
		return errors.New("name and match_expression are required")
	}
	if rule.MatchType != model.MatchTypePrefix && rule.MatchType != model.MatchTypeRegex {
		return errors.New("match_type must be prefix or regex")
	}
	if rule.MatchType == model.MatchTypeRegex {
		if _, err := regexp.Compile(rule.MatchExpression); err != nil {
			return errors.New("invalid regex match_expression")
		}
	}
	now := time.Now().UTC()
	rule.CreatedAt, rule.UpdatedAt = now, now
	rule.Enabled = true
	s.repo.SaveRule(rule)
	return nil
}

func (s *RuleService) ListRules() []*model.ExtractionRule {
	return s.repo.ListRules()
}

func (s *RuleService) MatchRule(url string, explicitRuleID string) (*model.ExtractionRule, error) {
	if explicitRuleID != "" {
		return s.repo.GetRule(explicitRuleID)
	}

	rules := s.repo.ListRules()
	regexCandidates := make([]*model.ExtractionRule, 0)
	prefixCandidates := make([]*model.ExtractionRule, 0)

	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		switch r.MatchType {
		case model.MatchTypeRegex:
			re, err := regexp.Compile(r.MatchExpression)
			if err != nil {
				continue
			}
			if re.MatchString(url) {
				regexCandidates = append(regexCandidates, r)
			}
		case model.MatchTypePrefix:
			if strings.HasPrefix(url, r.MatchExpression) {
				prefixCandidates = append(prefixCandidates, r)
			}
		}
	}

	if len(regexCandidates) > 0 {
		sort.Slice(regexCandidates, func(i, j int) bool {
			if regexCandidates[i].Priority == regexCandidates[j].Priority {
				return regexCandidates[i].CreatedAt.Before(regexCandidates[j].CreatedAt)
			}
			return regexCandidates[i].Priority > regexCandidates[j].Priority
		})
		return regexCandidates[0], nil
	}

	if len(prefixCandidates) > 0 {
		sort.Slice(prefixCandidates, func(i, j int) bool {
			li := len(prefixCandidates[i].MatchExpression)
			lj := len(prefixCandidates[j].MatchExpression)
			if li == lj {
				if prefixCandidates[i].Priority == prefixCandidates[j].Priority {
					return prefixCandidates[i].CreatedAt.Before(prefixCandidates[j].CreatedAt)
				}
				return prefixCandidates[i].Priority > prefixCandidates[j].Priority
			}
			return li > lj
		})
		return prefixCandidates[0], nil
	}

	return nil, store.ErrNotFound
}
