// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package authn_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/authn"
	"github.com/mainflux/mainflux/authn/jwt"
	"github.com/mainflux/mainflux/authn/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	secret = "secret"
	email  = "test@example.com"
)

func newService() authn.Service {
	repo := mocks.NewKeyRepository()
	idp := mocks.NewIdentityProvider()
	t := jwt.New(secret)
	return authn.New(repo, idp, t)
}

func TestIssue(t *testing.T) {
	svc := newService()
	loginKey, err := svc.Issue(context.Background(), email, authn.Key{Type: authn.UserKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	cases := []struct {
		desc   string
		key    authn.Key
		issuer string
		err    error
	}{
		{
			desc: "issue login key",
			key: authn.Key{
				Type:     authn.UserKey,
				IssuedAt: time.Now(),
			},
			issuer: email,
			err:    nil,
		},
		{
			desc: "issue login key no issue time",
			key: authn.Key{
				Type: authn.UserKey,
			},
			issuer: email,
			err:    authn.ErrInvalidKeyIssuedAt,
		},
		{
			desc: "issue user key",
			key: authn.Key{
				Type:     authn.APIKey,
				IssuedAt: time.Now(),
			},
			issuer: loginKey.Secret,
			err:    nil,
		},
		{
			desc: "issue user key unauthorized",
			key: authn.Key{
				Type:     authn.APIKey,
				IssuedAt: time.Now(),
			},
			issuer: "",
			err:    authn.ErrUnauthorizedAccess,
		},
		{
			desc: "issue user key no issue time",
			key: authn.Key{
				Type: authn.APIKey,
			},
			issuer: loginKey.Secret,
			err:    authn.ErrInvalidKeyIssuedAt,
		},
		{
			desc: "issue reset key",
			key: authn.Key{
				Type:     authn.RecoveryKey,
				IssuedAt: time.Now(),
			},
			issuer: loginKey.Secret,
			err:    nil,
		},
		{
			desc: "issue reset key no issue time",
			key: authn.Key{
				Type: authn.RecoveryKey,
			},
			issuer: loginKey.Secret,
			err:    authn.ErrInvalidKeyIssuedAt,
		},
	}

	for _, tc := range cases {
		_, err := svc.Issue(context.Background(), tc.issuer, tc.key)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}
func TestRevoke(t *testing.T) {
	svc := newService()
	loginKey, err := svc.Issue(context.Background(), email, authn.Key{Type: authn.UserKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := authn.Key{
		Type:     authn.APIKey,
		IssuedAt: time.Now(),
	}
	newKey, err := svc.Issue(context.Background(), loginKey.Secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	cases := []struct {
		desc   string
		id     string
		issuer string
		err    error
	}{
		{
			desc:   "revoke user key",
			id:     newKey.ID,
			issuer: loginKey.Secret,
			err:    nil,
		},
		{
			desc:   "revoke non-existing user key",
			id:     newKey.ID,
			issuer: loginKey.Secret,
			err:    nil,
		},
		{
			desc:   "revoke unauthorized",
			id:     newKey.ID,
			issuer: "",
			err:    authn.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		err := svc.Revoke(context.Background(), tc.issuer, tc.id)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}
func TestRetrieve(t *testing.T) {
	svc := newService()
	loginKey, err := svc.Issue(context.Background(), email, authn.Key{Type: authn.UserKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := authn.Key{
		ID:       "id",
		Type:     authn.APIKey,
		IssuedAt: time.Now(),
	}
	newKey, err := svc.Issue(context.Background(), loginKey.Secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	resetKey, err := svc.Issue(context.Background(), loginKey.Secret, authn.Key{Type: authn.RecoveryKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	userKey, err := svc.Issue(context.Background(), loginKey.Secret, authn.Key{Type: authn.APIKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	cases := []struct {
		desc   string
		id     string
		issuer string
		err    error
	}{
		{
			desc:   "retrieve user key",
			id:     newKey.ID,
			issuer: loginKey.Secret,
			err:    nil,
		},
		{
			desc:   "retrieve non-existing user key",
			id:     "invalid",
			issuer: loginKey.Secret,
			err:    authn.ErrNotFound,
		},
		{
			desc:   "retrieve unauthorized",
			id:     newKey.ID,
			issuer: "wrong",
			err:    authn.ErrUnauthorizedAccess,
		},
		{
			desc:   "retrieve with user key",
			id:     newKey.ID,
			issuer: userKey.Secret,
			err:    authn.ErrUnauthorizedAccess,
		},
		{
			desc:   "retrieve with reset key",
			id:     newKey.ID,
			issuer: resetKey.Secret,
			err:    authn.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		_, err := svc.Retrieve(context.Background(), tc.issuer, tc.id)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}
func TestIdentify(t *testing.T) {
	svc := newService()
	loginKey, err := svc.Issue(context.Background(), email, authn.Key{Type: authn.UserKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	resetKey, err := svc.Issue(context.Background(), loginKey.Secret, authn.Key{Type: authn.RecoveryKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	userKey, err := svc.Issue(context.Background(), loginKey.Secret, authn.Key{Type: authn.APIKey, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	exp1 := time.Now().Add(-2 * time.Second)
	expKey, err := svc.Issue(context.Background(), loginKey.Secret, authn.Key{Type: authn.APIKey, IssuedAt: time.Now(), ExpiresAt: exp1})
	assert.Nil(t, err, fmt.Sprintf("Issuing expired user key expected to succeed: %s", err))

	invalidKey, err := svc.Issue(context.Background(), loginKey.Secret, authn.Key{Type: 22, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	cases := []struct {
		desc string
		key  string
		id   string
		err  error
	}{
		{
			desc: "identify login key",
			key:  loginKey.Secret,
			id:   email,
			err:  nil,
		},
		{
			desc: "identify reset key",
			key:  resetKey.Secret,
			id:   email,
			err:  nil,
		},
		{
			desc: "identify user key",
			key:  userKey.Secret,
			id:   email,
			err:  nil,
		},
		{
			desc: "identify expired user key",
			key:  expKey.Secret,
			id:   "",
			err:  authn.ErrKeyExpired,
		},
		{
			desc: "identify expired key",
			key:  invalidKey.Secret,
			id:   "",
			err:  authn.ErrUnauthorizedAccess,
		},
		{
			desc: "identify invalid key",
			key:  "invalid",
			id:   "",
			err:  authn.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		id, err := svc.Identify(context.Background(), tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.id, id))
	}
}
