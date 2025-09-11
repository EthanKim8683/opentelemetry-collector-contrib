package publisher

import (
	"crypto/tls"
	"os"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

type setNatsOptionsFunc func(options *nats.Options)

type NatsOptions struct {
	setNatsOptions []setNatsOptionsFunc
}

func (co *NatsOptions) SetTLS(tls *tls.Config) {
	co.setNatsOptions = append(co.setNatsOptions, func(options *nats.Options) {
		options.TLSConfig = tls
	})
}

func (co *NatsOptions) SetPedantic(pedantic bool) {
	co.setNatsOptions = append(co.setNatsOptions, func(options *nats.Options) {
		options.Pedantic = pedantic
	})
}

func (co *NatsOptions) SetToken(token string) {
	co.setNatsOptions = append(co.setNatsOptions, func(options *nats.Options) {
		options.Token = token
	})
}

func (co *NatsOptions) SetUser(user string, password string) {
	co.setNatsOptions = append(co.setNatsOptions, func(options *nats.Options) {
		options.User = user
		options.Password = password
	})
}

func (co *NatsOptions) SetNkey(seed []byte) error {
	keyPair, err := nkeys.FromSeed(seed)
	if err != nil {
		return err
	}

	publicKey, err := keyPair.PublicKey()
	if err != nil {
		return err
	}

	co.setNatsOptions = append(co.setNatsOptions, func(options *nats.Options) {
		options.Nkey = publicKey
		options.SignatureCB = keyPair.Sign
	})
	return nil
}

func (co *NatsOptions) SetNkeyJWT(jwt string, seed []byte) error {
	keyPair, err := nkeys.FromSeed(seed)
	if err != nil {
		return err
	}

	co.setNatsOptions = append(co.setNatsOptions, func(options *nats.Options) {
		options.UserJWT = func() (string, error) {
			return jwt, nil
		}
		options.SignatureCB = keyPair.Sign
	})
	return nil
}

func (co *NatsOptions) SetNkeyUserFile(userFilePath string) error {
	userFile, err := os.ReadFile(userFilePath)
	if err != nil {
		return err
	}

	userJWT, err := jwt.ParseDecoratedJWT(userFile)
	if err != nil {
		return err
	}

	keyPair, err := jwt.ParseDecoratedNKey(userFile)
	if err != nil {
		return err
	}

	co.setNatsOptions = append(co.setNatsOptions, func(options *nats.Options) {
		options.UserJWT = func() (string, error) {
			return userJWT, nil
		}
		options.SignatureCB = keyPair.Sign
	})
	return nil
}

func (co *NatsOptions) buildOptions() *nats.Options {
	options := nats.GetDefaultOptions()
	for _, setOptions := range co.setNatsOptions {
		setOptions(&options)
	}
	return &options
}
