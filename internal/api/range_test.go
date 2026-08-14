package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRangeQuery(t *testing.T) {
	t.Run("default when empty", func(t *testing.T) {
		from, to, err := parseRangeQuery("")
		require.NoError(t, err)
		assert.Equal(t, defaultRangeFrom, from)
		assert.Equal(t, defaultRangeTo, to)
	})

	t.Run("hexlet large window with space", func(t *testing.T) {
		from, to, err := parseRangeQuery("[0, 1000]")
		require.NoError(t, err)
		assert.Equal(t, int32(0), from)
		assert.Equal(t, int32(1000), to)
	})

	t.Run("without brackets", func(t *testing.T) {
		from, to, err := parseRangeQuery("2,5")
		require.NoError(t, err)
		assert.Equal(t, int32(2), from)
		assert.Equal(t, int32(5), to)
	})

	t.Run("invalid order", func(t *testing.T) {
		_, _, err := parseRangeQuery("[5,1]")
		assert.Error(t, err)
	})

	t.Run("too large window still rejected", func(t *testing.T) {
		_, _, err := parseRangeQuery("[0, 2000]")
		assert.Error(t, err)
	})
}
