# Nested Objects

This document explains how `openapi-schema` handles nested object structures in OpenAPI schemas and converts them to Protocol Buffer nested messages.

## Overview

OpenAPI allows defining objects inline as property types. When converting to proto3, these inline objects become nested message definitions within the parent message. This maintains the hierarchical relationship while ensuring the generated protobuf is valid and idiomatic.

## Basic Nested Object

The simplest case is a single level of nesting:

**OpenAPI:**
```yaml
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
        profile:
          type: object
          properties:
            bio:
              type: string
            avatar:
              type: string
```

**Generated Proto3:**
```protobuf
message User {
  message Profile {
    string bio = 1 [json_name = "bio"];
    string avatar = 2 [json_name = "avatar"];
  }

  string name = 1 [json_name = "name"];
  Profile profile = 2 [json_name = "profile"];
}
```

### Key Points:
- The inline object becomes a nested message definition
- Nested message name is derived from the property name using PascalCase
- Nested messages appear **before** the fields that reference them
- Field numbering is independent in each message (nested message starts at 1)

## Deep Nesting

Nesting can go multiple levels deep:

**OpenAPI:**
```yaml
components:
  schemas:
    Company:
      type: object
      properties:
        name:
          type: string
        office:
          type: object
          properties:
            location:
              type: object
              properties:
                street:
                  type: string
                city:
                  type: string
            phone:
              type: string
```

**Generated Proto3:**
```protobuf
message Company {
  message Office {
    message Location {
      string street = 1 [json_name = "street"];
      string city = 2 [json_name = "city"];
    }

    Location location = 1 [json_name = "location"];
    string phone = 2 [json_name = "phone"];
  }

  string name = 1 [json_name = "name"];
  Office office = 2 [json_name = "office"];
}
```

Each level of nesting creates another nested message, maintaining the full hierarchy.

## Multiple Nested Objects

A single parent can have multiple nested objects:

**OpenAPI:**
```yaml
components:
  schemas:
    Profile:
      type: object
      properties:
        billing:
          type: object
          properties:
            card:
              type: string
        shipping:
          type: object
          properties:
            method:
              type: string
```

**Generated Proto3:**
```protobuf
message Profile {
  message Billing {
    string card = 1 [json_name = "card"];
  }

  message Shipping {
    string method = 1 [json_name = "method"];
  }

  Billing billing = 1 [json_name = "billing"];
  Shipping shipping = 2 [json_name = "shipping"];
}
```

Each nested object gets its own message definition within the parent.

## Nested Objects in Arrays

Arrays can contain inline objects, which also become nested messages:

**OpenAPI:**
```yaml
components:
  schemas:
    Company:
      type: object
      properties:
        contact:
          type: array
          items:
            type: object
            properties:
              name:
                type: string
              email:
                type: string
```

**Generated Proto3:**
```protobuf
message Company {
  message Contact {
    string name = 1 [json_name = "name"];
    string email = 2 [json_name = "email"];
  }

  repeated Contact contact = 1 [json_name = "contact"];
}
```

The `repeated` keyword is used for arrays, and the inline object becomes a nested message.

## Descriptions

Descriptions on inline objects are preserved as comments on the nested message:

**OpenAPI:**
```yaml
components:
  schemas:
    User:
      type: object
      description: A user account
      properties:
        name:
          type: string
          description: User's full name
        profile:
          type: object
          description: User's profile information
          properties:
            bio:
              type: string
              description: Biography
```

**Generated Proto3:**
```protobuf
// A user account
message User {
  // User's profile information
  message Profile {
    // Biography
    string bio = 1 [json_name = "bio"];
  }

  // User's full name
  string name = 1 [json_name = "name"];
  Profile profile = 2 [json_name = "profile"];
}
```

**Important:** The description appears on the nested message definition, **not** on the field that references it. This avoids redundancy and follows protobuf conventions.

## Naming Conventions

Nested message names are derived from property names:

| Property Name (camelCase) | Nested Message Name (PascalCase) |
|---------------------------|----------------------------------|
| `profile`                 | `Profile`                        |
| `shippingInfo`            | `ShippingInfo`                   |
| `billingAddress`          | `BillingAddress`                 |
| `office`                  | `Office`                         |

Property names are converted to PascalCase for message names (see main README for naming conventions).

## Plural Name Restriction

