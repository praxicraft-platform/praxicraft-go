package praxicraft

// Shared Public API response shapes. Extra JSON fields are ignored on decode.
// Use these for typed resource methods; Client.Do still returns decoded any for
// escape-hatch / forward-compatible access.

// Org is the workspace summary from GET /org/.
type Org struct {
	ID               string  `json:"id,omitempty"`
	Name             string  `json:"name,omitempty"`
	Slug             string  `json:"slug,omitempty"`
	Plan             string  `json:"plan,omitempty"`
	InvitesRemaining *int    `json:"invites_remaining,omitempty"`
	InvitesUsed      *int    `json:"invites_used,omitempty"`
	InvitesLimit     *int    `json:"invites_limit,omitempty"`
}

// Assessment is a single assessment object.
type Assessment struct {
	ID     string `json:"id,omitempty"`
	Slug   string `json:"slug,omitempty"`
	Title  string `json:"title,omitempty"`
	Status string `json:"status,omitempty"`
}

// Invite is an invitation object. Create responses include invite_token (not "token").
type Invite struct {
	ID          string `json:"id,omitempty"`
	InviteToken string `json:"invite_token,omitempty"`
	InviteURL   string `json:"invite_url,omitempty"`
	Email       string `json:"email,omitempty"`
	Name        string `json:"name,omitempty"`
	Status      string `json:"status,omitempty"`
	Assessment  string `json:"assessment,omitempty"`
}

// ResultRow is one cohort / candidate result row.
type ResultRow struct {
	InviteToken string  `json:"invite_token,omitempty"`
	Email       string  `json:"email,omitempty"`
	Name        string  `json:"name,omitempty"`
	Status      string  `json:"status,omitempty"`
	Score       *float64 `json:"score,omitempty"`
	Passed      *bool   `json:"passed,omitempty"`
}

// WebhookEndpoint is a registered webhook.
type WebhookEndpoint struct {
	ID        string   `json:"id,omitempty"`
	URL       string   `json:"url,omitempty"`
	Events    []string `json:"events,omitempty"`
	IsActive  *bool    `json:"is_active,omitempty"`
	SecretKey string   `json:"secret_key,omitempty"`
}

// Pipeline is a hiring pipeline summary.
type Pipeline struct {
	ID    string `json:"id,omitempty"`
	Slug  string `json:"slug,omitempty"`
	Name  string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

// Enrollment is a pipeline enrollment.
type Enrollment struct {
	EnrollmentID string `json:"enrollment_id,omitempty"`
	ID           string `json:"id,omitempty"`
	Email        string `json:"email,omitempty"`
	Name         string `json:"name,omitempty"`
	Status       string `json:"status,omitempty"`
}

// Page is a cursor-paginated list envelope.
type Page[T any] struct {
	Results    []T    `json:"results"`
	Next       string `json:"next,omitempty"`
	Previous   string `json:"previous,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	Count      *int   `json:"count,omitempty"`
}
