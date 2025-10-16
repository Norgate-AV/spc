package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"2", []string{"series2"}},
		{"3", []string{"series3"}},
		{"4", []string{"series4"}},
		{"23", []string{"series2", "series3"}},
		{"34", []string{"series3", "series4"}},
		{"234", []string{"series2", "series3", "series4"}},
		{"", []string{}},
		{"5", []string{}},
		{"13", []string{"series3"}},
	}

	for _, test := range tests {
		result := ParseTarget(test.input)
		assert.Equal(t, test.expected, result, "ParseTarget(%q)", test.input)
	}
}
