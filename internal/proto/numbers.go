package proto

// FieldNumbers carries an explicit, name-keyed proto field-number assignment that
// overrides the library's positional numbering. When a *FieldNumbers is supplied on
// ConvertOptions it fully drives numbering for any message or enum it has an entry
// for; messages/enums without an entry fall back to positional assignment.
//
// Keys are OpenAPI-native so they join cleanly to what the builder already tracks:
//   - Messages key  → ProtoMessage.OriginalSchema (component schema name)
//   - Fields key     → ProtoField.JSONName (JSON field name)
//   - Enums key      → the enum's OpenAPI schema name
//   - Variants key   → the literal OpenAPI enum value
//
// Limitation: only top-level component schemas are addressable. The fields of an
// inline nested object (an inline `type: object` property) are always numbered
// positionally and cannot be pinned, because an inline nested type has no
// globally unique key — only its property name, which is not unique across
// messages. Use a $ref to a named component schema for any type whose field
// numbers must be wire-stable across regeneration.
type FieldNumbers struct {
	Messages map[string]MessageNumbers
	Enums    map[string]EnumNumbers
}

// MessageNumbers pins a message's field numbers and the numbers retired via removal.
type MessageNumbers struct {
	Fields   map[string]int // JSON field name → proto field number
	Reserved []int          // rendered as `reserved N, M;`
}

// EnumNumbers pins a proto enum's variant numbers and its reserved numbers.
type EnumNumbers struct {
	Variants map[string]int // literal enum value → proto number
	Reserved []int          // rendered as `reserved N, M;`
}
