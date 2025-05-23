package common

import (
	"errors"
	"fmt"
	"testing"

	"github.com/grafana/sobek"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThrow(t *testing.T) {
	t.Parallel()
	rt := sobek.New()
	fn1, ok := sobek.AssertFunction(rt.ToValue(func() { Throw(rt, errors.New("aaaa")) }))
	require.True(t, ok, "fn1 is invalid")
	_, err := fn1(sobek.Undefined())
	assert.EqualError(t, err, "GoError: aaaa")

	fn2, ok := sobek.AssertFunction(rt.ToValue(func() { Throw(rt, err) }))
	require.True(t, ok, "fn2 is invalid")
	_, err = fn2(sobek.Undefined())
	assert.EqualError(t, err, "GoError: aaaa")
}

func TestToBytes(t *testing.T) {
	t.Parallel()
	rt := sobek.New()
	b := []byte("hello")
	testCases := []struct {
		in     interface{}
		expOut []byte
		expErr string
	}{
		{b, b, ""},
		{"hello", b, ""},
		{rt.NewArrayBuffer(b), b, ""},
		{struct{}{}, nil, "invalid type struct {}, expected string, []byte or ArrayBuffer"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%T", tc.in), func(t *testing.T) {
			t.Parallel()
			out, err := ToBytes(tc.in)
			if tc.expErr != "" {
				assert.EqualError(t, err, tc.expErr)
				return
			}
			assert.Equal(t, tc.expOut, out)
		})
	}
}

func TestToString(t *testing.T) {
	t.Parallel()
	rt := sobek.New()
	s := "hello"
	testCases := []struct {
		in             interface{}
		expOut, expErr string
	}{
		{s, s, ""},
		{"hello", s, ""},
		{rt.NewArrayBuffer([]byte(s)), s, ""},
		{struct{}{}, "", "invalid type struct {}, expected string, []byte or ArrayBuffer"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%T", tc.in), func(t *testing.T) {
			t.Parallel()
			out, err := ToString(tc.in)
			if tc.expErr != "" {
				assert.EqualError(t, err, tc.expErr)
				return
			}
			assert.Equal(t, tc.expOut, out)
		})
	}
}
