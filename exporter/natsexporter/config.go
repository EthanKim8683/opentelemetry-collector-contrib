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

type TokenConfig struct {
	Token string `mapstructure:"token"`
}

type UserConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type NkeyConfig struct {
	Seed string `mapstructure:"seed"`
}

func (c *NkeyConfig) Validate() error {
	if _, err := nkeys.FromSeed([]byte(c.Seed)); err != nil {
		return fmt.Errorf("failed to decode NKey seed: %w", err)
	}
	return nil
}

type NkeyJWTConfig struct {
	UserJWT string `mapstructure:"user_jwt"`
	Seed    string `mapstructure:"seed"`
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

type NkeyUserFileConfig struct {
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

type AuthConfig struct {
	TokenConfig        *TokenConfig        `mapstructure:"token"`
	UserConfig         *UserConfig         `mapstructure:"user"`
	NkeyConfig         *NkeyConfig         `mapstructure:"nkey"`
	NkeyJWTConfig      *NkeyJWTConfig      `mapstructure:"nkey_jwt"`
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

type ResolverConfig struct {
	MarshalerName         marshal.BuiltinMarshalerName `mapstructure:"marshaler"`
	EncodingExtensionName *string                      `mapstructure:"encoding_extension"`
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

type LogsConfig struct {
	Subject        string         `mapstructure:"subject"`
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

type MetricsConfig struct {
	Subject        string         `mapstructure:"subject"`
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

type TracesConfig struct {
	Subject        string         `mapstructure:"subject"`
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

type NatsConfig struct {
	Endpoint    string                 `mapstructure:"endpoint"`
	TLS         configtls.ClientConfig `mapstructure:"tls"`
	Pedantic    bool                   `mapstructure:"pedantic"`
	Compression bool                   `mapstructure:"compression"`
	AuthConfig  AuthConfig             `mapstructure:"auth"`
}

func (c *NatsConfig) Validate() error {
	var errs error
	errs = multierr.Append(errs, c.TLS.Validate())
	errs = multierr.Append(errs, c.AuthConfig.Validate())
	return errs
}

type JetStreamConfig struct {
	RetryWait     *time.Duration `mapstructure:"retry_wait"`
	RetryAttempts *int           `mapstructure:"retry_attempts"`
	StallWait     *time.Duration `mapstructure:"stall_wait"`
	Deduplication *bool          `mapstructure:"deduplication"`
}

type Config struct {
	NatsConfig       NatsConfig                      `mapstructure:",squash"`
	LogsConfig       LogsConfig                      `mapstructure:"logs"`
	MetricsConfig    MetricsConfig                   `mapstructure:"metrics"`
	TracesConfig     TracesConfig                    `mapstructure:"traces"`
	JetStreamConfig  *JetStreamConfig                `mapstructure:"jetstream"`
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
