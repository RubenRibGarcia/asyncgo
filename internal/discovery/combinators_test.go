package discovery

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCombinatorName(t *testing.T) {
	t.Run("should_resolve_short_name_against_declaring_package", func(t *testing.T) {
		assert.Equal(
			t,
			combinatorRef{ImportPath: "example.com/orders", TypeName: "OrderPlaced"},
			resolveCombinatorName("OrderPlaced", "example.com/orders"),
		)
	})

	t.Run("should_pass_through_fully_qualified_name", func(t *testing.T) {
		assert.Equal(
			t,
			combinatorRef{ImportPath: "example.com/orders", TypeName: "OrderPlaced"},
			resolveCombinatorName("example.com/orders.OrderPlaced", "unused"),
		)
	})

	t.Run("should_keep_path_without_dot_as_import_path", func(t *testing.T) {
		assert.Equal(
			t,
			combinatorRef{ImportPath: "foo/bar"},
			resolveCombinatorName("foo/bar", "unused"),
		)
	})
}

func TestCombinatorNamesInTag(t *testing.T) {
	tests := []struct {
		name   string
		tag    string
		key    string
		want   []string
		wantOK bool
	}{
		{
			name:   "should_return_names_when_present",
			tag:    "oneOf=A|B",
			key:    "oneOf",
			want:   []string{"A", "B"},
			wantOK: true,
		},
		{
			name:   "should_return_false_when_absent",
			tag:    "required",
			key:    "oneOf",
			wantOK: false,
		},
		{
			name:   "should_return_false_when_value_empty",
			tag:    "oneOf=",
			key:    "oneOf",
			wantOK: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := combinatorNamesInTag(tc.tag, tc.key)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCollectCombinatorRefs(t *testing.T) {
	t.Run("should_collect_union_members_from_example", func(t *testing.T) {
		dir, err := filepath.Abs(filepath.Join("..", "..", "test", "data", "oneof"))
		require.NoError(t, err)

		pkgs, err := load(dir)
		require.NoError(t, err)

		refs := collectCombinatorRefs(pkgs, reachableFromMain(pkgs))

		const pkg = "github.com/RubenRibGarcia/asyncgo/test/data/oneof"
		require.Contains(t, refs, combinatorRef{ImportPath: pkg, TypeName: "OrderPlaced"})
		require.Contains(t, refs, combinatorRef{ImportPath: pkg, TypeName: "OrderCancelled"})
	})
}
