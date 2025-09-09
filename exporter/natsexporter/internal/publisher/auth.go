package publisher

import (
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

type authFunc func(options *nats.Options)

type authConfig interface {
	Validate() error
	auth() authFunc
}

type tokenAuthConfig struct {
	Token string `mapstructure:"token"`

	tokenAuth authFunc
}

func (c *tokenAuthConfig) Validate() error {
	if c.tokenAuth != nil {
		return nil
	}

	c.tokenAuth = func(options *nats.Options) {
		options.Token = c.Token
	}
	return nil
}

func (c *tokenAuthConfig) auth() authFunc {
	return c.tokenAuth
}

var _ authConfig = (*tokenAuthConfig)(nil)

type userAuthConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`

	userAuth authFunc
}

func (c *userAuthConfig) Validate() error {
	if c.userAuth != nil {
		return nil
	}

	c.userAuth = func(options *nats.Options) {
		options.User = c.Username
		options.Password = c.Password
	}
	return nil
}

var _ authConfig = (*userAuthConfig)(nil)

func (c *userAuthConfig) auth() authFunc {
	return c.userAuth
}

type nkeyAuthConfig struct {
	PublicKey string `mapstructure:"public_key"`
	Seed      []byte `mapstructure:"seed"`

	nkeyAuth authFunc
}

func (c *nkeyAuthConfig) Validate() error {
	if c.nkeyAuth != nil {
		return nil
	}

	keyPair, err := nkeys.FromSeed(c.Seed)
	if err != nil {
		return err
	}

	c.nkeyAuth = func(options *nats.Options) {
		options.Nkey = c.PublicKey
		options.SignatureCB = keyPair.Sign
	}
	return nil
}

func (c *nkeyAuthConfig) auth() authFunc {
	return c.nkeyAuth
}

var _ authConfig = (*nkeyAuthConfig)(nil)

type nkeyJWTAuthConfig struct {
	JWT  string `mapstructure:"jwt"`
	Seed []byte `mapstructure:"seed"`

	nkeyJWTAuth authFunc
}

func (c *nkeyJWTAuthConfig) Validate() error {
	if c.nkeyJWTAuth != nil {
		return nil
	}

	keyPair, err := nkeys.FromSeed(c.Seed)
	if err != nil {
		return err
	}

	c.nkeyJWTAuth = func(options *nats.Options) {
		options.UserJWT = func() (string, error) {
			return c.JWT, nil
		}
		options.SignatureCB = keyPair.Sign
	}
	return nil
}

func (c *nkeyJWTAuthConfig) auth() authFunc {
	return c.nkeyJWTAuth
}

var _ authConfig = (*nkeyJWTAuthConfig)(nil)

type nkeyUserFileAuthConfig struct {
	UserFilePath []byte `mapstructure:"user_file"`

	nkeyUserFileAuth authFunc
}

func (c *nkeyUserFileAuthConfig) Validate() error {
	if c.nkeyUserFileAuth != nil {
		return nil
	}

	userJWT, err := jwt.ParseDecoratedJWT(c.UserFilePath)
	if err != nil {
		return err
	}

	keyPair, err := jwt.ParseDecoratedNKey(c.UserFilePath)
	if err != nil {
		return err
	}

	c.nkeyUserFileAuth = func(options *nats.Options) {
		options.UserJWT = func() (string, error) {
			return userJWT, nil
		}
		options.SignatureCB = keyPair.Sign
	}
	return nil
}

func (c *nkeyUserFileAuthConfig) auth() authFunc {
	return c.nkeyUserFileAuth
}

var _ authConfig = (*nkeyUserFileAuthConfig)(nil)

type AuthConfig struct {
	tokenAuthConfig        *tokenAuthConfig        `mapstructure:"token"`
	userAuthConfig         *userAuthConfig         `mapstructure:"user"`
	nkeyAuthConfig         *nkeyAuthConfig         `mapstructure:"nkey"`
	nkeyJWTAuthConfig      *nkeyJWTAuthConfig      `mapstructure:"nkey_jwt"`
	nkeyUserFileAuthConfig *nkeyUserFileAuthConfig `mapstructure:"nkey_user_file"`

	auth authFunc
}

func (c *AuthConfig) Validate() error {
	if c.auth != nil {
		return nil
	}

	var authConfigs []authConfig
	if c.tokenAuthConfig != nil {
		authConfigs = append(authConfigs, c.tokenAuthConfig)
	}
	if c.userAuthConfig != nil {
		authConfigs = append(authConfigs, c.userAuthConfig)
	}
	if c.nkeyAuthConfig != nil {
		authConfigs = append(authConfigs, c.nkeyAuthConfig)
	}
	if c.nkeyJWTAuthConfig != nil {
		authConfigs = append(authConfigs, c.nkeyJWTAuthConfig)
	}
	if c.nkeyUserFileAuthConfig != nil {
		authConfigs = append(authConfigs, c.nkeyUserFileAuthConfig)
	}

	auths := make([]authFunc, 0, len(authConfigs))
	for _, authConfig := range authConfigs {
		if err := authConfig.Validate(); err != nil {
			return err
		}
		auths = append(auths, authConfig.auth())
	}

	c.auth = func(options *nats.Options) {
		for _, auth := range auths {
			auth(options)
		}
	}
	return nil
}

func NewDefaultAuthConfig() AuthConfig {
	return AuthConfig{}
}
