package main

import "testing"

func TestDecideDelivery(t *testing.T) {
	base := AssetAccess{
		CreatorID: "creator-7", AssetID: "preset-pack", Phone: "+15551230000",
		SubscriberVersion: 4, CurrentVersion: 4, ProcessingState: "ready", AttemptID: "login-42",
	}
	tests := []struct {
		name       string
		access     AssetAccess
		verified   bool
		wantState  string
		wantReason string
	}{
		{"verified current subscriber gets processed asset", base, true, "delivery_ready", ""},
		{"unverified phone stays pending", base, false, "delivery_pending", "phone_unverified"},
		{"processing must finish", withState(base, "transcoding"), true, "delivery_pending", "content_processing"},
		{"subscriber update must be current", withSubscriberVersion(base, 3), true, "delivery_pending", "subscriber_update_required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideDelivery(tt.access, tt.verified)
			if got.State != tt.wantState || got.Reason != tt.wantReason || got.AssetID != "preset-pack" {
				t.Fatalf("DecideDelivery() = %+v, want state=%q reason=%q asset=preset-pack", got, tt.wantState, tt.wantReason)
			}
		})
	}
}

func withState(access AssetAccess, state string) AssetAccess {
	access.ProcessingState = state
	return access
}

func withSubscriberVersion(access AssetAccess, version int) AssetAccess {
	access.SubscriberVersion = version
	return access
}
