package retry

import "context"

// DoOption defines extra configuration for a one-shot run via Do.
type DoOption func(req *Request)

// WithOnRetry adds a callback invoked before waiting and
// running the next retry attempt.
func WithOnRetry(onRetry OnRetry) DoOption {
	return func(req *Request) {
		req.OnRetry = onRetry
	}
}

// Do executes an operation with retries based on the provided configuration.
// A new Engine is created for each call.
// For frequently called code, prefer creating one Engine with
// NewEngine and then invoking Run.
func Do(
	ctx context.Context,
	cfg *EngineConfig,
	operation func(context.Context) error,
	opts ...DoOption,
) error {
	engine, err := NewEngine(cfg)
	if err != nil {
		return err
	}

	req := &Request{
		Operation: operation,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(req)
	}

	return engine.Run(ctx, req)
}
