package bot

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDispatchRoutesEveryCallback checks each known "kind:value" reaches a
// service method (no error) and unknown values are rejected.
func TestDispatchRoutesEveryCallback(t *testing.T) {
	ctx := context.Background()
	r, _, _ := newRouter(t)
	known := []string{
		"variant:gb",
		"nav:menu",
		"menu:test",
		"menu:learn",
		"menu:mywords",
		"menu:list",
		"menu:search",
		"wl:all",
		"quiz:help",
		"quiz:skip",
		"res:keep",
		"res:drop",
		"lc:0",
		"lp:1",
		"tog:go",
		"list:ok",
		"list:cancel",
		"list:back",
	}
	unknown := []string{
		"menu:nope",
		"quiz:nope",
		"res:nope",
		"lc:x",
		"lp:x",
		"list:nope",
		"nope:x",
	}
	for _, data := range append(known, unknown...) {
		kind, value, _ := strings.Cut(data, ":")
		_, err := r.dispatch(ctx, dispatchArgs{
			UserID: 7,
			Kind:   kind,
			Value:  value,
		})
		if slices.Contains(unknown, data) {
			require.Error(t, err, data)
			continue
		}
		require.NoError(t, err, data)
	}
}
