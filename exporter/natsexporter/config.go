// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter"

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
)

// TokenConfig is the config for token auth.
type TokenConfig struct {
	// Token is the plaintext or bcrypt-hashed token.
	Token string `mapstructure:"token"`
}

// UserConfig is the config for username/password auth.
type UserConfig struct {
	// Username is the plaintext username.
	Username string `mapstructure:"username"`
	// Password is the plaintext or bcrypt-hashed password.
	Password string `mapstructure:"password"`
}

// NkeyConfig is the config for NKey auth.
type NkeyConfig struct {
	// Seed is the NKey seed.
	Seed string `mapstructure:"seed"`
}

func (c *NkeyConfig) Validate() error {
	if _, err := nkeys.FromSeed([]byte(c.Seed)); err != nil {
		return fmt.Errorf("failed to decode NKey seed: %w", err)
	}
	return nil
}

// NkeyJWTConfig is the config for decentralized NKey auth via JWT.
type NkeyJWTConfig struct {
	// UserJWT is the NKey user JWT.
	UserJWT string `mapstructure:"user_jwt"`
	// Seed is the NKey seed.
	Seed string `mapstructure:"seed"`
}

func (c *NkeyJWTConfig) Validate() error {
	var errs error

	if _, err := jwt.Decode(c.UserJWT); err != nil {
		errs = multierr.Append(errs,
			fmt.Errorf("failed to validate NKey user JWT: %w", err),
		)
	}

	if _, err := nkeys.FromSeed([]byte(c.Seed)); err != nil {
		errs = multierr.Append(errs,
			fmt.Errorf("failed to validate NKey seed: %w", err),
		)
	}

	return errs
}

// NkeyUserFileConfig is the config for decentralized NKey auth via user file.
type NkeyUserFileConfig struct {
	// UserFilePath is the path to the NKey user file.
	UserFilePath string `mapstructure:"user_file"`
}

func (c *NkeyUserFileConfig) Validate() error {
	userFile, err := os.ReadFile(c.UserFilePath)
	if err != nil {
		return fmt.Errorf("failed to read NKey user file: %w", err)
	}

	var errs error

	if _, err := jwt.ParseDecoratedJWT(userFile); err != nil {
		errs = multierr.Append(errs,
			fmt.Errorf("failed to parse NKey user JWT from NKey user file: %w", err),
		)
	}

	if _, err := jwt.ParseDecoratedNKey(userFile); err != nil {
		errs = multierr.Append(errs,
			fmt.Errorf("failed to parse NKey seed from NKey user file: %w", err),
		)
	}

	return errs
}

// AuthConfig is the config for auth.
type AuthConfig struct {
	// TokenConfig is the optional token auth config.
	TokenConfig *TokenConfig `mapstructure:"token"`
	// UserConfig is the optional username/password auth config.
	UserConfig *UserConfig `mapstructure:"user"`
	// NkeyConfig is the optional NKey auth config.
	NkeyConfig *NkeyConfig `mapstructure:"nkey"`
	// NkeyJWTConfig is the optional NKey JWT auth config.
	NkeyJWTConfig *NkeyJWTConfig `mapstructure:"nkey_jwt"`
	// NkeyUserFileConfig is the NKey user file auth config.
	NkeyUserFileConfig *NkeyUserFileConfig `mapstructure:"nkey_user_file"`
}

func (c *AuthConfig) Validate() error {
	nkeyCount := 0
	var errs error

	if c.NkeyConfig != nil {
		nkeyCount++
		errs = multierr.Append(errs, c.NkeyConfig.Validate())
	}

	if c.NkeyJWTConfig != nil {
		nkeyCount++
		errs = multierr.Append(errs, c.NkeyJWTConfig.Validate())
	}

	if c.NkeyUserFileConfig != nil {
		nkeyCount++
		errs = multierr.Append(errs, c.NkeyUserFileConfig.Validate())
	}

	if nkeyCount > 1 {
		errs = multierr.Append(errs, errors.New("NKey configured more than once"))
	}

	return errs
}

// ResolverConfig is the config for the marshaler resolver.
type ResolverConfig struct {
	// MarshalerName is the name of the built-in marshaler.
	MarshalerName marshal.BuiltinMarshalerName `mapstructure:"marshaler"`
	// EncodingExtensionName is the optional name of the encoding extension. Behavior overrides MarshalerName if set.
	EncodingExtensionName *string `mapstructure:"encoding_extension"`
}

func (c *ResolverConfig) Validate() error {
	if c.EncodingExtensionName != nil {
		var id component.ID
		if err := id.UnmarshalText([]byte(*c.EncodingExtensionName)); err != nil {
			return fmt.Errorf("invalid encoding extension name: %w", err)
		}
		return nil
	}

	if c.MarshalerName != marshal.OtlpProtoBuiltinMarshalerName &&
		c.MarshalerName != marshal.OtlpJSONBuiltinMarshalerName {
		return fmt.Errorf("unsupported built-in marshaler: %s", c.MarshalerName)
	}

	return nil
}

