package schema

import (
	"maps"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/RubenRibGarcia/asyncgo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The types below are the corpus for the mapping tests: every mapping is keyed by
// a fully-qualified name, so the types under test have to be named and
// package-level. They shadow nothing in derive_test.go.

// mappingUUID stands in for a third-party named byte array: without a mapping it
// derives as an array of integers, which is the defect the feature exists for.
type mappingUUID [16]byte

// mappingQuantity is a named scalar whose mapping individual tests replace.
type mappingQuantity int

// mappingRaw pins that a byte slice stays kind-derived: the mapping registry is
// exact-name keyed, so it cannot accidentally capture the base64 rule.
type mappingRaw []byte

// mappingOrderID is a derived alias of mappingUUID: with a discovery-emitted
// edge, it inherits mappingUUID's mapping.
type mappingOrderID mappingUUID

// mappingLineID is an alias of an alias, so the chain is two hops deep.
type mappingLineID mappingOrderID

// mappingRecord is a named struct: a mapping for it must bypass hoisting, so its
// fields are never derived.
type mappingRecord struct {
	Name string `json:"name"`
}

// mappingProvider records the type's own suggested schema.
type mappingProvider struct {
	X int `json:"x"`
}

// AsyncAPISchema is the provider that an exact mapping outranks and an inherited
// mapping does not.
func (mappingProvider) AsyncAPISchema() *spec.Schema {
	return &spec.Schema{Type: "string", Format: "provider"}
}

// mappingProviderAlias is a derived alias of a provider-backed struct, pinning
// the tier order inside hoistStruct: the alias's own provider is consulted before
// the mapping it inherits. A defined type does not inherit its base's methods, so
// the provider is declared on the alias itself.
type mappingProviderAlias mappingProvider

// AsyncAPISchema is the alias's own provider, which outranks the inherited
// mapping.
func (mappingProviderAlias) AsyncAPISchema() *spec.Schema {
	return &spec.Schema{Type: "string", Format: "provider"}
}

// mappingRecordAlias is a derived alias of a mapped struct. A named struct is
// hoisted either way, so it derives as a $ref to a component carrying the
// inherited mapping.
type mappingRecordAlias mappingRecord

// mappingCombinator references mappingUUID by name, the way a combinator
// directive names a hoisted member.
type mappingCombinator struct {
	Data any `json:"data" asyncapi:"oneOf=mappingUUID"`
}

var (
	mappingUUIDName          = Name(reflect.TypeFor[mappingUUID]())
	mappingQuantityName      = Name(reflect.TypeFor[mappingQuantity]())
	mappingOrderIDName       = Name(reflect.TypeFor[mappingOrderID]())
	mappingLineIDName        = Name(reflect.TypeFor[mappingLineID]())
	mappingRecordName        = Name(reflect.TypeFor[mappingRecord]())
	mappingProviderName      = Name(reflect.TypeFor[mappingProvider]())
	mappingProviderAliasName = Name(reflect.TypeFor[mappingProviderAlias]())
	mappingRecordAliasName   = Name(reflect.TypeFor[mappingRecordAlias]())
	mappingCombinatorName    = Name(reflect.TypeFor[mappingCombinator]())
)

// isolateMappings restores the registry to its post-init state and puts the
// caller's state back when the test ends. MapSchema, MapAlias and derivation's
// consultation tracking are package-level, so a subtest that registers without
// this leaks into every later test — including the built-in assertions.
func isolateMappings(t *testing.T) {
	t.Helper()

	savedRegistry := maps.Clone(mappingRegistry)
	savedAliases := maps.Clone(aliasRegistry)
	savedConsulted := maps.Clone(consulted)
	savedBuiltins := maps.Clone(builtins)
	savedDeferred := maps.Clone(deferredPayloads)
	t.Cleanup(func() {
		mappingRegistry = savedRegistry
		aliasRegistry = savedAliases
		consulted = savedConsulted
		builtins = savedBuiltins
		deferredPayloads = savedDeferred
	})

	clear(mappingRegistry)
	clear(aliasRegistry)
	clear(consulted)
	clear(builtins)
	clear(deferredPayloads)
	registerBuiltins()
}

// isolated is t.Run with the mapping registries restored around it: every
// subtest starts from the post-init state, so a mapping one subtest registers
// cannot decide another's outcome.
func isolated(t *testing.T, name string, fn func(t *testing.T)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		isolateMappings(t)
		fn(t)
	})
}

