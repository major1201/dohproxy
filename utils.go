package main

import "strings"

func fillLeftDot(s string) string {
	if !strings.HasPrefix(s, ".") {
		return "." + s
	}
	return s
}

func fillRightDot(s string) string {
	if !strings.HasSuffix(s, ".") {
		return s + "."
	}
	return s
}

func fillBothDots(s string) string {
	return fillRightDot(fillLeftDot(s))
}
