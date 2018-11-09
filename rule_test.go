package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFqdnRule_Matches(t *testing.T) {
	ta := assert.New(t)
	rule := &FQDNRule{
		RuleImpl{
			expression: "www.google.com",
		},
	}
	ta.True(rule.Matches("www.google.com"))
	ta.True(rule.Matches("www.google.com."))
	ta.False(rule.Matches("www.google1.com"))

	rule.expression = "www.google.com."
	ta.True(rule.Matches("www.google.com"))
	ta.True(rule.Matches("www.google.com."))
	ta.False(rule.Matches("www.google1.com"))
}

func TestPrefixRule_Matches(t *testing.T) {
	ta := assert.New(t)
	rule := &PrefixRule{
		RuleImpl{
			expression: "www.google",
		},
	}
	ta.True(rule.Matches("www.google.com"))
	ta.True(rule.Matches("www.google"))
	ta.False(rule.Matches("www.google1.com"))

	rule.expression = "www.google."
	ta.True(rule.Matches("www.google.com"))
	ta.True(rule.Matches("www.google"))
	ta.False(rule.Matches("www.google1.com"))
}

func TestSuffixRule_Matches(t *testing.T) {
	ta := assert.New(t)
	rule := &SuffixRule{
		RuleImpl{
			expression: "google.com",
		},
	}
	ta.True(rule.Matches("www.google.com"))
	ta.True(rule.Matches("google.com"))
	ta.False(rule.Matches("www.higoogle.com"))

	rule.expression = ".google.com"
	ta.True(rule.Matches("www.google.com"))
	ta.True(rule.Matches("google.com"))
	ta.False(rule.Matches("www.higoogle.com"))
}
