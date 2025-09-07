// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package marshaler // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshaler"

import "go.opentelemetry.io/collector/component"

type Marshaler[T any] struct {
	resolver Resolver
	pick     PickFunc[T]
	marshal  MarshalFunc[T]
}

func (m *Marshaler[T]) Resolve(host component.Host) error {
	genericMarshaler, err := m.resolver.Resolve(host)
	if err != nil {
		return err
	}

	m.marshal, err = m.pick(genericMarshaler)
	if err != nil {
		return err
	}

	return nil
}

func (m *Marshaler[T]) Marshal(data T) ([]byte, error) {
	return m.marshal(data)
}

func NewMarshaler[T any](resolver Resolver, pick PickFunc[T]) Marshaler[T] {
	return Marshaler[T]{
		resolver: resolver,
		pick:     pick,
	}
}