// LogsConfig is the config for the logs exporter.
type LogsConfig struct {
	// Subject is the OTTL value expression used to construct the logs subject.
	Subject string `mapstructure:"subject"`
	// ResolverConfig is the logs marshaler resolver config.
	ResolverConfig ResolverConfig `mapstructure:",squash"`
}

func (c *LogsConfig) Validate() error {
	var errs error

	parser, err := ottllog.NewParser(
		ottlfuncs.StandardConverters[ottllog.TransformContext](),
		componenttest.NewNopTelemetrySettings(),
	)
	if err != nil {
		panic(err)
	}

	if _, err := parser.ParseValueExpression(c.Subject); err != nil {
		errs = multierr.Append(errs,
			fmt.Errorf("failed to parse logs subject expression: %w", err),
		)
	}

	errs = multierr.Append(errs, c.ResolverConfig.Validate())

	return errs
}

// MetricsConfig is the config for the metrics exporter.
type MetricsConfig struct {
	// Subject is the OTTL value expression used to construct the metrics subject.
	Subject string `mapstructure:"subject"`
	// ResolverConfig is the metrics marshaler resolver config.
	ResolverConfig ResolverConfig `mapstructure:",squash"`
}

func (c *MetricsConfig) Validate() error {
	var errs error

	parser, err := ottlmetric.NewParser(
		ottlfuncs.StandardConverters[ottlmetric.TransformContext](),
		componenttest.NewNopTelemetrySettings(),
	)
	if err != nil {
		panic(err)
	}

	if _, err := parser.ParseValueExpression(c.Subject); err != nil {
		errs = multierr.Append(errs,
			fmt.Errorf("failed to parse metrics subject expression: %w", err),
		)
	}

	errs = multierr.Append(errs, c.ResolverConfig.Validate())

	return errs
}

// TracesConfig is the config for the traces exporter.
type TracesConfig struct {
	// Subject is the OTTL value expression used to construct the traces subject.
	Subject string `mapstructure:"subject"`
	// ResolverConfig is the traces marshaler resolver config.
	ResolverConfig ResolverConfig `mapstructure:",squash"`
}

func (c *TracesConfig) Validate() error {
	var errs error

	parser, err := ottlspan.NewParser(
		ottlfuncs.StandardConverters[ottlspan.TransformContext](),
		componenttest.NewNopTelemetrySettings(),
	)
	if err != nil {
		panic(err)
	}

	if _, err := parser.ParseValueExpression(c.Subject); err != nil {
		errs = multierr.Append(errs,
			fmt.Errorf("failed to parse traces subject expression: %w", err),
		)
	}

	errs = multierr.Append(errs, c.ResolverConfig.Validate())

	return errs
}

// NatsConfig is the config for the NATS connection.
type NatsConfig struct {
	// Endpoint is the NATS server endpoint.
	Endpoint string `mapstructure:"endpoint"`
	// TLS is the TLS config.
	TLS configtls.ClientConfig `mapstructure:"tls"`
	// Pedantic is NATS pedantic mode flag.
	Pedantic bool `mapstructure:"pedantic"`
	// Compression is NATS compression flag.
	Compression bool `mapstructure:"compression"`
	// AuthConfig is the auth config.
	AuthConfig AuthConfig `mapstructure:"auth"`
}

func (c *NatsConfig) Validate() error {
	var errs error
	errs = multierr.Append(errs, c.TLS.Validate())
	errs = multierr.Append(errs, c.AuthConfig.Validate())
	return errs
}

// JetStreamConfig is the config for JetStream subsystem.
type JetStreamConfig struct {
	// RetryWait is the retry wait duration passed to JetStream.
	RetryWait *time.Duration `mapstructure:"retry_wait"`
	// RetryAttempts is the retry attempts passed to JetStream.
	RetryAttempts *int `mapstructure:"retry_attempts"`
	// StallWait is the stall wait duration passed to JetStream.
	StallWait *time.Duration `mapstructure:"stall_wait"`
	// Deduplication is the deduplication flag.
	Deduplication *bool `mapstructure:"deduplication"`
}

// Config is the config for the NATS exporter.
type Config struct {
	// NatsConfig is the NATS connection config.
	NatsConfig NatsConfig `mapstructure:",squash"`
	// LogsConfig is the logs exporter config.
	LogsConfig LogsConfig `mapstructure:"logs"`
	// MetricsConfig is the metrics exporter config.
	MetricsConfig MetricsConfig `mapstructure:"metrics"`
	// TracesConfig is the traces exporter config.
	TracesConfig TracesConfig `mapstructure:"traces"`
	// JetStreamConfig is the JetStream config.
	JetStreamConfig *JetStreamConfig `mapstructure:"jetstream"`
	// QueueBatchConfig is the queue batch config.
	QueueBatchConfig exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`
}

func (c *Config) Validate() error {
	var errs error
	errs = multierr.Append(errs, c.NatsConfig.Validate())
	errs = multierr.Append(errs, c.LogsConfig.Validate())
	errs = multierr.Append(errs, c.MetricsConfig.Validate())
	errs = multierr.Append(errs, c.TracesConfig.Validate())
	errs = multierr.Append(errs, c.QueueBatchConfig.Validate())
	return errs
}
