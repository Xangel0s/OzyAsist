package agent

import (
	"regexp"
	"strings"
)

var actionVerbs = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bmodific`),
	regexp.MustCompile(`(?i)\barregl`),
	regexp.MustCompile(`(?i)\bimplement`),
	regexp.MustCompile(`(?i)\bagreg`),
	regexp.MustCompile(`(?i)\bborr`),
	regexp.MustCompile(`(?i)\belimin`),
	regexp.MustCompile(`(?i)\bcorrig`),
	regexp.MustCompile(`(?i)\brefactoriz`),
	regexp.MustCompile(`(?i)\bcrea.*archivo`),
	regexp.MustCompile(`(?i)\bescribe.*c[oó]digo`),
	regexp.MustCompile(`(?i)\bcambi.*c[oó]digo`),
	regexp.MustCompile(`(?i)\barregla.*bug`),
	regexp.MustCompile(`(?i)\bcorrige.*error`),
	regexp.MustCompile(`(?i)\boptimiz`),
	regexp.MustCompile(`(?i)\bactualiz`),
	regexp.MustCompile(`(?i)\brenombr`),
	regexp.MustCompile(`(?i)\bmover.*archivo`),
	regexp.MustCompile(`(?i)\bejecut`),
}

func DetectActionIntent(message string) bool {
	lower := strings.ToLower(message)
	for _, re := range actionVerbs {
		if re.MatchString(lower) {
			return true
		}
	}
	return false
}
