// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/metadata"
)

func createNkey(t *testing.T) (string, []byte) {
	keyPair, err := nkeys.CreateUser()
	require.NoError(t, err)
	t.Cleanup(func() {
		keyPair.Wipe()
	})

	pubKey, err := keyPair.PublicKey()
	require.NoError(t, err)

	seed, err := keyPair.Seed()
	require.NoError(t, err)

	return pubKey, seed
}

func createNkeyJWT(t *testing.T, expirationDuration time.Duration) (string, []byte) {
	userPubKey, userSeed := createNkey(t)

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
		expirationDuration,
	)
	require.NoError(t, err)

	return userJWT, userSeed
}

func createNkeyUserFile(t *testing.T, expirationDuration time.Duration) string {
	userJWT, userSeed := createNkeyJWT(t, expirationDuration)

	userConfig, err := jwt.FormatUserConfig(userJWT, userSeed)
	require.NoError(t, err)

	userFile, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)
	userFilePath := userFile.Name()
	t.Cleanup(func() {
		os.Remove(userFilePath)
	})

	_, err = userFile.Write(userConfig)
	require.NoError(t, err)
	err = userFile.Close()
	require.NoError(t, err)

	return userFilePath
}

func TestNkeyConfig(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for valid config", func(t *testing.T) {
		_, seed := createNkey(t)

		cfg := &NkeyConfig{
			Seed: string(seed),
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("should return error for invalid seed", func(t *testing.T) {
		cfg := &NkeyConfig{
			Seed: "invalid",
		}
		assert.Error(t, cfg.Validate())
	})
}

func TestNkeyJWTConfig(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for valid config", func(t *testing.T) {
		userJWT, userSeed := createNkeyJWT(t, 5*time.Minute)

		cfg := &NkeyJWTConfig{
			UserJWT: userJWT,
			Seed:    string(userSeed),
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("should return error for invalid JWT", func(t *testing.T) {
		_, userSeed := createNkeyJWT(t, 5*time.Minute)

		cfg := &NkeyJWTConfig{
			UserJWT: "invalid",
			Seed:    string(userSeed),
		}
		assert.Error(t, cfg.Validate())
	})

	t.Run("should return error for invalid seed", func(t *testing.T) {
		userJWT, _ := createNkeyJWT(t, 5*time.Minute)

		cfg := &NkeyJWTConfig{
			UserJWT: userJWT,
			Seed:    "invalid",
		}
		assert.Error(t, cfg.Validate())
	})
}

func TestNkeyUserFileConfig(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for valid config", func(t *testing.T) {
		userFilePath := createNkeyUserFile(t, 5*time.Minute)

		cfg := &NkeyUserFileConfig{
			UserFilePath: userFilePath,
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("should return error for non-existent user file", func(t *testing.T) {
		cfg := &NkeyUserFileConfig{
			UserFilePath: "invalid",
		}
		assert.Error(t, cfg.Validate())
	})

	t.Run("should return error for invalid user file", func(t *testing.T) {
		userFile, err := os.CreateTemp(t.TempDir(), "")
		require.NoError(t, err)
		userFilePath := userFile.Name()
		t.Cleanup(func() {
			os.Remove(userFilePath)
		})

		cfg := &NkeyUserFileConfig{
			UserFilePath: userFilePath,
		}
		assert.Error(t, cfg.Validate())
	})
}

func TestAuthConfig(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for valid config", func(t *testing.T) {
		cfg := &AuthConfig{}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("should return error for multiple NKey configs", func(t *testing.T) {
		_, seed := createNkey(t)
		userJWT, userSeed := createNkeyJWT(t, 5*time.Minute)
		userFilePath := createNkeyUserFile(t, 5*time.Minute)

		cfg := &AuthConfig{
			NkeyConfig: &NkeyConfig{
				Seed: string(seed),
			},
			NkeyJWTConfig: &NkeyJWTConfig{
				UserJWT: userJWT,
				Seed:    string(userSeed),
			},
			NkeyUserFileConfig: &NkeyUserFileConfig{
				UserFilePath: userFilePath,
			},
		}
		assert.Error(t, cfg.Validate())
	})
}

func TestResolverConfig(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for valid config", func(t *testing.T) {
		cfg := &ResolverConfig{
			MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("should return error for unsupported built-in marshaler name", func(t *testing.T) {
		cfg := &ResolverConfig{
			MarshalerName: "unsupported",
		}
		assert.Error(t, cfg.Validate())
	})

	t.Run("should return error for invalid encoding extension name", func(t *testing.T) {
		cfg := &ResolverConfig{
			EncodingExtensionName: &[]string{"/"}[0],
		}
		assert.Error(t, cfg.Validate())
	})

	t.Run("should return nil for unsupported built-in marshaler name and valid encoding extension name", func(t *testing.T) {
		cfg := &ResolverConfig{
			MarshalerName:         "unsupported",
			EncodingExtensionName: &[]string{"extension"}[0],
		}
		assert.NoError(t, cfg.Validate())
	})
}

func TestLogsConfig(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for valid config", func(t *testing.T) {
		cfg := &LogsConfig{
			Subject: "\"otel_logs\"",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("should return error for invalid subject", func(t *testing.T) {
		cfg := &LogsConfig{
			Subject: "invalid",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		}
		assert.Error(t, cfg.Validate())
	})
}

func TestMetricsConfig(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for valid config", func(t *testing.T) {
		cfg := &MetricsConfig{
			Subject: "\"otel_metrics\"",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("should return error for invalid subject", func(t *testing.T) {
		cfg := &MetricsConfig{
			Subject: "invalid",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		}
		assert.Error(t, cfg.Validate())
	})
}

func TestTracesConfig(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for valid config", func(t *testing.T) {
		cfg := &TracesConfig{
			Subject: "\"otel_traces\"",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("should return error for invalid subject", func(t *testing.T) {
		cfg := &TracesConfig{
			Subject: "invalid",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		}
		assert.Error(t, cfg.Validate())
	})
}

func TestNatsConfig(t *testing.T) {
	t.Parallel()

	cfg := &NatsConfig{
		TLS: configtls.NewDefaultClientConfig(),
	}
	assert.NoError(t, cfg.Validate())
}

func TestConfig(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		LogsConfig: LogsConfig{
			Subject: "\"otel_logs\"",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		},
		MetricsConfig: MetricsConfig{
			Subject: "\"otel_metrics\"",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		},
		TracesConfig: TracesConfig{
			Subject: "\"otel_traces\"",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		},
	}
	assert.NoError(t, cfg.Validate())
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	wantCfg := &Config{
		NatsConfig: NatsConfig{
			Endpoint:    "nats://localhost:1234",
			TLS:         configtls.NewDefaultClientConfig(),
			Pedantic:    true,
			Compression: true,
			AuthConfig: AuthConfig{
				TokenConfig: &TokenConfig{
					Token: "token",
				},
				UserConfig: &UserConfig{
					Username: "user",
					Password: "password",
				},
			},
		},
		LogsConfig: LogsConfig{
			Subject: "\"otel_logs\"",
			ResolverConfig: ResolverConfig{
				MarshalerName:         marshal.OtlpJSONBuiltinMarshalerName,
				EncodingExtensionName: &[]string{"encoding"}[0],
			},
		},
		MetricsConfig: MetricsConfig{
			Subject: "\"otel_metrics\"",
			ResolverConfig: ResolverConfig{
				MarshalerName:         marshal.OtlpJSONBuiltinMarshalerName,
				EncodingExtensionName: &[]string{"encoding"}[0],
			},
		},
		TracesConfig: TracesConfig{
			Subject: "\"otel_traces\"",
			ResolverConfig: ResolverConfig{
				MarshalerName:         marshal.OtlpJSONBuiltinMarshalerName,
				EncodingExtensionName: &[]string{"encoding"}[0],
			},
		},
		JetStreamConfig: &JetStreamConfig{
			RetryWait:     &[]time.Duration{2 * time.Second}[0],
			RetryAttempts: &[]int{3}[0],
			StallWait:     &[]time.Duration{4 * time.Second}[0],
			Deduplication: &[]bool{true}[0],
		},
		QueueBatchConfig: exporterhelper.NewDefaultQueueConfig(),
	}

	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	id := component.NewID(metadata.Type)

	sub, err := cm.Sub(id.String())
	require.NoError(t, err)

	haveCfg := createDefaultConfig()
	require.NoError(t, sub.Unmarshal(haveCfg))

	assert.NoError(t, xconfmap.Validate(haveCfg))
	assert.Equal(t, wantCfg, haveCfg)
}
