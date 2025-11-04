package ditto

import (
	"encoding/json"
	"fmt"
)

// AttachmentToken represents a lightweight reference to an attachment.
// Tokens are stored in documents to reference attachments without including
// the actual binary data. When a document with an attachment token is synced,
// the receiving peer can use the token to fetch the actual attachment data.
//
// Tokens are immutable once created and safe for concurrent access.
//
// Example usage:
//
//	// Create a token from an attachment
//	token := Token()
//
//	// Store the token in a document
//	doc := map[string]any{
//		"image": token,
//		"description": "Profile picture",
//	}
//
//	// Later, deserialize a token from a document
//	if tokenData, ok := doc["image"].([]byte); ok {
//		token, err := AttachmentTokenFromBytes(tokenData)
//		if err == nil {
//			// Use token to fetch the attachment
//		}
//	}
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type AttachmentToken struct {
	id       string
	len      int64
	metadata map[string]any
}

// NewAttachmentToken creates a new attachment token.
//
// Parameters:
//   - id: The unique identifier of the attachment (hex-encoded string)
//   - length: The size of the attachment in bytes
//   - metadata: Optional metadata associated with the attachment
//
// Returns:
//   - *AttachmentToken: The created token
//
// Tokens are typically created automatically when creating attachments,
// but this function allows manual token creation when needed.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func NewAttachmentToken(id string, length int64, metadata map[string]any) *AttachmentToken {
	return &AttachmentToken{
		id:       id,
		len:      length,
		metadata: metadata,
	}
}

// ID returns the unique identifier of the referenced
// This ID is used to fetch the actual attachment data from the Ditto network.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *AttachmentToken) ID() string {
	return t.id
}

// Length returns the size of the referenced attachment in bytes.
// This can be used to display file sizes or estimate download times
// before actually fetching the attachment.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *AttachmentToken) Length() int64 {
	return t.len
}

// Metadata returns the metadata associated with the referenced attachment.
// This might include information like content type, original filename,
// creation date, or any custom attributes defined when the attachment was created.
//
// The returned map is a reference to the internal metadata. Modifications
// will affect the token's metadata.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *AttachmentToken) Metadata() map[string]any {
	return t.metadata
}

// String returns a JSON string representation of the token.
// This is useful for debugging or logging purposes.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *AttachmentToken) String() string {
	data, _ := json.Marshal(map[string]any{
		"id":       t.id,
		"len":      t.len,
		"metadata": t.metadata,
	})
	return string(data)
}

// ToBytes serializes the token to a byte array.
// This is the format used when storing tokens in documents.
//
// Returns:
//   - []byte: The serialized token data
//   - error: An error if serialization fails
//
// The returned bytes can be stored in a document and later
// deserialized using AttachmentTokenFromBytes.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *AttachmentToken) ToBytes() ([]byte, error) {
	// Use explicit struct for marshaling since fields are unexported
	data := map[string]any{
		"id":       t.id,
		"len":      t.len,
		"metadata": t.metadata,
	}
	return json.Marshal(data)
}

// AttachmentTokenFromBytes deserializes a token from a byte array.
//
// Parameters:
//   - data: The serialized token data, typically retrieved from a document
//
// Returns:
//   - *AttachmentToken: The deserialized token
//   - error: An error if the data is invalid or deserialization fails
//
// This function is used to reconstruct tokens from data stored in documents,
// allowing you to fetch the referenced attachments.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func AttachmentTokenFromBytes(data []byte) (*AttachmentToken, error) {
	if len(data) == 0 {
		// Return error for empty data
		return nil, fmt.Errorf("empty attachment token data")
	}

	// Parse JSON into map since fields are unexported
	var tokenData map[string]any
	err := json.Unmarshal(data, &tokenData)
	if err != nil {
		return nil, err
	}

	// Extract fields from map
	id, _ := tokenData["id"].(string)
	length, _ := tokenData["len"].(float64) // JSON numbers are float64
	metadata, _ := tokenData["metadata"].(map[string]any)

	return &AttachmentToken{
		id:       id,
		len:      int64(length),
		metadata: metadata,
	}, nil
}

// FromString creates a token from its JSON string representation.
//
// Parameters:
//   - s: A JSON string representing the token, typically from String()
//
// Returns:
//   - *AttachmentToken: The deserialized token
//   - error: An error if the string is invalid JSON or deserialization fails
//
// This is useful for recreating tokens from logged or transmitted string data.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func FromString(s string) (*AttachmentToken, error) {
	// Parse JSON string to create token
	return AttachmentTokenFromBytes([]byte(s))
}

// AttachmentTokenLike is an interface for types that can be converted to attachment tokens.
// This allows for flexible token handling where different types can provide token representations.
//
// Implement this interface on custom types that need to be used as attachment references.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type AttachmentTokenLike interface {
	// ToToken converts the implementing type to a AttachmentToken
	ToToken() *AttachmentToken
}
