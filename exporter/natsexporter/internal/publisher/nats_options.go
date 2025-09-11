package publisher

import (
	"crypto/tls"
	"fmt"
	"os"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

type setNatsOptionsFunc func(options *nats.Options)

type NatsOptions struct {
	setOptionFuncs []setNatsOptionsFunc
}

func (no *NatsOptions) SetURL(url string) {
	no.setOptionFuncs = append(no.setOptionFuncs, func(options *nats.Options) {
		options.Url = url
	})
}

func (no *NatsOptions) SetTLS(tls *tls.Config) {
	no.setOptionFuncs = append(no.setOptionFuncs, func(options *nats.Options) {
		options.TLSConfig = tls
	})
}

func (no *NatsOptions) SetPedantic(pedantic bool) {
	no.setOptionFuncs = append(no.setOptionFuncs, func(options *nats.Options) {
		options.Pedantic = pedantic
	})
}

func (no *NatsOptions) SetToken(token string) {
	no.setOptionFuncs = append(no.setOptionFuncs, func(options *nats.Options) {
		options.Token = token
	})
}

func (no *NatsOptions) SetUser(user string, password string) {
	no.setOptionFuncs = append(no.setOptionFuncs, func(options *nats.Options) {
		options.User = user
		options.Password = password
	})
}

func (no *NatsOptions) SetNkey(seed []byte) error {
	keyPair, err := nkeys.FromSeed(seed)
	if err != nil {
		return fmt.Errorf("failed to decode seed: %w", err)
	}

	publicKey, err := keyPair.PublicKey()
	if err != nil {
		return fmt.Errorf("failed to derive public key from seed: %w", err)
	}

	no.setOptionFuncs = append(no.setOptionFuncs, func(options *nats.Options) {
		options.Nkey = publicKey
		options.SignatureCB = keyPair.Sign
	})
	return nil
}

func (no *NatsOptions) SetNkeyJWT(userJWT string, seed []byte) error {
	keyPair, err := nkeys.FromSeed(seed)
	if err != nil {
		return fmt.Errorf("failed to decode seed: %w", err)
	}

	no.setOptionFuncs = append(no.setOptionFuncs, func(options *nats.Options) {
		options.UserJWT = func() (string, error) {
			return userJWT, nil
		}
		options.SignatureCB = keyPair.Sign
	})
	return nil
}

func (no *NatsOptions) SetNkeyUserFile(userFilePath string) error {
	userFile, err := os.ReadFile(userFilePath)
	if err != nil {
		return fmt.Errorf("failed to read user file: %w", err)
	}

	userJWT, err := jwt.ParseDecoratedJWT(userFile)
	if err != nil {
		return fmt.Errorf("failed to parse JWT from user file: %w", err)
	}

	keyPair, err := jwt.ParseDecoratedNKey(userFile)
	if err != nil {
		return fmt.Errorf("failed to parse seed from user file: %w", err)
	}

	no.setOptionFuncs = append(no.setOptionFuncs, func(options *nats.Options) {
		options.UserJWT = func() (string, error) {
			return userJWT, nil
		}
		options.SignatureCB = keyPair.Sign
	})
	return nil
}

func (no *NatsOptions) buildOptions() *nats.Options {
	options := nats.GetDefaultOptions()
	for _, setOptionFunc := range no.setOptionFuncs {
		setOptionFunc(&options)
	}
	return &options
}
