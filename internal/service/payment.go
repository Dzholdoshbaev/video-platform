package service

import (
	"context"
	"fmt"
)

type Authorizer interface {
	Authorize(ctx context.Context, userID string, amount float64) error
}

type Capturer interface {
	Capture(ctx context.Context, userID string, amount float64) error
}

type PaymentProvider interface {
	Authorizer
	Capturer
}

type FakePaymentProvider struct{}

func (FakePaymentProvider) Authorize(ctx context.Context, _ string, amount float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (FakePaymentProvider) Capture(ctx context.Context, _ string, _ float64) error {
	return ctx.Err()
}

func (FakePaymentProvider) String() string { return "fake-payment-provider" }

var _ fmt.Stringer = FakePaymentProvider{}
