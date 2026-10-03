// Package retry provides a reusable retry engine and delay policies.
//  1. An Engine instance created with NewEngine is immutable after initialization
//     and supports concurrent Run calls from multiple goroutines.
//  2. Run does not synchronize user-provided callbacks/policies and does not make them thread-safe.
//  3. If DelayPolicy, IsRetryable, Sleeper, Operation, or OnRetry are shared across goroutines,
//     caller code must provide safe concurrent access to them.
//  4. Request is intended for a single Run call and must not be mutated concurrently.
//  5. In EngineConfig.MaxRetries, a value of 0 means unlimited retries.
//  6. WaitIfRetryable reuses an Engine when the caller owns the attempt loop
//     and must share one retry budget across multiple steps of one operation.
package retry
