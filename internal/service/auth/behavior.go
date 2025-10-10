package auth

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/oshokin/zvuk-grabber/internal/logger"
)

// simulateHumanBehavior performs random mouse movements and scrolling to appear more human-like.
func (s *ServiceImpl) simulateHumanBehavior(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.Debugf(ctx, "simulateHumanBehavior panic recovered: %v", r)
		}
	}()

	// Get page dimensions.
	eval, err := s.page.Eval(`() => ({width: window.innerWidth, height: window.innerHeight})`)
	if err != nil {
		return
	}

	dims := eval.Value.Map()
	maxX := int(dims["width"].Num())
	maxY := int(dims["height"].Num())

	if maxX <= 0 || maxY <= 0 {
		return
	}

	// Perform random mouse movements.
	for range mouseMovementsPerCheck {
		//nolint:gosec // Weak random is fine for simulating human behavior.
		x := rand.IntN(maxX)
		//nolint:gosec // Weak random is fine for simulating human behavior.
		y := rand.IntN(maxY)

		// Move mouse to random position.
		s.page.Mouse.MustMoveTo(float64(x), float64(y))

		// Random small delay between movements.
		delayRange := int(mouseMovementMaxDelay - mouseMovementMinDelay)
		//nolint:gosec // Weak random is fine for simulating human behavior.
		time.Sleep(time.Duration(rand.IntN(delayRange)) + mouseMovementMinDelay)
	}

	// Occasionally scroll a bit.
	//nolint:gosec // Weak random is fine for simulating human behavior.
	if rand.IntN(scrollProbability) == 0 {
		//nolint:gosec // Weak random is fine for simulating human behavior.
		scrollAmount := rand.IntN(scrollMaxAmount) + scrollMinAmount
		s.page.Mouse.MustScroll(0, float64(scrollAmount))
	}
}

// randomHumanDelay sleeps for a random duration to simulate human timing.
func randomHumanDelay() {
	//nolint:gosec // Weak random is fine for simulating human behavior.
	delay := time.Duration(rand.Int64N(int64(humanBehaviorMaxDelay-humanBehaviorMinDelay))) + humanBehaviorMinDelay
	time.Sleep(delay)
}

// simulateRandomPageInteraction performs random, harmless page interactions.
func (s *ServiceImpl) simulateRandomPageInteraction(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.Debugf(ctx, "simulateRandomPageInteraction panic recovered: %v", r)
		}
	}()

	//nolint:gosec // Weak random is fine for simulating human behavior.
	action := rand.IntN(interactionActionCount)

	switch action {
	case 0:
		// Small random scroll.
		//nolint:gosec // Weak random is fine for simulating human behavior.
		scrollDelta := float64(rand.IntN(smallScrollRange) - smallScrollOffset)
		s.page.Mouse.MustScroll(0, scrollDelta)
	case 1:
		// Move mouse cursor slightly from current position.
		eval, err := s.page.Eval(`() => ({width: window.innerWidth, height: window.innerHeight})`)
		if err == nil {
			dims := eval.Value.Map()
			//nolint:gosec // Weak random is fine for simulating human behavior.
			newX := float64(rand.IntN(int(dims["width"].Num())))
			//nolint:gosec // Weak random is fine for simulating human behavior.
			newY := float64(rand.IntN(int(dims["height"].Num())))
			s.page.Mouse.MustMoveTo(newX, newY)
		}
	case 2:
		// Pause (humans don't move constantly).
		pauseRange := int(pauseMaxDelay - pauseMinDelay)
		//nolint:gosec // Weak random is fine for simulating human behavior.
		time.Sleep(time.Duration(rand.IntN(pauseRange)) + pauseMinDelay)
	default:
		// Very small random movement.
		eval, err := s.page.Eval(`() => ({width: window.innerWidth, height: window.innerHeight})`)
		if err == nil {
			dims := eval.Value.Map()
			//nolint:gosec // Weak random is fine for simulating human behavior.
			x := float64(rand.IntN(int(dims["width"].Num())))
			//nolint:gosec // Weak random is fine for simulating human behavior.
			y := float64(rand.IntN(int(dims["height"].Num())))
			s.page.Mouse.MustMoveTo(x, y)
		}
	}
}
