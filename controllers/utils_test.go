package controllers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuffixLimit(t *testing.T) {
	assert.Equal(t, "", suffixLimit("", ""))
	assert.Equal(t, "a-0", suffixLimit("a", "-0"))
	assert.Len(t, suffixLimit(strings.Repeat("a", 100), "-0"), 63)
	assert.True(t, strings.HasSuffix(suffixLimit(strings.Repeat("a", 100), "-0"), "-0"))
}
