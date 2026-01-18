package project

// CreateProjectRequest is the request to create a new project.
type CreateProjectRequest struct {
	Name        string `json:"name"`                  // Required: Project name
	Description string `json:"description,omitempty"` // Optional: Project description
}

// GetProjectRequest is the request to get a project by ID.
type GetProjectRequest struct {
	ID string `json:"id"` // Required: Project ID
}

// ListProjectsRequest is the request to list projects for the authenticated user.
type ListProjectsRequest struct {
	// Empty for now - could add pagination in the future
}

// UpdateProjectRequest is the request to update a project.
// Only provided fields will be updated (partial update).
type UpdateProjectRequest struct {
	ID          string  `json:"id"`                    // Required: Project ID
	Name        *string `json:"name,omitempty"`        // Optional: New name
	Description *string `json:"description,omitempty"` // Optional: New description
}

// DeleteProjectRequest is the request to delete a project.
type DeleteProjectRequest struct {
	ID string `json:"id"` // Required: Project ID
}

// ProjectResponse is the response for a single project.
// Timestamps are formatted as RFC3339 strings.
type ProjectResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     string `json:"owner_id"`
	CreatedAt   string `json:"created_at"` // RFC3339 format
	UpdatedAt   string `json:"updated_at"` // RFC3339 format
}

// ListProjectsResponse is the response for listing projects.
type ListProjectsResponse struct {
	Projects []ProjectResponse `json:"projects"` // List of projects
	Total    int               `json:"total"`    // Total count
}
