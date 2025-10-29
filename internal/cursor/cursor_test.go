package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursor_EncodeAndDecode(t *testing.T) {
	tests := []struct {
		name string
		data *CursorData
	}{
		{
			name: "simple cursor",
			data: &CursorData{
				LastID:    "507f1f77bcf86cd799439011",
				Direction: "next",
			},
		},
		{
			name: "cursor with sort value",
			data: &CursorData{
				LastID:        "507f1f77bcf86cd799439011",
				LastSortValue: "2020-01-03",
				Direction:     "next",
			},
		},
		{
			name: "cursor with offset",
			data: &CursorData{
				LastID:    "507f1f77bcf86cd799439011",
				Offset:    50,
				Direction: "next",
			},
		},
		{
			name: "cursor with random seed",
			data: &CursorData{
				LastID:     "507f1f77bcf86cd799439011",
				Offset:     20,
				RandomSeed: 12345678,
				Direction:  "next",
			},
		},
		{
			name: "previous page cursor",
			data: &CursorData{
				LastID:    "507f1f77bcf86cd799439011",
				Direction: "prev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := Encode(tt.data)
			require.NoError(t, err)
			require.NotEmpty(t, encoded)

			// Decode
			decoded, err := Decode(encoded)
			require.NoError(t, err)
			require.NotNil(t, decoded)

			// Compare
			assert.Equal(t, tt.data.LastID, decoded.LastID)
			assert.Equal(t, tt.data.Direction, decoded.Direction)
			assert.Equal(t, tt.data.Offset, decoded.Offset)
			assert.Equal(t, tt.data.RandomSeed, decoded.RandomSeed)
		})
	}
}

func TestCursor_EncodeNil(t *testing.T) {
	encoded, err := Encode(nil)
	require.NoError(t, err)
	assert.Empty(t, encoded)
}

func TestCursor_DecodeEmpty(t *testing.T) {
	decoded, err := Decode("")
	require.NoError(t, err)
	assert.Nil(t, decoded)
}

func TestCursor_DecodeInvalid(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{
			name:   "invalid base64",
			cursor: "not-valid-base64!@#$",
		},
		{
			name:   "valid base64 but invalid JSON",
			cursor: "aGVsbG8gd29ybGQ=", // "hello world" in base64
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := Decode(tt.cursor)
			assert.Error(t, err)
			assert.Nil(t, decoded)
		})
	}
}

func TestCursor_RoundTrip(t *testing.T) {
	original := &CursorData{
		LastID:        "test-id-123",
		LastSortValue: "test-value",
		Offset:        100,
		Direction:     "next",
		RandomSeed:    int64(987654321),
	}
	
	// Encode
	encoded, err := Encode(original)
	require.NoError(t, err)
	
	// Decode
	decoded, err := Decode(encoded)
	require.NoError(t, err)
	
	// Verify all fields
	assert.Equal(t, original.LastID, decoded.LastID)
	assert.Equal(t, original.LastSortValue, decoded.LastSortValue)
	assert.Equal(t, original.Offset, decoded.Offset)
	assert.Equal(t, original.Direction, decoded.Direction)
	assert.Equal(t, original.RandomSeed, decoded.RandomSeed)
}