// uuidSchema is the mapping the tests register for mappingUUID, and the shape the
// design doc uses for a third-party UUID.
func uuidSchema() *spec.Schema { return &spec.Schema{Type: "string", Format: "uuid"} }

func TestMapSchema(t *testing.T) {
	isolateMappings(t)

	isolated(t, "should_register_the_result_of_the_function", func(t *testing.T) {
		calls := 0
		MapSchema[mappingQuantity](func() *spec.Schema {
			calls++
			return &spec.Schema{Type: "integer"}
		})

		// The function is called once at registration, not lazily per use site.
		assert.Equal(t, 1, calls)

		s := FromType(reflect.TypeFor[mappingQuantity](), nil)
		assert.Equal(t, "integer", s.Type)
	})

	isolated(t, "should_register_a_schema_from_json", func(t *testing.T) {
		require.NoError(t, MapSchemaJSON(mappingQuantityName, `{"type":"string","pattern":"^a$"}`))

		s := FromType(reflect.TypeFor[mappingQuantity](), nil)
		assert.Equal(t, "string", s.Type)
		assert.Equal(t, "^a$", s.Pattern)
	})

	isolated(t, "should_return_error_for_a_json_payload_that_is_not_a_schema", func(t *testing.T) {
		err := MapSchemaJSON(mappingQuantityName, `{"type":1}`)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "mapping "+mappingQuantityName)
	})

	isolated(t, "should_replace_mapping_on_reregistration", func(t *testing.T) {
		MapSchemaByName(mappingQuantityName, &spec.Schema{Type: "integer"})
		MapSchemaByName(mappingQuantityName, &spec.Schema{Type: "string", Format: "uuid"})

		s := FromType(reflect.TypeFor[mappingQuantity](), nil)
		assert.Equal(t, "string", s.Type)
		assert.Equal(t, "uuid", s.Format)
	})

	isolated(t, "should_panic_on_a_nil_schema", func(t *testing.T) {
		assert.Panics(t, func() { MapSchemaByName(mappingQuantityName, nil) })
	})

	isolated(t, "should_panic_on_an_unnamed_type", func(t *testing.T) {
		assert.Panics(
			t,
			func() { MapSchema[[]string](func() *spec.Schema { return &spec.Schema{} }) },
		)
	})

	isolated(t, "should_ignore_empty_and_self_referential_alias_edges", func(t *testing.T) {
		assert.NotPanics(t, func() {
			MapAlias("", mappingUUIDName)
			MapAlias(mappingUUIDName, "")
			MapAlias(mappingUUIDName, mappingUUIDName)
		})
		assert.Empty(t, aliasRegistry)
	})
}

