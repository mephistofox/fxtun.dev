package database

import (
	"errors"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")

	ErrTokenNotFound = errors.New("token not found")

	ErrSessionNotFound = errors.New("session not found")

	ErrDomainNotFound      = errors.New("domain not found")
	ErrDomainAlreadyExists = errors.New("domain already reserved")
	ErrMaxDomainsReached   = errors.New("maximum domains reached")

	ErrCustomDomainNotFound      = errors.New("custom domain not found")
	ErrCustomDomainAlreadyExists = errors.New("custom domain already exists")

	ErrTOTPNotFound = errors.New("totp secret not found")

	ErrBundleNotFound      = errors.New("bundle not found")
	ErrBundleAlreadyExists = errors.New("bundle already exists")

	ErrHistoryNotFound = errors.New("history entry not found")

	ErrSettingNotFound = errors.New("setting not found")

	ErrPlanNotFound = errors.New("plan not found")
	ErrPlanHasUsers = errors.New("plan has users assigned")

	ErrTLSCertNotFound = errors.New("tls certificate not found")

	ErrEdgeNodeNotFound = errors.New("edge node not found")

	ErrInviteCodeNotFound = errors.New("invite code not found")
)
