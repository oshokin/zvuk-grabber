package auth

import (
	"context"
	"math/rand/v2"

	"github.com/go-rod/rod/lib/proto"

	"github.com/oshokin/zvuk-grabber/internal/logger"
	"github.com/oshokin/zvuk-grabber/internal/utils"
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
		if err = s.page.Mouse.MoveTo(proto.Point{X: float64(x), Y: float64(y)}); err != nil {
			logger.Debugf(ctx, "simulateHumanBehavior mouse move failed: %v", err)
			return
		}

		// Random small delay between movements.
		utils.RandomPause(mouseMovementMinDelay, mouseMovementMaxDelay)
	}

	// Occasionally scroll a bit.
	//nolint:gosec // Weak random is fine for simulating human behavior.
	if rand.IntN(scrollProbability) == 0 {
		//nolint:gosec // Weak random is fine for simulating human behavior.
		scrollAmount := rand.IntN(scrollMaxAmount) + scrollMinAmount
		if err = s.page.Mouse.Scroll(0, float64(scrollAmount), 1); err != nil {
			logger.Debugf(ctx, "simulateHumanBehavior mouse scroll failed: %v", err)
		}
	}
}

// randomHumanDelay sleeps for a random duration to simulate human timing.
func randomHumanDelay() {
	utils.RandomPause(humanBehaviorMinDelay, humanBehaviorMaxDelay)
}

// moveMouseToRandomViewportPosition moves the browser mouse to a random screen position.
func (s *ServiceImpl) moveMouseToRandomViewportPosition(ctx context.Context) {
	eval, err := s.page.Eval(`() => ({width: window.innerWidth, height: window.innerHeight})`)
	if err != nil {
		return
	}

	dims := eval.Value.Map()

	maxX, maxY := int(dims["width"].Num()), int(dims["height"].Num())
	if maxX <= 0 || maxY <= 0 {
		return
	}

	//nolint:gosec // Weak random is fine for simulating human behavior.
	if err = s.page.Mouse.MoveTo(proto.Point{
		X: float64(rand.IntN(maxX)),
		Y: float64(rand.IntN(maxY)),
	}); err != nil {
		logger.Debugf(ctx, "moveMouseToRandomViewportPosition mouse move failed: %v", err)
	}
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
		if err := s.page.Mouse.Scroll(0, scrollDelta, 1); err != nil {
			logger.Debugf(ctx, "simulateRandomPageInteraction mouse scroll failed: %v", err)
		}
	case 2:
		// Pause (humans don't move constantly).
		utils.RandomPause(pauseMinDelay, pauseMaxDelay)
	default:
		// Move mouse cursor to a random visible position.
		s.moveMouseToRandomViewportPosition(ctx)
	}
}