func TestMappingDerivation(t *testing.T) {
	isolateMappings(t)

	isolated(t, "should_map_exact_type_inline", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())

		defs := map[string]*spec.Schema{}
		s := FromType(reflect.TypeFor[mappingUUID](), defs)

		assert.Equal(t, "string", s.Type)
		assert.Equal(t, "uuid", s.Format)
		assert.Empty(t, s.Ref, "a mapped type is inlined, never hoisted")
		assert.Empty(t, defs)
	})

	isolated(t, "should_map_pointer_to_mapped_schema", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())

		s := FromType(reflect.TypeFor[*mappingUUID](), nil)

		assert.Equal(t, "string", s.Type)
		assert.Equal(t, "uuid", s.Format)
	})

	isolated(t, "should_map_transitive_slice_and_map", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())

		slice := FromType(reflect.TypeFor[[]mappingUUID](), nil)
		require.NotNil(t, slice.Items)
		assert.Equal(t, "array", slice.Type)
		assert.Equal(t, "uuid", slice.Items.Format)

		m := FromType(reflect.TypeFor[map[string]mappingUUID](), nil)
		require.NotNil(t, m.AdditionalProperties)
		assert.Equal(t, "object", m.Type)
		assert.Equal(t, "uuid", m.AdditionalProperties.Format)
	})

	isolated(t, "should_map_inside_an_anonymous_struct", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())

		anon := reflect.StructOf([]reflect.StructField{{
			Name: "ID",
			Type: reflect.TypeFor[mappingUUID](),
			Tag:  reflect.StructTag(`json:"id"`),
		}})
		s := FromType(anon, map[string]*spec.Schema{})

		require.Contains(t, s.Properties, "id")
		assert.Equal(t, "uuid", s.Properties["id"].Format)
	})

	isolated(t, "should_win_over_schema_provider", func(t *testing.T) {
		MapSchemaByName(mappingProviderName, &spec.Schema{Type: "integer"})

		defs := map[string]*spec.Schema{}
		s := FromType(reflect.TypeFor[mappingProvider](), defs)

		// The mapping outranks the provider: it is the document author's override.
		assert.Equal(t, "integer", s.Type)
		assert.NotContains(t, defs, mappingProviderName)
	})

	isolated(t, "should_inline_mapped_named_struct_without_hoisting", func(t *testing.T) {
		MapSchemaByName(mappingRecordName, &spec.Schema{Type: "string", Format: "csv"})

		defs := map[string]*spec.Schema{}
		s := FromType(reflect.TypeFor[mappingRecord](), defs)

		assert.Equal(t, "string", s.Type)
		assert.Empty(t, s.Properties, "a mapped struct's fields are never derived")
		assert.NotContains(t, defs, mappingRecordName)
	})

	isolated(t, "should_derive_byte_slice_unmapped", func(t *testing.T) {
		s := FromType(reflect.TypeFor[mappingRaw](), nil)

		assert.Equal(t, "string", s.Type)
		assert.Equal(t, "byte", s.Format)
		assert.Equal(t, "base64", s.ContentEncoding)
	})

	isolated(t, "should_derive_time_as_builtin_mapping", func(t *testing.T) {
		defs := map[string]*spec.Schema{}
		s := FromType(reflect.TypeFor[time.Time](), defs)

		assert.Equal(t, "string", s.Type)
		assert.Equal(t, "date-time", s.Format)
		assert.Empty(t, s.Ref)
		assert.Empty(t, defs)
	})

	isolated(t, "should_override_builtin_mapping", func(t *testing.T) {
		MapSchemaByName("time.Time", &spec.Schema{Type: "integer", Format: "unix"})

		s := FromType(reflect.TypeFor[time.Time](), nil)

		assert.Equal(t, "integer", s.Type)
		assert.Equal(t, "unix", s.Format)
	})

	isolated(t, "should_clone_mapped_schema_per_consultation", func(t *testing.T) {
		MapSchemaByName(mappingQuantityName, &spec.Schema{
			Type: "string",
			Enum: []any{"a", "b"},
		})

		first := FromType(reflect.TypeFor[mappingQuantity](), nil)
		require.Len(t, first.Enum, 2)
		first.Enum[0] = "mutated"
		first.Format = "mutated"

		second := FromType(reflect.TypeFor[mappingQuantity](), nil)
		assert.Equal(t, "a", second.Enum[0], "a use site must not corrupt the registry")
		assert.Empty(t, second.Format)
		assert.Equal(t, "a", mappingRegistry[mappingQuantityName].Enum[0])
	})
}

func TestMappingFinalize(t *testing.T) {
	isolateMappings(t)
	restoreRegistry(t)

	isolated(t, "should_hoist_registered_mapped_type_in_finalize", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())
		Register(mappingUUID{})

		doc := spec.New()
		Finalize(doc)

		require.Contains(t, doc.Components.Schemas, mappingUUIDName)
		hoisted := doc.Components.Schemas[mappingUUIDName]
		assert.Equal(t, "string", hoisted.Type)
		assert.Equal(t, "uuid", hoisted.Format)
	})

	isolated(t, "should_resolve_combinator_ref_to_registered_mapped_type", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())
		Register(mappingUUID{})

		defs := map[string]*spec.Schema{}
		FromType(reflect.TypeFor[mappingCombinator](), defs)
		require.Contains(t, defs[mappingCombinatorName].Properties, "data")
		members := defs[mappingCombinatorName].Properties["data"].OneOf
		require.Len(t, members, 1)
		assert.Equal(t, RefByName(mappingUUIDName), members[0].Ref)

		doc := spec.New()
		Finalize(doc)

		// The directive's $ref has to resolve, and to the mapping rather than to
		// an array of integers.
		require.Contains(t, doc.Components.Schemas, mappingUUIDName)
		assert.Equal(t, "string", doc.Components.Schemas[mappingUUIDName].Type)
	})

	isolated(t, "should_hoist_alias_mapping_for_combinator_ref", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())
		MapAlias(mappingOrderIDName, mappingUUIDName)
		Register(mappingOrderID{})

		doc := spec.New()
		Finalize(doc)

		require.Contains(t, doc.Components.Schemas, mappingOrderIDName)
		assert.Equal(t, "uuid", doc.Components.Schemas[mappingOrderIDName].Format)
	})
}

