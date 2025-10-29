package cursor

import (
	"encoding/base64"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// CursorData contains the data encoded in a cursor
type CursorData struct {
	// LastID is the ID of the last item in the page
	LastID interface{} `cbor:"1,keyasint"`
	
	// LastSortValue is the sort value of the last item (for sorting)
	LastSortValue interface{} `cbor:"2,keyasint,omitempty"`
	
	// Offset is the offset for random ordering
	Offset int `cbor:"3,keyasint,omitempty"`
	
	// Direction indicates the pagination direction ("next" or "prev")
	Direction string `cbor:"4,keyasint"`
	
	// RandomSeed is the seed for random ordering
	RandomSeed int64 `cbor:"5,keyasint,omitempty"`
}

// Encode encodes cursor data into a base64 string using CBOR
func Encode(data *CursorData) (string, error) {
	if data == nil {
		return "", nil
	}
	
	cborData, err := cbor.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cursor data: %w", err)
	}
	
	return base64.URLEncoding.EncodeToString(cborData), nil
}

// Decode decodes a base64 cursor string into cursor data using CBOR
func Decode(cursor string) (*CursorData, error) {
	if cursor == "" {
		return nil, nil
	}
	
	cborData, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cursor: %w", err)
	}
	
	var data CursorData
	if err := cbor.Unmarshal(cborData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cursor data: %w", err)
	}
	
	return &data, nil
}
