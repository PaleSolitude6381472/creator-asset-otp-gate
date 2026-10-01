package main

import (
	"context"
	"fmt"
)

type OTPVerifier interface {
	Issue(context.Context, string, string) (otpReply, error)
	Verify(context.Context, string, string, string) (verifyReply, error)
}

type AssetAccess struct {
	CreatorID         string `json:"creator_id"`
	Phone             string `json:"phone"`
	AssetID           string `json:"asset_id"`
	SubscriberVersion int    `json:"subscriber_version"`
	CurrentVersion    int    `json:"current_version"`
	ProcessingState   string `json:"processing_state"`
	AttemptID         string `json:"attempt_id"`
}

type DeliveryDecision struct {
	State   string `json:"state"`
	AssetID string `json:"asset_id"`
	Reason  string `json:"reason,omitempty"`
}

type AccessService struct {
	SMS OTPVerifier
}

func (s AccessService) Start(ctx context.Context, access AssetAccess) (otpReply, error) {
	if err := validateAccess(access); err != nil {
		return otpReply{}, err
	}
	return s.SMS.Issue(ctx, access.Phone, access.AttemptID+":issue")
}

func (s AccessService) Verify(ctx context.Context, access AssetAccess, code string) (DeliveryDecision, error) {
	if err := validateAccess(access); err != nil {
		return DeliveryDecision{}, err
	}
	if code == "" {
		return DeliveryDecision{}, fmt.Errorf("code is required")
	}
	result, err := s.SMS.Verify(ctx, access.Phone, code, access.AttemptID+":verify")
	if err != nil {
		return DeliveryDecision{}, err
	}
	return DecideDelivery(access, result.Verified), nil
}

func DecideDelivery(access AssetAccess, phoneVerified bool) DeliveryDecision {
	decision := DeliveryDecision{State: "delivery_pending", AssetID: access.AssetID}
	switch {
	case !phoneVerified:
		decision.Reason = "phone_unverified"
	case access.ProcessingState != "ready":
		decision.Reason = "content_processing"
	case access.SubscriberVersion != access.CurrentVersion:
		decision.Reason = "subscriber_update_required"
	default:
		decision.State = "delivery_ready"
	}
	return decision
}

func validateAccess(access AssetAccess) error {
	if access.CreatorID == "" || access.Phone == "" || access.AssetID == "" || access.AttemptID == "" {
		return fmt.Errorf("creator_id, phone, asset_id, and attempt_id are required")
	}
	return nil
}
