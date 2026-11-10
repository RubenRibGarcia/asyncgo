package asyncgo

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamed(t *testing.T) {
	t.Run("should_record_the_document_name", func(t *testing.T) {
		got := Spec().Named("orders")

		assert.Equal(t, "orders", got.Name)
		require.NoError(t, got.Err)
	})

	t.Run("should_return_the_same_result_so_it_chains_off_spec", func(t *testing.T) {
		base := Spec()

		assert.Same(t, base, base.Named("orders"))
	})

	t.Run("should_clear_an_earlier_name_when_empty", func(t *testing.T) {
		got := Spec().Named("orders").Named("")

		assert.Empty(t, got.Name, "an empty name means \"no explicit name\"")
		require.NoError(t, got.Err)
	})

	t.Run("should_reject_a_name_that_is_not_filename_safe", func(t *testing.T) {
		for _, name := range []string{"../escape", "a/b", "with space", "with:colon"} {
			t.Run(name, func(t *testing.T) {
				got := Spec().Named(name)

				require.Error(t, got.Err)
				assert.Contains(t, got.Err.Error(), "document name")
				assert.Contains(t, got.Err.Error(), "must match")
			})
		}
	})

	t.Run("should_keep_a_validation_error_already_on_the_result", func(t *testing.T) {
		// Named joins its error onto whatever the build already reported, so a
		// mistyped name cannot hide a catalog that failed validation.
		got := Spec(badItem{}).Named("../escape")

		require.Error(t, got.Err)
		assert.Contains(t, got.Err.Error(), "boom")
		assert.Contains(t, got.Err.Error(), "document name")
	})
}

// errBoom is the failure a badItem reports, so a SpecResult carries a validation
// error to join onto.
var errBoom = errors.New("boom")

// badItem is an Item whose apply fails.
type badItem struct{}

func (badItem) apply(*builder) error { return errBoom }
