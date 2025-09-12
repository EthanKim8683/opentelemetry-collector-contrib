// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"crypto/tls"
	"os"
	"testing"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createNkey(t *testing.T) nkeys.KeyPair {
	keyPair, err := nkeys.CreateUser()
	require.NoError(t, err)
	t.Cleanup(func() {
		keyPair.Wipe()
	})

	return keyPair
}

func createNkeyJWT(t *testing.T) (string, nkeys.KeyPair) {
	userKeyPair := createNkey(t)
	userPubKey, err := userKeyPair.PublicKey()
	require.NoError(t, err)

	accountKeyPair, err := nkeys.CreateAccount()
	require.NoError(t, err)
	t.Cleanup(func() {
		accountKeyPair.Wipe()
	})
	accountPubKey, err := accountKeyPair.PublicKey()
	require.NoError(t, err)

	userJWT, err := jwt.IssueUserJWT(
		accountKeyPair,
		accountPubKey,
		userPubKey,
		"",
		5*time.Minute,
	)
	require.NoError(t, err)

	return userJWT, userKeyPair
}

func createNkeyUserFile(t *testing.T) (string, nkeys.KeyPair, string) {
	userJWT, userKeyPair := createNkeyJWT(t)
	userSeed, err := userKeyPair.Seed()
	require.NoError(t, err)

	userConfig, err := jwt.FormatUserConfig(userJWT, userSeed)
	require.NoError(t, err)

	userFile, err := os.CreateTemp("", "")
	require.NoError(t, err)
	userFilePath := userFile.Name()
	t.Cleanup(func() {
		os.Remove(userFilePath)
	})

	_, err = userFile.Write(userConfig)
	require.NoError(t, err)
	err = userFile.Close()
	require.NoError(t, err)

	return userJWT, userKeyPair, userFilePath
}

func validateUserJWTHandler(t *testing.T, wantUserJWT string, haveUserJWTHandler nats.UserJWTHandler) {
	haveUserJWT, err := haveUserJWTHandler()
	assert.NoError(t, err)
	assert.Equal(t, wantUserJWT, haveUserJWT)
}

func validateSignatureCB(t *testing.T, wantSignatureCB, haveSignatureCB nats.SignatureHandler) {
	nonce := []byte("nonce")
	haveSignedNonce, err := haveSignatureCB(nonce)
	assert.NoError(t, err)
	wantSignedNonce, err := wantSignatureCB(nonce)
	assert.NoError(t, err)
	assert.Equal(t, wantSignedNonce, haveSignedNonce)
}

func TestNatsOptions(t *testing.T) {
	t.Parallel()

	t.Run("SetURL", func(t *testing.T) {
		url := "url"

		var no NatsOptions
		no.SetURL(url)
		options := no.buildOptions()

		assert.Equal(t, url, options.Url)
	})

	t.Run("SetTLS", func(t *testing.T) {
		tlsConfig := &tls.Config{}

		var no NatsOptions
		no.SetTLS(tlsConfig)
		options := no.buildOptions()

		assert.Equal(t, tlsConfig, options.TLSConfig)
	})

	t.Run("SetPedantic", func(t *testing.T) {
		pedantic := true

		var no NatsOptions
		no.SetPedantic(pedantic)
		options := no.buildOptions()

		assert.Equal(t, pedantic, options.Pedantic)
	})

	t.Run("SetToken", func(t *testing.T) {
		token := "token"

		var no NatsOptions
		no.SetToken(token)
		options := no.buildOptions()

		assert.Equal(t, token, options.Token)
	})

	t.Run("SetUser", func(t *testing.T) {
		user := "user"
		password := "password"

		var no NatsOptions
		no.SetUser(user, password)
		options := no.buildOptions()

		assert.Equal(t, user, options.User)
		assert.Equal(t, password, options.Password)
	})

	t.Run("SetNkey", func(t *testing.T) {
		keyPair := createNkey(t)
		pubKey, err := keyPair.PublicKey()
		require.NoError(t, err)
		seed, err := keyPair.Seed()
		require.NoError(t, err)

		var no NatsOptions
		err = no.SetNkey(seed)
		assert.NoError(t, err)
		options := no.buildOptions()

		assert.Equal(t, pubKey, options.Nkey)

		validateSignatureCB(t, keyPair.Sign, options.SignatureCB)
	})

	t.Run("SetNkeyJWT", func(t *testing.T) {
		userJWT, userKeyPair := createNkeyJWT(t)
		userSeed, err := userKeyPair.Seed()
		require.NoError(t, err)

		var no NatsOptions
		err = no.SetNkeyJWT(userJWT, userSeed)
		assert.NoError(t, err)
		options := no.buildOptions()

		validateUserJWTHandler(t, userJWT, options.UserJWT)
		validateSignatureCB(t, userKeyPair.Sign, options.SignatureCB)
	})

	t.Run("SetNkeyUserFile", func(t *testing.T) {
		userJWT, userKeyPair, userFilePath := createNkeyUserFile(t)

		var no NatsOptions
		err := no.SetNkeyUserFile(userFilePath)
		assert.NoError(t, err)
		options := no.buildOptions()

		validateUserJWTHandler(t, userJWT, options.UserJWT)
		validateSignatureCB(t, userKeyPair.Sign, options.SignatureCB)
	})
}
