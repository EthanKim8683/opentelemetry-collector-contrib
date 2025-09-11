// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter"

import (
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshaler"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type tokenConfig struct {
	Token string `mapstructure:"token"`
}

type userConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type nkeyConfig struct {
	Seed []byte `mapstructure:"seed"`
}

type nkeyJWTConfig struct {
	JWT  string `mapstructure:"jwt"`
	Seed []byte `mapstructure:"seed"`
}

type nkeyUserFileConfig struct {
	UserFilePath string `mapstructure:"user_file_path"`
}

type authConfig struct {
	Token        *tokenConfig        `mapstructure:"token"`
	User         *userConfig         `mapstructure:"user"`
	Nkey         *nkeyConfig         `mapstructure:"nkey"`
	NkeyJWT      *nkeyJWTConfig      `mapstructure:"nkey_jwt"`
	NkeyUserFile *nkeyUserFileConfig `mapstructure:"nkey_user_file"`
}

type resolverConfig struct {
	MarshalerName         marshaler.BuiltinMarshalerName `mapstructure:"marshaler"`
	EncodingExtensionName []byte                         `mapstructure:"encoding_extension"`
}

type logsConfig struct {
	Subject        string         `mapstructure:"subject"`
	resolverConfig resolverConfig `mapstructure:",squash"`
}

type metricsConfig struct {
	Subject        string         `mapstructure:"subject"`
	resolverConfig resolverConfig `mapstructure:",squash"`
}

type tracesConfig struct {
	Subject        string         `mapstructure:"subject"`
	resolverConfig resolverConfig `mapstructure:",squash"`
}

type natsConfig struct {
	Endpoint string                 `mapstructure:"endpoint"`
	TLS      configtls.ClientConfig `mapstructure:"tls"`
	Pedantic bool                   `mapstructure:"pedantic"`
	Auth     authConfig             `mapstructure:"auth"`
}

type jetStreamConfig struct {
	RetryWait     *time.Duration `mapstructure:"retry_wait"`
	RetryAttempts *int           `mapstructure:"retry_attempts"`
	StallWait     *time.Duration `mapstructure:"stall_wait"`
	Deduplicate   *bool          `mapstructure:"deduplicate"`
}

type Config struct {
	natsConfig       natsConfig                      `mapstructure:",squash"`
	Logs             logsConfig                      `mapstructure:"logs"`
	Metrics          metricsConfig                   `mapstructure:"metrics"`
	Traces           tracesConfig                    `mapstructure:"traces"`
	JetStream        *jetStreamConfig                `mapstructure:"jetstream"`
	QueueBatchConfig exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`
}