func TestMappingAliases(t *testing.T) {
	isolateMappings(t)

	isolated(t, "should_follow_alias_chain_to_mapping", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())
		MapAlias(mappingOrderIDName, mappingUUIDName)

		s := FromType(reflect.TypeFor[mappingOrderID](), map[string]*spec.Schema{})

		assert.Equal(t, "string", s.Type)
		assert.Equal(t, "uuid", s.Format)
	})

	isolated(t, "should_prefer_alias_own_mapping_over_inherited", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())
		MapSchemaByName(mappingOrderIDName, &spec.Schema{Type: "integer"})
		MapAlias(mappingOrderIDName, mappingUUIDName)

		s := FromType(reflect.TypeFor[mappingOrderID](), nil)

		assert.Equal(t, "integer", s.Type)
	})

	isolated(t, "should_follow_transitive_alias_chain", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())
		MapAlias(mappingOrderIDName, mappingUUIDName)
		MapAlias(mappingLineIDName, mappingOrderIDName)

		s := FromType(reflect.TypeFor[mappingLineID](), nil)

		assert.Equal(t, "uuid", s.Format)
	})

	isolated(t, "should_apply_a_mapping_on_an_intermediate_alias", func(t *testing.T) {
		MapSchemaByName(mappingOrderIDName, &spec.Schema{Type: "integer"})
		MapAlias(mappingLineIDName, mappingOrderIDName)

		s := FromType(reflect.TypeFor[mappingLineID](), nil)

		// One edge per declaration, resolved at derivation time: a mapping on the
		// intermediate alias applies to the types derived from it.
		assert.Equal(t, "integer", s.Type)
	})

	isolated(t, "should_stop_alias_chain_at_missing_edge", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())

		// No edge for the alias: it derives by kind, exactly like the base would
		// without its mapping.
		s := FromType(reflect.TypeFor[mappingOrderID](), map[string]*spec.Schema{})

		assert.Equal(t, "array", s.Type)
		require.NotNil(t, s.Items)
		assert.Equal(t, "integer", s.Items.Type)
	})

	isolated(t, "should_stop_on_a_cyclic_alias_chain", func(t *testing.T) {
		MapAlias(mappingUUIDName, mappingQuantityName)
		MapAlias(mappingQuantityName, mappingUUIDName)

		assert.NotPanics(t, func() {
			s := FromType(reflect.TypeFor[mappingUUID](), nil)
			assert.Equal(t, "array", s.Type)
		})
	})

	isolated(t, "should_prefer_a_provider_over_an_inherited_mapping", func(t *testing.T) {
		MapSchemaByName(mappingProviderName, &spec.Schema{Type: "integer"})
		MapAlias(mappingProviderAliasName, mappingProviderName)

		defs := map[string]*spec.Schema{}
		s := FromType(reflect.TypeFor[mappingProviderAlias](), defs)

		// Inside hoistStruct the provider is consulted before the alias chain: the
		// schema written for the alias wins, and it is hoisted under the alias.
		assert.Equal(t, RefByName(mappingProviderAliasName), s.Ref)
		require.Contains(t, defs, mappingProviderAliasName)
		assert.Equal(t, "provider", defs[mappingProviderAliasName].Format)
	})

	isolated(t, "should_hoist_an_inherited_mapping_for_a_struct_alias", func(t *testing.T) {
		MapSchemaByName(mappingRecordName, &spec.Schema{Type: "string", Format: "csv"})
		MapAlias(mappingRecordAliasName, mappingRecordName)

		defs := map[string]*spec.Schema{}
		s := FromType(reflect.TypeFor[mappingRecordAlias](), defs)

		assert.Equal(t, RefByName(mappingRecordAliasName), s.Ref)
		require.Contains(t, defs, mappingRecordAliasName)
		assert.Equal(t, "csv", defs[mappingRecordAliasName].Format)
	})
}

