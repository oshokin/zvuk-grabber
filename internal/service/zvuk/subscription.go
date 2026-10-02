package zvuk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

var errNoSubscription = errors.New("user does not have an active subscription")

// checkUserSubscription checks the user's subscription status.
func (s *ServiceImpl) checkUserSubscription(ctx context.Context) error {
	userProfile, err := s.zvukClient.GetUserProfile(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve user profile: %w", err)
	}

	if userProfile == nil || userProfile.Subscription == nil {
		return errNoSubscription
	}

	expiration := time.UnixMilli(userProfile.Subscription.Expiration).Format(time.RFC1123)
	logger.Infof(ctx, "Active subscription: '%s', expires on %s", userProfile.Subscription.Title, expiration)

	return nil
}
