package zvuk

import (
	"context"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// checkUserSubscription checks the user's subscription status.
func (s *ServiceImpl) checkUserSubscription(ctx context.Context) {
	userProfile, err := s.zvukClient.GetUserProfile(ctx)
	if err != nil {
		logger.Fatalf(ctx, "Failed to retrieve user profile: %v", err)
	}

	if userProfile.Subscription == nil {
		logger.Fatalf(ctx, "User does not have an active subscription")
	}

	expiration := time.UnixMilli(userProfile.Subscription.Expiration).Format(time.RFC1123)
	logger.Infof(ctx, "Active subscription: '%s', expires on %s", userProfile.Subscription.Title, expiration)
}
