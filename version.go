package praxicraft

// Version is the SDK release version. Keep in sync with git tags (v0.1.0).
const Version = "0.1.0"

const (
	defaultBaseURL   = "https://assess.praxicraft.com"
	defaultAPIPrefix = "/api/v1/public"
	defaultTimeout   = 30 // seconds
	userAgent        = "praxicraft-go/" + Version
)