To avoid ambiguity in naming, property names ending in 's' or 'es' are **not allowed** for inline objects:

**❌ Not Allowed:**
```yaml
User:
  type: object
  properties:
    contacts:  # Error: ends with 's'
      type: object
      properties:
        phone: { type: string }
```

**Error:**
```
cannot derive message name from property 'contacts'; use singular form or $ref
```

**✅ Solutions:**

1. **Use singular property name:**
```yaml
contact:
  type: object
  properties:
    phone: { type: string }
```

2. **Use array with singular name if multiple items:**
```yaml
contact:
  type: array
  items:
    type: object
    properties:
      phone: { type: string }
```

3. **Use `$ref` to reference a top-level schema:**
```yaml
contacts:
  type: object
  $ref: '#/components/schemas/Contact'
```

This restriction applies to both regular nested objects and array items.

## Field Numbering

Each message (top-level or nested) has **independent** field numbering starting at 1:

```protobuf
message Company {
  message Office {
    string street = 1;  // Starts at 1
    string city = 2;    // Independent numbering
  }

  string name = 1;      // Also starts at 1
  Office office = 2;    // Independent from nested
}
```

This ensures proper protobuf structure where each message has its own field number sequence.

## Message Ordering

Within a parent message, nested messages appear **before** the fields that use them:

```protobuf
message Parent {
  // 1. Nested message definitions first
  message Nested1 {
    string field = 1;
  }

  message Nested2 {
    string field = 1;
  }

  // 2. Then fields that reference them
  Nested1 nested1 = 1;
  Nested2 nested2 = 2;
  string other = 3;
}
```

This ordering is a protobuf best practice and ensures the nested types are defined before use.

## Combining Nested Objects with Other Features

Nested objects work seamlessly with other features:

**With Enums:**
```yaml
User:
  type: object
  properties:
    profile:
      type: object
      properties:
        role:
          type: string
          enum: [admin, user, guest]
```

```protobuf
enum Role {
  ROLE_UNSPECIFIED = 0;
  ROLE_ADMIN = 1;
  ROLE_USER = 2;
  ROLE_GUEST = 3;
}

message User {
  message Profile {
    Role role = 1 [json_name = "role"];
  }

  Profile profile = 1 [json_name = "profile"];
}
```

**With References:**
```yaml
User:
  type: object
  properties:
    profile:
      type: object
      properties:
        address:
          $ref: '#/components/schemas/Address'
```

```protobuf
message Address {
  string street = 1 [json_name = "street"];
}

message User {
  message Profile {
    Address address = 1 [json_name = "address"];
  }

  Profile profile = 1 [json_name = "profile"];
}
```

## When NOT to Use Inline Objects

Consider using top-level schemas with `$ref` instead of inline objects when:

1. **The object is reused** across multiple schemas
2. **The property name is plural** (ends with 's'/'es')
3. **You need fine control** over the proto message name
4. **The object is complex** and deserves top-level visibility

**Instead of inline:**
```yaml
User:
  properties:
    contacts:  # Would error due to plural
      type: object
      properties:
        phone: { type: string }
```

**Use top-level with $ref:**
```yaml
components:
  schemas:
    Contact:
      type: object
      properties:
        phone: { type: string }

    User:
      properties:
        contacts:
          $ref: '#/components/schemas/Contact'
```

## Limitations

The following OpenAPI features are **not supported** with nested objects:

- **Schema composition**: `allOf`, `anyOf`, `oneOf`, `not`
- **Additional properties**: `additionalProperties` (no map support)
- **Pattern properties**: `patternProperties`
- **Property name patterns**: `propertyNames`
- **Conditional schemas**: `if`/`then`/`else`

If your inline object uses these features, you'll receive a clear error message. Consider simplifying the schema or using a top-level definition with `$ref`.

## Best Practices

1. **Use singular names** for inline object properties to avoid plural restrictions
2. **Keep nesting shallow** (1-2 levels) for better readability
3. **Add descriptions** to inline objects for documentation
4. **Consider `$ref`** for reusable or complex objects
5. **Test field numbering** stays independent across messages
6. **Avoid plural names** or use arrays with singular property names

## Related Documentation

- [Enums](enums.md) - Inline enum handling
- [Arrays](arrays.md) - Array field handling
- [References](references.md) - Using `$ref` instead of inline
- [Naming Conventions](naming.md) - Name transformation rules
