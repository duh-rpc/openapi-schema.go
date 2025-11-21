package internal_test

import (
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnionThreeVariants validates unions with 3+ variants
func TestUnionThreeVariants(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Bird'
      discriminator:
        propertyName: petType
    Dog:
      type: object
      properties:
        petType:
          type: string
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
    Bird:
      type: object
      properties:
        petType:
          type: string
        chirp:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	// Verify union struct with all three pointer fields
	assert.Contains(t, goCode, "type Pet struct")
	assert.Contains(t, goCode, "Dog *Dog")
	assert.Contains(t, goCode, "Cat *Cat")
	assert.Contains(t, goCode, "Bird *Bird")

	// Verify all three variants are generated
	assert.Contains(t, goCode, "type Dog struct")
	assert.Contains(t, goCode, "type Cat struct")
	assert.Contains(t, goCode, "type Bird struct")

	// Verify MarshalJSON and UnmarshalJSON are generated for Pet
	assert.Contains(t, goCode, "func (u *Pet) MarshalJSON()")
	assert.Contains(t, goCode, "func (u *Pet) UnmarshalJSON(")

	// Verify all types are Go-only
	require.NotNil(t, result.TypeMap)
	petInfo, exists := result.TypeMap["Pet"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, petInfo.Location)

	for _, variant := range []string{"Dog", "Cat", "Bird"} {
		info, exists := result.TypeMap[variant]
		require.True(t, exists, "variant %s should exist", variant)
		assert.Equal(t, schema.TypeLocationGolang, info.Location, "variant %s should be Go-only", variant)
	}
}

// TestUnionNestedInProperty validates union type as object property
func TestUnionNestedInProperty(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Owner:
      type: object
      properties:
        name:
          type: string
        pet:
          $ref: '#/components/schemas/Pet'
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/Cat'
      discriminator:
        propertyName: petType
    Dog:
      type: object
      properties:
        petType:
          type: string
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	// Verify Owner struct has Pet pointer field
	assert.Contains(t, goCode, "type Owner struct")
	assert.Contains(t, goCode, "Pet *Pet")

	// Verify Pet union is generated
	assert.Contains(t, goCode, "type Pet struct")
	assert.Contains(t, goCode, "Dog *Dog")
	assert.Contains(t, goCode, "Cat *Cat")

	// Verify Owner is Go-only due to referencing union
	require.NotNil(t, result.TypeMap)
	ownerInfo, exists := result.TypeMap["Owner"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, ownerInfo.Location)
	assert.Contains(t, ownerInfo.Reason, "references union type")
}

// TestUnionArrayOfUnions validates array containing union types
func TestUnionArrayOfUnions(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Container:
      type: object
      properties:
        name:
          type: string
        pets:
          type: array
          items:
            $ref: '#/components/schemas/Pet'
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/Cat'
      discriminator:
        propertyName: petType
    Dog:
      type: object
      properties:
        petType:
          type: string
        name:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        name:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	// Verify Container has array of Pet pointers
	assert.Contains(t, goCode, "type Container struct")
	assert.Contains(t, goCode, "Pets []*Pet")

	// Verify Pet union is generated
	assert.Contains(t, goCode, "type Pet struct")
	assert.Contains(t, goCode, "Dog *Dog")
	assert.Contains(t, goCode, "Cat *Cat")

	// Verify Container is Go-only
	require.NotNil(t, result.TypeMap)
	containerInfo, exists := result.TypeMap["Container"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, containerInfo.Location)
	assert.Contains(t, containerInfo.Reason, "references union type")
}

// TestUnionMissingDiscriminatorProperty validates error when variant lacks discriminator
func TestUnionMissingDiscriminatorProperty(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/Cat'
      discriminator:
        propertyName: petType
    Dog:
      type: object
      properties:
        petType:
          type: string
        name:
          type: string
    Cat:
      type: object
      properties:
        name:
          type: string
`

	_, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "discriminator property")
	assert.ErrorContains(t, err, "missing in variant")
	assert.ErrorContains(t, err, "Cat")
}

// TestUnionDiscriminatorConflict validates error when discriminator values conflict
func TestUnionDiscriminatorConflict(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/dog'
      discriminator:
        propertyName: petType
    Dog:
      type: object
      properties:
        petType:
          type: string
        bark:
          type: string
    dog:
      type: object
      properties:
        petType:
          type: string
        woof:
          type: string
`

	_, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "discriminator conflict")
}

