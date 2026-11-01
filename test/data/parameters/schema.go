// Package parameters is a test fixture: a channel whose address carries a
// Channel Address Expression, declared with a matching Parameter Object.
package parameters

// UserUpdated is the message emitted for a single user.
type UserUpdated struct {
	UserID string `json:"userId" asyncapi:"required"`
}
