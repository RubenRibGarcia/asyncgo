package discovery

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCombinatorName(t *testing.T) {
	t.Run("should_resolve_short_name_against_declaring_package", func(t *testing.T) {
		ref, err := resolveCombinatorName("OrderPlaced", "example.com/orders")
		require.NoError(t, err)
		assert.Equal(
			t,
			combinatorRef{ImportPath: "example.com/orders", TypeName: "OrderPlaced"},
			ref,
		)
	})

	t.Run("should_pass_through_fully_qualified_name", func(t *testing.T) {
		ref, err := resolveCombinatorName("example.com/orders.OrderPlaced", "unused")
		require.NoError(t, err)
		assert.Equal(
			t,
			combinatorRef{ImportPath: "example.com/orders", TypeName: "OrderPlaced"},
			ref,
		)
	})

	t.Run("should_reject_a_path_without_a_type_name", func(t *testing.T) {
		// The harness emits new(alias.TypeName); a slash with no dot produced
		// new(pkgN.) — a Go syntax error surfacing as `exit status 1`.
		_, err := resolveCombinatorName("foo/bar", "unused")
		require.Error(t, err)
	})

	t.Run("should_reject_a_semi_qualified_name", func(t *testing.T) {
		// Without an import path the name is package-local, so a dot cannot be
		// resolved and new(alias.pkg.Type) does not compile.
		_, err := resolveCombinatorName("orders.OrderPlaced", "example.com/app")
		require.Error(t, err)
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

		pkgs, err := load(t.Context(), dir)
		require.NoError(t, err)

		refs, err := collectCombinatorRefs(pkgs, reachableFromMain(pkgs))
		require.NoError(t, err)

		const pkg = "github.com/RubenRibGarcia/asyncgo/test/data/oneof"
		require.Contains(t, refs, combinatorRef{ImportPath: pkg, TypeName: "OrderPlaced"})
		require.Contains(t, refs, combinatorRef{ImportPath: pkg, TypeName: "OrderCancelled"})
	})
}
