package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorrelationIDsRegistersComponents(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		CorrelationIDs(CorrelationID("CorrelationID", spec.CorrelationID{
			Description: "Correlation ID",
			Location:    "$message.header#/correlationId",
		})),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.CorrelationIDs, 1)
	assert.Equal(
		t,
		"$message.header#/correlationId",
		res.Doc.Components.CorrelationIDs["CorrelationID"].Location,
	)
}

// TestCorrelationIDFromReusesAnIdenticalComponent pins the dedupe rule: an
// inline value whose derived key already holds the same value is reused, not
// registered twice.
func TestCorrelationIDFromReusesAnIdenticalComponent(t *testing.T) {
	corr := spec.CorrelationID{Location: "$message.header#/correlationId"}
	b := &builder{doc: spec.New(), defs: map[string]*spec.Schema{}}

	first, err := correlationIDFromRef(b, "OrderPlaced", corr)
	require.NoError(t, err)

	second, err := correlationIDFromRef(b, "OrderPlaced", corr)
	require.NoError(t, err)

	assert.Equal(t, first.Ref, second.Ref)
	assert.Len(t, b.doc.Components.CorrelationIDs, 1)
}

func TestCorrelationIDValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_correlation_id_name_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					CorrelationIDs(CorrelationID("", spec.CorrelationID{
						Location: "$message.header#/correlationId",
					})),
				)
			},
			want: "correlationId.name: is required",
		},
		{
			name: "should_return_error_when_correlation_id_location_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					CorrelationIDs(CorrelationID("CorrelationID", spec.CorrelationID{})),
				)
			},
			want: "correlationId.CorrelationID.location: is required",
		},
		{
			name: "should_return_error_when_correlation_id_name_is_duplicate",
			spec: func() *SpecResult {
				corr := spec.CorrelationID{Location: "$message.header#/correlationId"}
				return Spec(
					Info("Orders", "1.0.0"),
					CorrelationIDs(
						CorrelationID("CorrelationID", corr),
						CorrelationID("CorrelationID", corr),
					),
				)
			},
			want: "correlationId.CorrelationID: duplicate name",
		},
		{
			name: "should_return_error_when_inline_correlation_id_duplicates_a_declared_component",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					CorrelationIDs(CorrelationID("OrderPlacedCorrelationID", spec.CorrelationID{
						Location: "$message.header#/declared",
					})),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Message(MessageOf(OrderPlaced{}).
									Name("OrderPlaced").
									CorrelationIDFrom(spec.CorrelationID{
										Location: "$message.header#/inline",
									}))),
					),
				)
			},
			want: "correlationId.OrderPlacedCorrelationID: duplicate name",
		},
		{
			name: "should_return_error_when_inline_correlation_id_location_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					Channels(
						Channel("order-placed").
							Send(Operation().
								Message(MessageOf(OrderPlaced{}).
									Name("OrderPlaced").
									CorrelationIDFrom(spec.CorrelationID{}))),
					),
				)
			},
			want: "correlationId.OrderPlacedCorrelationID.location: is required",
		},
		{
			name: "should_reject_a_correlation_id_name_outside_the_key_pattern",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					CorrelationIDs(CorrelationID(
						"tenant id",
						spec.CorrelationID{Location: "$message.header#/id"},
					)),
				)
			},
			want: "correlationId.tenant id: name must match ^[\\w\\d.\\-_]+$",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := tc.spec()
			if tc.want == "" {
				require.NoError(t, res.Err)
				assert.Nil(t, res.ValidationErrors())
				return
			}
			require.Error(t, res.Err)
			assert.Equal(t, tc.want, res.Err.Error())
		})
	}
}
