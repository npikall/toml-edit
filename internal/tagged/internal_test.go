package tagged

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromValuePanicsOnUnknownValue(t *testing.T) {
	require.Panics(t, func() { fromValue(struct{}{}) })
}
