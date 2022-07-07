package scope

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManagerValidate(t *testing.T) {
	manager, err := NewManager([]string{`google\..*`, `192\.168\.1\.1`}, []string{`uber\..*`}, true)
	require.NoError(t, err, "could not create scope manager")

	parsed, _ := url.Parse("https://google.com")
	validated, err := manager.Validate(parsed)
	require.NoError(t, err, "could not validate url")
	require.True(t, validated, "could not get correct in-scope validation")

	parsed, _ = url.Parse("https://test.google.com")
	validated, err = manager.Validate(parsed)
	require.NoError(t, err, "could not validate url")
	require.True(t, validated, "could not get correct in-scope validation")

	parsed, _ = url.Parse("https://uber.com")
	validated, err = manager.Validate(parsed)
	require.NoError(t, err, "could not validate url")
	require.False(t, validated, "could not get correct out-scope validation")

	parsed, _ = url.Parse("https://192.168.1.1")
	validated, err = manager.Validate(parsed)
	require.NoError(t, err, "could not validate url")
	require.True(t, validated, "could not get correct in-scope ip validation")
}