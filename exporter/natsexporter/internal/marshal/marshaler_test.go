// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package marshal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

type fakeGenericMarshaler struct{}

func (m *fakeGenericMarshaler) MarshalString(sd string) ([]byte, error) {
	return []byte(sd), nil
}

var _ GenericMarshaler = (*fakeGenericMarshaler)(nil)

type fakeResolver struct{}

func (r *fakeResolver) Resolve(host component.Host) (GenericMarshaler, error) {
	return &fakeGenericMarshaler{}, nil
}

var _ Resolver = (*fakeResolver)(nil)

func fakePick(genericMarshaler GenericMarshaler) (MarshalFunc[string], error) {
	return genericMarshaler.(*fakeGenericMarshaler).MarshalString, nil
}

var _ PickFunc[string] = fakePick

func TestMarshaler(t *testing.T) {
	t.Parallel()

	t.Run("composes resolver and pickFunc", func(t *testing.T) {
		marshaler := NewMarshaler(&fakeResolver{}, fakePick)

		err := marshaler.Resolve(componenttest.NewNopHost())
		assert.NoError(t, err)

		marshaled, err := marshaler.Marshal("test")
		assert.NoError(t, err)
		assert.Equal(t, []byte("test"), marshaled)
	})
}
