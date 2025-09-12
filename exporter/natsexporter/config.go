// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter"

import (
	"time"

	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshal"
)

type TokenConfig struct {
	Token string `mapstructure:"token"`
}

type UserConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type NkeyConfig struct {
	Seed []byte `mapstructure:"seed"`
}

type NkeyJWTConfig struct {
	JWT  string `mapstructure:"jwt"`
	Seed []byte `mapstructure:"seed"`
}

type NkeyUserFileConfig struct {
	UserFilePath string `mapstructure:"user_file"`
}

type AuthConfig struct {
	Token        *TokenConfig        `mapstructure:"token"`
	User         *UserConfig         `mapstructure:"user"`
	Nkey         *NkeyConfig         `mapstructure:"nkey"`
	NkeyJWT      *NkeyJWTConfig      `mapstructure:"nkey_jwt"`
	NkeyUserFile *NkeyUserFileConfig `mapstructure:"nkey_user_file"`
}

type ResolverConfig struct {
	MarshalerName         marshal.BuiltinMarshalerName `mapstructure:"marshaler"`
	EncodingExtensionName []byte                       `mapstructure:"encoding_extension"`
}

type LogsConfig struct {
	Subject        string         `mapstructure:"subject"`
	ResolverConfig ResolverConfig `mapstructure:",squash"`
}

type MetricsConfig struct {
	Subject        string         `mapstructure:"subject"`
	ResolverConfig ResolverConfig `mapstructure:",squash"`
}

type TracesConfig struct {
	Subject        string         `mapstructure:"subject"`
	ResolverConfig ResolverConfig `mapstructure:",squash"`
}

type NatsConfig struct {
	Endpoint   string                 `mapstructure:"endpoint"`
	TLS        configtls.ClientConfig `mapstructure:"tls"`
	Pedantic   bool                   `mapstructure:"pedantic"`
	AuthConfig AuthConfig             `mapstructure:"auth"`
}

type JetStreamConfig struct {
	RetryWait     *time.Duration `mapstructure:"retry_wait"`
	RetryAttempts *int           `mapstructure:"retry_attempts"`
	StallWait     *time.Duration `mapstructure:"stall_wait"`
	Dedup         *bool          `mapstructure:"dedup"`
}

type Config struct {
	NatsConfig       NatsConfig                      `mapstructure:",squash"`
	LogsConfig       LogsConfig                      `mapstructure:"logs"`
	MetricsConfig    MetricsConfig                   `mapstructure:"metrics"`
	TracesConfig     TracesConfig                    `mapstructure:"traces"`
	JetStreamConfig  *JetStreamConfig                `mapstructure:"jetstream"`
	QueueBatchConfig exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`
}
