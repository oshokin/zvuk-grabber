package model

import "testing"

// TestFlexibleIDString_NilReceiver verifies String returns an empty value for a nil receiver.
func TestFlexibleIDString_NilReceiver(t *testing.T) {
	t.Parallel()

	var id *FlexibleID
	if got := id.String(); got != "" {
		t.Fatalf("expected empty string for nil FlexibleID, got %q", got)
	}
}
