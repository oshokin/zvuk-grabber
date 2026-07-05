package zvuk

// UserProfile represents a user's profile information.
type UserProfile struct {
	// Subscription contains the user's subscription details.
	Subscription *UserSubscription `json:"subscription"`
}

// UserSubscription represents a user's subscription details.
type UserSubscription struct {
	// Title is the subscription plan name.
	Title string `json:"title"`
	// Expiration is the subscription expiration timestamp.
	Expiration int64 `json:"expiration"`
}
