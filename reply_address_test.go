package asyncgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplyAddressFields(t *testing.T) {
	ra := ReplyAddress("ReplyTo").
		Description("Consumer inbox").
		Location("$message.header#/replyTo")

	assert.Equal(t, "ReplyTo", ra.name)
	assert.Equal(t, "Consumer inbox", ra.a.Description)
	assert.Equal(t, "$message.header#/replyTo", ra.a.Location)
}

func TestReplyAddressesRegistersComponents(t *testing.T) {
	res := Spec(
		Info("Orders", "1.0.0"),
		ReplyAddresses(
			ReplyAddress("ReplyTo").Location("$message.header#/replyTo"),
			ReplyAddress("Inbox").Location("$message.header#/inbox"),
		),
	)

	require.NoError(t, res.Err)
	require.NotNil(t, res.Doc.Components)
	assert.Len(t, res.Doc.Components.ReplyAddresses, 2)
	assert.Equal(
		t,
		"$message.header#/replyTo",
		res.Doc.Components.ReplyAddresses["ReplyTo"].Location,
	)
}

func TestReplyAddressValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_reply_address_name_is_empty",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					ReplyAddresses(ReplyAddress("").Location("$message.header#/replyTo")),
				)
			},
			want: "replyAddress.name: is required",
		},
		{
			name: "should_return_error_when_reply_address_location_is_empty",
			spec: func() *SpecResult {
				return Spec(Info("Orders", "1.0.0"), ReplyAddresses(ReplyAddress("ReplyTo")))
			},
			want: "replyAddress.ReplyTo.location: is required",
		},
		{
			name: "should_return_error_when_reply_address_name_is_duplicate",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					ReplyAddresses(
						ReplyAddress("ReplyTo").Location("$message.header#/replyTo"),
						ReplyAddress("ReplyTo").Location("$message.header#/inbox"),
					),
				)
			},
			want: "replyAddress.ReplyTo: duplicate name",
		},
		{
			name: "should_reject_a_reply_address_name_outside_the_key_pattern",
			spec: func() *SpecResult {
				return Spec(
					Info("Orders", "1.0.0"),
					ReplyAddresses(
						ReplyAddress("tenant addr").Location("$message.header#/replyTo"),
					),
				)
			},
			want: "replyAddress.tenant addr: name must match ^[\\w\\d.\\-_]+$",
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
