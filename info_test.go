package asyncgo

import (
	"testing"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfoFields(t *testing.T) {
	i := Info("Orders", "1.0.0").
		Description("desc").
		TermsOfService("tos").
		Contact(spec.Contact{Name: "Orders Team", Email: "orders@example.com"}).
		License(spec.License{Name: "MIT"}).
		Tags(spec.Tag{Name: "orders"})

	assert.Equal(t, "desc", i.info.Description)
	assert.Equal(t, "tos", i.info.TermsOfService)
	require.NotNil(t, i.info.Contact)
	assert.Equal(t, "Orders Team", i.info.Contact.Name)
	require.NotNil(t, i.info.License)
	assert.Equal(t, "MIT", i.info.License.Name)
	require.Len(t, i.info.Tags, 1)
	assert.Equal(t, "orders", i.info.Tags[0].Name)
}

func TestInfoValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec func() *SpecResult
		want string
	}{
		{
			name: "should_return_error_when_info_title_is_missing",
			spec: func() *SpecResult {
				return Spec(Info("", "1.0.0"))
			},
			want: "info.title: is required",
		},
		{
			name: "should_return_error_when_info_version_is_missing",
			spec: func() *SpecResult {
				return Spec(Info("Orders", ""))
			},
			want: "info.version: is required",
		},
		{
			name: "should_join_multiple_validation_errors",
			spec: func() *SpecResult {
				return Spec(
					Info("", ""),
					Servers(Server("prod", "", "")),
				)
			},
			want: "info.title: is required\ninfo.version: is required\nserver.prod.protocol: is required\nserver.prod.host: is required",
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
