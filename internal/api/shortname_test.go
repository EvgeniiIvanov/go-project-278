package api

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateShortName(t *testing.T) {
	name, err := generateShortName(8)
	require.NoError(t, err)
	assert.Len(t, name, 8)

	for _, r := range name {
		assert.True(t, unicode.IsLetter(r) || unicode.IsDigit(r), "unexpected rune %q", r)
	}

	// Extremely likely to differ across calls.
	other, err := generateShortName(8)
	require.NoError(t, err)
	assert.NotEqual(t, name, other)
}
