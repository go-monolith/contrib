package project

import "time"

// Project represents a project entity owned by a user.
// Projects are the main resource managed by this application.
type Project struct {
	ID          string    `json:"id"`          // Unique identifier (UUID)
	Name        string    `json:"name"`        // Project name (1-100 characters, unique per owner)
	Description string    `json:"description"` // Project description (max 500 characters)
	OwnerID     string    `json:"owner_id"`    // User ID from JWT sub claim (immutable)
	CreatedAt   time.Time `json:"created_at"`  // Creation timestamp
	UpdatedAt   time.Time `json:"updated_at"`  // Last update timestamp
}
