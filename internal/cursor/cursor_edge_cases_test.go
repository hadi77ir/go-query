package cursor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursor_EdgeCases(t *testing.T) {
	t.Run("nil cursor data", func(t *testing.T) {
		encoded, err := Encode(nil)
		require.NoError(t, err)
		assert.Empty(t, encoded)
	})

	t.Run("empty cursor string", func(t *testing.T) {
		decoded, err := Decode("")
		require.NoError(t, err)
		assert.Nil(t, decoded)
	})

	t.Run("whitespace cursor", func(t *testing.T) {
		_, err := Decode("   ")
		assert.Error(t, err) // Should fail to decode
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := Decode("not-valid-base64!@#$%^&*()")
		assert.Error(t, err)
	})

	t.Run("valid base64 but invalid CBOR", func(t *testing.T) {
		// "hello world" in base64
		_, err := Decode("aGVsbG8gd29ybGQ=")
		assert.Error(t, err)
	})

	t.Run("cursor with nil LastID", func(t *testing.T) {
		data := &CursorData{
			LastID:    nil,
			Direction: "next",
		}
		encoded, err := Encode(data)
		require.NoError(t, err)
		
		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.Nil(t, decoded.LastID)
	})

	t.Run("cursor with zero offset", func(t *testing.T) {
		data := &CursorData{
			LastID:    "test",
			Offset:    0,
			Direction: "next",
		}
		encoded, err := Encode(data)
		require.NoError(t, err)
		
		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, 0, decoded.Offset)
	})

	t.Run("cursor with negative offset", func(t *testing.T) {
		data := &CursorData{
			LastID:    "test",
			Offset:    -10,
			Direction: "prev",
		}
		encoded, err := Encode(data)
		require.NoError(t, err)
		
		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, -10, decoded.Offset)
	})

	t.Run("cursor with very large offset", func(t *testing.T) {
		data := &CursorData{
			LastID:    "test",
			Offset:    999999999,
			Direction: "next",
		}
		encoded, err := Encode(data)
		require.NoError(t, err)
		
		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, 999999999, decoded.Offset)
	})

	t.Run("cursor with empty direction", func(t *testing.T) {
		data := &CursorData{
			LastID:    "test",
			Direction: "",
		}
		encoded, err := Encode(data)
		require.NoError(t, err)
		
		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, "", decoded.Direction)
	})

	t.Run("cursor with complex LastID", func(t *testing.T) {
		data := &CursorData{
			LastID:    map[string]interface{}{"id": 123, "name": "test"},
			Direction: "next",
		}
		encoded, err := Encode(data)
		require.NoError(t, err)
		
		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.NotNil(t, decoded.LastID)
	})

	t.Run("cursor with unicode in LastID", func(t *testing.T) {
		data := &CursorData{
			LastID:    "日本語-テスト-🎉",
			Direction: "next",
		}
		encoded, err := Encode(data)
		require.NoError(t, err)
		
		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, "日本語-テスト-🎉", decoded.LastID)
	})

	t.Run("very long cursor", func(t *testing.T) {
		data := &CursorData{
			LastID:        strings.Repeat("a", 10000),
			LastSortValue: strings.Repeat("b", 10000),
			Direction:     "next",
		}
		encoded, err := Encode(data)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)
		
		decoded, err := Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, strings.Repeat("a", 10000), decoded.LastID)
	})

	t.Run("cursor size comparison", func(t *testing.T) {
		data := &CursorData{
			LastID:        "507f1f77bcf86cd799439011",
			LastSortValue: "test-value",
			Offset:        100,
			Direction:     "next",
			RandomSeed:    12345,
		}
		
		encoded, err := Encode(data)
		require.NoError(t, err)
		
		// CBOR should be more compact than the raw data
		assert.Less(t, len(encoded), 200, "cursor should be reasonably sized")
	})
}

func TestCursor_Tampering(t *testing.T) {
	t.Run("modified cursor", func(t *testing.T) {
		data := &CursorData{
			LastID:    "test",
			Direction: "next",
		}
		encoded, _ := Encode(data)
		
		// Modify the cursor
		if len(encoded) > 10 {
			modified := encoded[:len(encoded)-5] + "XXXXX"
			_, err := Decode(modified)
			assert.Error(t, err, "modified cursor should fail to decode")
		}
	})

	t.Run("truncated cursor", func(t *testing.T) {
		data := &CursorData{
			LastID:    "test",
			Direction: "next",
		}
		encoded, _ := Encode(data)
		
		if len(encoded) > 5 {
			truncated := encoded[:len(encoded)/2]
			_, err := Decode(truncated)
			assert.Error(t, err, "truncated cursor should fail")
		}
	})
}