// TestUnionWithNestedObjects validates union variants containing nested objects
func TestUnionWithNestedObjects(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Payment:
      oneOf:
        - $ref: '#/components/schemas/CreditCard'
        - $ref: '#/components/schemas/BankTransfer'
      discriminator:
        propertyName: paymentType
    CreditCard:
      type: object
      properties:
        paymentType:
          type: string
        cardNumber:
          type: string
        billingAddress:
          $ref: '#/components/schemas/Address'
    BankTransfer:
      type: object
      properties:
        paymentType:
          type: string
        accountNumber:
          type: string
        bank:
          $ref: '#/components/schemas/Bank'
    Address:
      type: object
      properties:
        street:
          type: string
        city:
          type: string
    Bank:
      type: object
      properties:
        name:
          type: string
        routingNumber:
          type: string
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	// Verify Payment union is generated
	assert.Contains(t, goCode, "type Payment struct")
	assert.Contains(t, goCode, "CreditCard *CreditCard")
	assert.Contains(t, goCode, "BankTransfer *BankTransfer")

	// Verify variants with nested objects
	assert.Contains(t, goCode, "type CreditCard struct")
	assert.Contains(t, goCode, "BillingAddress *Address")

	assert.Contains(t, goCode, "type BankTransfer struct")
	assert.Contains(t, goCode, "Bank *Bank")

	// Verify Payment and variants are Go-only
	require.NotNil(t, result.TypeMap)
	paymentInfo, exists := result.TypeMap["Payment"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, paymentInfo.Location)

	for _, variant := range []string{"CreditCard", "BankTransfer"} {
		info, exists := result.TypeMap[variant]
		require.True(t, exists, "variant %s should exist", variant)
		assert.Equal(t, schema.TypeLocationGolang, info.Location, "variant %s should be Go-only", variant)
	}

	// Address and Bank should be Proto-only since they don't reference unions
	// They will be in the proto output, not Go output
	addressInfo, exists := result.TypeMap["Address"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationProto, addressInfo.Location)

	bankInfo, exists := result.TypeMap["Bank"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationProto, bankInfo.Location)

	// Verify nested objects are in proto output, not Go
	protoCode := string(result.Protobuf)
	assert.Contains(t, protoCode, "message Address")
	assert.Contains(t, protoCode, "message Bank")

	// Verify nested objects are NOT in Go output (they're proto-only)
	assert.NotContains(t, goCode, "type Address struct")
	assert.NotContains(t, goCode, "type Bank struct")
}

// TestUnionMultipleUnionFields validates struct with multiple union-typed fields
func TestUnionMultipleUnionFields(t *testing.T) {
	given := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Order:
      type: object
      properties:
        orderId:
          type: string
        payment:
          $ref: '#/components/schemas/Payment'
        shipping:
          $ref: '#/components/schemas/Shipping'
    Payment:
      oneOf:
        - $ref: '#/components/schemas/CreditCard'
        - $ref: '#/components/schemas/Cash'
      discriminator:
        propertyName: paymentType
    Shipping:
      oneOf:
        - $ref: '#/components/schemas/Express'
        - $ref: '#/components/schemas/Standard'
      discriminator:
        propertyName: shippingType
    CreditCard:
      type: object
      properties:
        paymentType:
          type: string
        cardNumber:
          type: string
    Cash:
      type: object
      properties:
        paymentType:
          type: string
        amount:
          type: number
    Express:
      type: object
      properties:
        shippingType:
          type: string
        deliveryTime:
          type: string
    Standard:
      type: object
      properties:
        shippingType:
          type: string
        estimatedDays:
          type: integer
`

	result, err := schema.Convert([]byte(given), schema.ConvertOptions{
		GoPackagePath: "github.com/example/types/v1",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Golang)

	goCode := string(result.Golang)

	// Verify Order has both union fields
	assert.Contains(t, goCode, "type Order struct")
	assert.Contains(t, goCode, "Payment *Payment")
	assert.Contains(t, goCode, "Shipping *Shipping")

	// Verify both Payment and Shipping unions are generated
	assert.Contains(t, goCode, "type Payment struct")
	assert.Contains(t, goCode, "CreditCard *CreditCard")
	assert.Contains(t, goCode, "Cash *Cash")

	assert.Contains(t, goCode, "type Shipping struct")
	assert.Contains(t, goCode, "Express *Express")
	assert.Contains(t, goCode, "Standard *Standard")

	// Verify all variants are generated
	assert.Contains(t, goCode, "type CreditCard struct")
	assert.Contains(t, goCode, "type Cash struct")
	assert.Contains(t, goCode, "type Express struct")
	assert.Contains(t, goCode, "type Standard struct")

	// Verify MarshalJSON for both unions
	assert.Contains(t, goCode, "func (u *Payment) MarshalJSON()")
	assert.Contains(t, goCode, "func (u *Shipping) MarshalJSON()")

	// Verify Order references multiple unions
	require.NotNil(t, result.TypeMap)
	orderInfo, exists := result.TypeMap["Order"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, orderInfo.Location)
	assert.Contains(t, orderInfo.Reason, "references union type")

	// Verify both unions are Go-only
	paymentInfo, exists := result.TypeMap["Payment"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, paymentInfo.Location)

	shippingInfo, exists := result.TypeMap["Shipping"]
	require.True(t, exists)
	assert.Equal(t, schema.TypeLocationGolang, shippingInfo.Location)

	// Verify all variants are Go-only
	for _, variant := range []string{"CreditCard", "Cash", "Express", "Standard"} {
		info, exists := result.TypeMap[variant]
		require.True(t, exists, "variant %s should exist", variant)
		assert.Equal(t, schema.TypeLocationGolang, info.Location, "variant %s should be Go-only", variant)
	}
}