func TestMappingWarnings(t *testing.T) {
	isolateMappings(t)

	isolated(t, "should_report_mapping_warnings", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())
		MapSchemaByName(mappingQuantityName, &spec.Schema{Type: "integer"})

		assert.Equal(t, []string{mappingQuantityName, mappingUUIDName}, MappingWarnings())
	})

	isolated(t, "should_clear_a_warning_once_the_mapping_is_consulted", func(t *testing.T) {
		MapSchemaByName(mappingUUIDName, uuidSchema())
		MapSchemaByName(mappingQuantityName, &spec.Schema{Type: "integer"})

		FromType(reflect.TypeFor[mappingUUID](), nil)

		assert.Equal(t, []string{mappingQuantityName}, MappingWarnings())
	})

	isolated(t, "should_exclude_builtins_from_mapping_warnings", func(t *testing.T) {
		// Built-ins are registered but most modules use none of them, so warning
		// about every unconsumed one would fire on essentially every run.
		assert.Empty(t, MappingWarnings())
	})

	isolated(t, "should_warn_about_an_unused_override_of_a_builtin", func(t *testing.T) {
		MapSchemaByName("time.Time", &spec.Schema{Type: "integer"})

		// The override is a user mapping now: unconsulted, so it warns.
		assert.Contains(t, MappingWarnings(), "time.Time")
	})
}

func TestBuiltinMappings(t *testing.T) {
	isolateMappings(t)

	// Each entry is asserted through the registry rather than through FromType:
	// asyncgo does not import these libraries (that is what keeps the table
	// dependency-free), so no reflect.Type with those names exists in this
	// process. The embedded table is the contract; mappingUUID exercises the same
	// path with a type this package owns, and the mapping fixture exercises a
	// real stdlib one end to end.
	tests := []struct {
		name    string
		fqn     string
		want    string
		format  string
		pattern string
	}{
		{
			name:   "should_map_gofrs_uuid_to_string_format_uuid",
			fqn:    "github.com/gofrs/uuid.UUID",
			want:   "string",
			format: "uuid",
		},
		{
			// v5 moved the module to the /v5 path. Lookup is exact-name, so the
			// unsuffixed entry above does not cover it and a missing row here would
			// silently downgrade uuid.UUID to an array of integers.
			name:   "should_map_gofrs_uuid_v5_to_string_format_uuid",
			fqn:    "github.com/gofrs/uuid/v5.UUID",
			want:   "string",
			format: "uuid",
		},
		{
			name:   "should_map_google_uuid_to_string_format_uuid",
			fqn:    "github.com/google/uuid.UUID",
			want:   "string",
			format: "uuid",
		},
		{
			name:    "should_map_ulid_with_crockford_pattern",
			fqn:     "github.com/oklog/ulid/v2.ULID",
			want:    "string",
			pattern: "^[0-9ABCDEFGHJKMNPQRSTVWXYZ]{26}$",
		},
		{
			name:    "should_map_decimal_to_string",
			fqn:     "github.com/shopspring/decimal.Decimal",
			want:    "string",
			pattern: `^-?\d+(\.\d+)?$`,
		},
		{
			name: "should_map_big_int_to_integer",
			fqn:  "math/big.Int",
			want: "integer",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Contains(t, mappingRegistry, tc.fqn)
			s := mappingRegistry[tc.fqn]
			assert.Equal(t, tc.want, s.Type)
			assert.Equal(t, tc.format, s.Format)
			assert.Equal(t, tc.pattern, s.Pattern)
		})
	}

	isolated(t, "should_flag_every_table_entry_as_builtin", func(t *testing.T) {
		for _, tc := range tests {
			assert.True(t, builtins[tc.fqn], "%s must be excluded from warnings", tc.fqn)
		}
	})

	isolated(t, "should_map_big_int_through_derivation", func(t *testing.T) {
		defs := map[string]*spec.Schema{}
		s := FromType(reflect.TypeFor[*big.Int](), defs)

		// big.Int is a struct with unexported fields: kind derivation would give an
		// empty object, and encoding/json marshals it as a bare JSON number.
		assert.Equal(t, "integer", s.Type)
		assert.Empty(t, defs)
	})

	isolated(t, "should_name_the_time_mapping_as_builtin", func(t *testing.T) {
		assert.True(t, builtins["time.Time"])
	})
}
