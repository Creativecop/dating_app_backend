package provider

import "context"

type OTPProvider interface {
	SendOTP(ctx context.Context, to string, code string, purpose string) (string, error)
}
