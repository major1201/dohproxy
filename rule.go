package main

import (
	"github.com/major1201/goutils"
	"go.uber.org/zap"
	"regexp"
	"strings"
)

// Rule describes the DNS rule interface
type Rule interface {
	Matches(address string) bool

	Expression() string
	SetExpression(o string)

	Upstream() Upstream
	SetUpstream(o Upstream)

	StaticResult() string
	SetStaticResult(o string)
}

// RuleImpl is the implement of Rule interface
type RuleImpl struct {
	expression   string
	upstream     Upstream
	staticResult string
}

// Expression returns the expression of a rule
func (r *RuleImpl) Expression() string {
	return r.expression
}

// SetExpression set the rule expression attribute
func (r *RuleImpl) SetExpression(o string) {
	r.expression = o
}

// Upstream returns the upstream of a rule
func (r *RuleImpl) Upstream() Upstream {
	return r.upstream
}

// SetUpstream set the rule upstream attribute
func (r *RuleImpl) SetUpstream(o Upstream) {
	r.upstream = o
}

// StaticResult returns the static result of a rule
func (r *RuleImpl) StaticResult() string {
	return r.staticResult
}

// SetStaticResult set the rule static result attribute
func (r *RuleImpl) SetStaticResult(o string) {
	r.staticResult = o
}

// FQDNRule matches a domain by FQDN
type FQDNRule struct {
	RuleImpl
}

// PrefixRule matches a domain by prefix
type PrefixRule struct {
	RuleImpl
}

// SuffixRule matches a domain by suffix
type SuffixRule struct {
	RuleImpl
}

// KeywordRule matches a domain by keyword
type KeywordRule struct {
	RuleImpl
}

// WildcardRule matches a domain by wildcard
type WildcardRule struct {
	RuleImpl
}

// RegexRule matches a domain by regex
type RegexRule struct {
	RuleImpl
	regex *regexp.Regexp
}

// Matches returns if the address matches the FQDN rule
func (rule *FQDNRule) Matches(address string) bool {
	return strings.TrimSuffix(rule.expression, ".") == strings.TrimSuffix(strings.ToLower(address), ".")
}

// Matches returns if the address matches the prefix rule
func (rule *PrefixRule) Matches(address string) bool {
	return strings.HasPrefix(fillBothDots(strings.ToLower(address)), fillBothDots(rule.expression))
}

// Matches returns if the address matches the suffix rule
func (rule *SuffixRule) Matches(address string) bool {
	return strings.HasSuffix(fillBothDots(strings.ToLower(address)), fillBothDots(rule.expression))
}

// Matches returns if the address matches the keyword rule
func (rule *KeywordRule) Matches(address string) bool {
	return strings.Index(strings.ToLower(address), rule.expression) >= 0
}

// Matches returns if the address matches the wildcard rule
func (rule *WildcardRule) Matches(address string) bool {
	return goutils.WildcardMatch(rule.expression, strings.ToLower(address))
}

// Matches returns if the address matches the regex rule
func (rule *RegexRule) Matches(address string) bool {
	return rule.regex.MatchString(strings.ToLower(address))
}

// AddRule converts a rule in raw string into Rule and appends it the handler rules
func (handler *Handler) AddRule(text string) {
	logger := zap.L().Named("config")

	parts := strings.Fields(text)
	if len(parts) != 2 {
		logger.Fatal("rule fields must be 2 parts", zap.Strings("fields", parts))
	}

	condition := strings.Split(parts[0], ":")
	if len(condition) != 2 {
		logger.Fatal("rule condition must be 2 parts", zap.Strings("condition", condition))
	}

	conditionType := condition[0]

	var rule Rule
	switch conditionType {
	case "fqdn":
		rule = &FQDNRule{}
	case "prefix":
		rule = &PrefixRule{}
	case "suffix":
		rule = &SuffixRule{}
	case "keyword":
		rule = &KeywordRule{}
	case "wildcard":
		rule = &WildcardRule{}
	case "regex":
		regex, err := regexp.Compile(condition[1])
		if err != nil {
			logger.Fatal("regex compile failed", zap.String("exp", condition[1]))
		}
		rule = &RegexRule{regex: regex}
	default:
		logger.Fatal("unknown condition type", zap.String("type", conditionType))
	}
	rule.SetExpression(strings.ToLower(condition[1]))

	// upstream
	if upstream, ok := handler.Upstreams[parts[1]]; ok {
		rule.SetUpstream(upstream)
	} else {
		if goutils.IsIPv4(parts[1]) {
			rule.SetStaticResult(parts[1])
		} else {
			logger.Fatal("unknown upstream", zap.String("name", parts[1]))
		}
	}

	handler.Rules = append(handler.Rules, rule)
}
