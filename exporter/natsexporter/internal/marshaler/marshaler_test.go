// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package marshaler

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

var _ genericMarshaler = (*fakeGenericMarshaler)(nil)

type fakeResolver struct{}

func (r *fakeResolver) resolve(host component.Host) (genericMarshaler, error) {
	return &fakeGenericMarshaler{}, nil
}

var _ resolver = (*fakeResolver)(nil)

func fakePick(genericMarshaler genericMarshaler) (marshalFunc[string], error) {
	return genericMarshaler.(*fakeGenericMarshaler).MarshalString, nil
}

var _ pickFunc[string] = fakePick

func newMarshalerWithFakes() *Marshaler[string] {
	return newMarshaler(&fakeResolver{}, fakePick)
}

func TestMarshaler(t *testing.T) {
	t.Parallel()

	t.Run("composes resolver and pick", func(t *testing.T) {
		marshaler := newMarshalerWithFakes()

		err := marshaler.Resolve(componenttest.NewNopHost())
		assert.NoError(t, err)

		marshaled, err := marshaler.Marshal("test")
		assert.NoError(t, err)
		assert.Equal(t, []byte("test"), marshaled)
	})
}
