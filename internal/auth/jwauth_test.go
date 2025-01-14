//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// package auth contains the authentication logic for the control plane
package auth

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mockjwt "github.com/stacklok/minder/internal/auth/mock"
	"github.com/stacklok/minder/internal/util/rand"
)

func TestParseAndValidate(t *testing.T) {
	t.Parallel()

	jwks := jwk.NewSet()
	privateKey, publicKey := rand.RandomKeypair(2048)
	privateJwk, _ := jwk.FromRaw(privateKey)
	err := privateJwk.Set(jwk.KeyIDKey, `mykey`)
	require.NoError(t, err, "failed to setup private key ID")

	publicJwk, _ := jwk.FromRaw(publicKey)
	err = publicJwk.Set(jwk.KeyIDKey, "mykey")
	require.NoError(t, err, "failed to setup public key ID")
	err = publicJwk.Set(jwk.AlgorithmKey, jwa.RS256)
	require.NoError(t, err, "failed to setup public key algorithm")

	err = jwks.AddKey(publicJwk)
	require.NoError(t, err, "failed to setup JWK set")

	testCases := []struct {
		name       string
		buildToken func() string
		checkError func(t *testing.T, err error)
	}{
		{
			name: "Valid token",
			buildToken: func() string {
				token, _ := jwt.NewBuilder().Subject("123").Expiration(time.Now().Add(time.Duration(1) * time.Minute)).Build()
				signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, privateJwk))
				return string(signed)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()

				assert.NoError(t, err)
			},
		},
		{
			name: "Expired token",
			buildToken: func() string {
				token, _ := jwt.NewBuilder().Subject("123").Expiration(time.Now().Add(-time.Duration(1) * time.Minute)).Build()
				signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, privateJwk))
				return string(signed)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()

				assert.Error(t, err)
			},
		},
		{
			name: "Invalid signature",
			buildToken: func() string {
				otherKey, _ := rand.RandomKeypair(2048)
				otherJwk, _ := jwk.FromRaw(otherKey)
				err = otherJwk.Set(jwk.KeyIDKey, `otherKey`)
				require.NoError(t, err, "failed to setup signing key ID")
				token, _ := jwt.NewBuilder().Subject("123").Expiration(time.Now().Add(time.Duration(1) * time.Minute)).Build()
				signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, otherJwk))
				return string(signed)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()

				assert.Error(t, err)
			},
		},
		{
			name: "Invalid token",
			buildToken: func() string {
				return "invalid"
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()

				assert.Error(t, err)
			},
		},
		{
			name: "Missing subject claim",
			buildToken: func() string {
				token, _ := jwt.NewBuilder().Expiration(time.Now().Add(-time.Duration(1) * time.Minute)).Build()
				signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, privateJwk))
				return string(signed)
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()

				assert.Error(t, err)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockKeyFetcher := mockjwt.NewMockKeySetFetcher(ctrl)
			mockKeyFetcher.EXPECT().GetKeySet().Return(jwks, nil)

			jwtValidator := JwkSetJwtValidator{jwksFetcher: mockKeyFetcher}
			_, err := jwtValidator.ParseAndValidate(tc.buildToken())
			tc.checkError(t, err)
		})
	}
}
