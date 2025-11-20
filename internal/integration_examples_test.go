package internal_test

import (
	"encoding/json"
	"testing"

	conv "github.com/duh-rpc/openapi-proto.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationComplexSchema validates complete OpenAPI schema with mixed types
func TestIntegrationComplexSchema(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: Complex Schema API
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      enum:
        - active
        - inactive
        - pending
    Address:
      type: object
      properties:
        street:
          type: string
          minLength: 5
          maxLength: 100
        city:
          type: string
          minLength: 2
        zipCode:
          type: string
          pattern: '^\d{5}$'
        country:
          type: string
          default: "USA"
    User:
      type: object
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
          format: email
        age:
          type: integer
          minimum: 18
          maximum: 120
        balance:
          type: number
          minimum: 0.0
          maximum: 1000000.0
        isActive:
          type: boolean
        status:
          $ref: '#/components/schemas/Status'
        address:
          $ref: '#/components/schemas/Address'
        tags:
          type: array
          items:
            type: string
          minItems: 1
          maxItems: 5
        createdAt:
          type: string
          format: date-time
    Product:
      type: object
      properties:
        productId:
          type: string
          format: uuid
        name:
          type: string
          minLength: 3
          maxLength: 50
        price:
          type: number
          minimum: 0.01
        inStock:
          type: boolean
        categories:
          type: array
          items:
            type: string
          minItems: 1
`)

	result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
		IncludeAll: true,
		MaxDepth:   5,
		Seed:       12345,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Examples)

	assert.Contains(t, result.Examples, "Status")
	assert.Contains(t, result.Examples, "Address")
	assert.Contains(t, result.Examples, "User")
	assert.Contains(t, result.Examples, "Product")

	var status string
	err = json.Unmarshal(result.Examples["Status"], &status)
	require.NoError(t, err)
	assert.Equal(t, "active", status)

	var address map[string]interface{}
	err = json.Unmarshal(result.Examples["Address"], &address)
	require.NoError(t, err)
	assert.Contains(t, address, "street")
	assert.Contains(t, address, "city")
	assert.Contains(t, address, "zipCode")
	assert.Contains(t, address, "country")
	assert.Equal(t, "USA", address["country"])

	streetStr := address["street"].(string)
	assert.GreaterOrEqual(t, len(streetStr), 5)
	assert.LessOrEqual(t, len(streetStr), 100)

	cityStr := address["city"].(string)
	assert.GreaterOrEqual(t, len(cityStr), 2)

	var user map[string]interface{}
	err = json.Unmarshal(result.Examples["User"], &user)
	require.NoError(t, err)
	assert.Contains(t, user, "id")
	assert.Contains(t, user, "email")
	assert.Contains(t, user, "age")
	assert.Contains(t, user, "balance")
	assert.Contains(t, user, "isActive")
	assert.Contains(t, user, "status")
	assert.Contains(t, user, "address")
	assert.Contains(t, user, "tags")
	assert.Contains(t, user, "createdAt")

	age := int(user["age"].(float64))
	assert.GreaterOrEqual(t, age, 18)
	assert.LessOrEqual(t, age, 120)

	balance := user["balance"].(float64)
	assert.GreaterOrEqual(t, balance, 0.0)
	assert.LessOrEqual(t, balance, 1000000.0)

	assert.Equal(t, "active", user["status"])

	tags := user["tags"].([]interface{})
	assert.GreaterOrEqual(t, len(tags), 1)
	assert.LessOrEqual(t, len(tags), 5)

	var product map[string]interface{}
	err = json.Unmarshal(result.Examples["Product"], &product)
	require.NoError(t, err)
	assert.Contains(t, product, "productId")
	assert.Contains(t, product, "name")
	assert.Contains(t, product, "price")
	assert.Contains(t, product, "inStock")
	assert.Contains(t, product, "categories")

	price := product["price"].(float64)
	assert.GreaterOrEqual(t, price, 0.01)

	name := product["name"].(string)
	assert.GreaterOrEqual(t, len(name), 3)
	assert.LessOrEqual(t, len(name), 50)

	categories := product["categories"].([]interface{})
	assert.GreaterOrEqual(t, len(categories), 1)
}

// TestIntegrationPetStore validates PetStore-style schema
func TestIntegrationPetStore(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: Pet Store API
  version: 1.0.0
components:
  schemas:
    Category:
      type: object
      properties:
        id:
          type: integer
          format: int64
          minimum: 1
        name:
          type: string
          minLength: 1
          maxLength: 50
    Tag:
      type: object
      properties:
        id:
          type: integer
          format: int64
          minimum: 1
        name:
          type: string
          minLength: 1
    PetStatus:
      type: string
      enum:
        - available
        - pending
        - sold
    Pet:
      type: object
      properties:
        id:
          type: integer
          format: int64
          minimum: 1
        name:
          type: string
          minLength: 1
          maxLength: 100
          example: "Fluffy"
        category:
          $ref: '#/components/schemas/Category'
        photoUrls:
          type: array
          items:
            type: string
            format: uri
          minItems: 1
        tags:
          type: array
          items:
            $ref: '#/components/schemas/Tag'
        status:
          $ref: '#/components/schemas/PetStatus'
    Order:
      type: object
      properties:
        id:
          type: integer
          format: int64
          minimum: 1
        petId:
          type: integer
          format: int64
          minimum: 1
        quantity:
          type: integer
          minimum: 1
          maximum: 100
          default: 1
        shipDate:
          type: string
          format: date-time
        status:
          type: string
          enum:
            - placed
            - approved
            - delivered
        complete:
          type: boolean
`)

	result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
		IncludeAll: true,
		MaxDepth:   5,
		Seed:       67890,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Examples, "Category")
	assert.Contains(t, result.Examples, "Tag")
	assert.Contains(t, result.Examples, "PetStatus")
	assert.Contains(t, result.Examples, "Pet")
	assert.Contains(t, result.Examples, "Order")

	var category map[string]interface{}
	err = json.Unmarshal(result.Examples["Category"], &category)
	require.NoError(t, err)
	assert.Contains(t, category, "id")
	assert.Contains(t, category, "name")

	categoryID := int(category["id"].(float64))
	assert.GreaterOrEqual(t, categoryID, 1)

	var pet map[string]interface{}
	err = json.Unmarshal(result.Examples["Pet"], &pet)
	require.NoError(t, err)
	assert.Contains(t, pet, "id")
	assert.Contains(t, pet, "name")
	assert.Contains(t, pet, "category")
	assert.Contains(t, pet, "photoUrls")
	assert.Contains(t, pet, "tags")
	assert.Contains(t, pet, "status")

	assert.Equal(t, "Fluffy", pet["name"])

	photoUrls := pet["photoUrls"].([]interface{})
	assert.GreaterOrEqual(t, len(photoUrls), 1)

	assert.Equal(t, "available", pet["status"])

	var order map[string]interface{}
	err = json.Unmarshal(result.Examples["Order"], &order)
	require.NoError(t, err)
	assert.Contains(t, order, "id")
	assert.Contains(t, order, "petId")
	assert.Contains(t, order, "quantity")
	assert.Contains(t, order, "shipDate")
	assert.Contains(t, order, "status")
	assert.Contains(t, order, "complete")

	assert.Equal(t, float64(1), order["quantity"])

	quantity := int(order["quantity"].(float64))
	assert.GreaterOrEqual(t, quantity, 1)
	assert.LessOrEqual(t, quantity, 100)

	assert.Equal(t, "placed", order["status"])
}

// TestIntegrationCircularReferences validates circular reference handling
func TestIntegrationCircularReferences(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: Circular Reference API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
          minLength: 1
        email:
          type: string
          format: email
        address:
          $ref: '#/components/schemas/Address'
        friends:
          type: array
          items:
            $ref: '#/components/schemas/User'
    Address:
      type: object
      properties:
        street:
          type: string
          minLength: 1
        city:
          type: string
          minLength: 1
        resident:
          $ref: '#/components/schemas/User'
    Node:
      type: object
      properties:
        value:
          type: string
        left:
          $ref: '#/components/schemas/Node'
        right:
          $ref: '#/components/schemas/Node'
        parent:
          $ref: '#/components/schemas/Node'
`)

	result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
		IncludeAll: true,
		MaxDepth:   3,
		Seed:       11111,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Examples, "User")
	assert.Contains(t, result.Examples, "Address")
	assert.Contains(t, result.Examples, "Node")

	var user map[string]interface{}
	err = json.Unmarshal(result.Examples["User"], &user)
	require.NoError(t, err)
	assert.Contains(t, user, "id")
	assert.Contains(t, user, "name")
	assert.Contains(t, user, "email")

	if address, ok := user["address"]; ok {
		addressMap := address.(map[string]interface{})
		assert.Contains(t, addressMap, "street")
		assert.Contains(t, addressMap, "city")
	}

	var address map[string]interface{}
	err = json.Unmarshal(result.Examples["Address"], &address)
	require.NoError(t, err)
	assert.Contains(t, address, "street")
	assert.Contains(t, address, "city")

	var node map[string]interface{}
	err = json.Unmarshal(result.Examples["Node"], &node)
	require.NoError(t, err)
	assert.Contains(t, node, "value")
}

// TestIntegrationDepthLimit validates depth limit prevents stack overflow
func TestIntegrationDepthLimit(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: Deep Nesting API
  version: 1.0.0
components:
  schemas:
    Level1:
      type: object
      properties:
        data:
          type: string
        level2:
          $ref: '#/components/schemas/Level2'
    Level2:
      type: object
      properties:
        data:
          type: string
        level3:
          $ref: '#/components/schemas/Level3'
    Level3:
      type: object
      properties:
        data:
          type: string
        level4:
          $ref: '#/components/schemas/Level4'
    Level4:
      type: object
      properties:
        data:
          type: string
        level5:
          $ref: '#/components/schemas/Level5'
    Level5:
      type: object
      properties:
        data:
          type: string
        level6:
          $ref: '#/components/schemas/Level6'
    Level6:
      type: object
      properties:
        data:
          type: string
        level7:
          $ref: '#/components/schemas/Level7'
    Level7:
      type: object
      properties:
        data:
          type: string
`)

	result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
		IncludeAll: true,
		MaxDepth:   3,
		Seed:       22222,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Examples, "Level1")

	var level1 map[string]interface{}
	err = json.Unmarshal(result.Examples["Level1"], &level1)
	require.NoError(t, err)
	assert.Contains(t, level1, "data")

	if level2, ok := level1["level2"]; ok {
		level2Map := level2.(map[string]interface{})
		assert.Contains(t, level2Map, "data")

		if level3, ok := level2Map["level3"]; ok {
			level3Map := level3.(map[string]interface{})
			assert.Contains(t, level3Map, "data")
		}
	}

	resultShallow, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
		IncludeAll: true,
		MaxDepth:   1,
		Seed:       22222,
	})
	require.NoError(t, err)
	require.NotNil(t, resultShallow)

	var level1Shallow map[string]interface{}
	err = json.Unmarshal(resultShallow.Examples["Level1"], &level1Shallow)
	require.NoError(t, err)
	assert.Contains(t, level1Shallow, "data")
}

// TestIntegrationConstraintValidation validates all constraints are honored
func TestIntegrationConstraintValidation(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: Constraint Validation API
  version: 1.0.0
components:
  schemas:
    StrictTypes:
      type: object
      properties:
        age:
          type: integer
          minimum: 0
          maximum: 150
        score:
          type: number
          minimum: 0.0
          maximum: 100.0
        username:
          type: string
          minLength: 3
          maxLength: 20
        tags:
          type: array
          items:
            type: string
          minItems: 2
          maxItems: 10
        email:
          type: string
          format: email
        website:
          type: string
          format: uri
        birthday:
          type: string
          format: date
        createdAt:
          type: string
          format: date-time
        userId:
          type: string
          format: uuid
`)

	result, err := conv.ConvertToExamples(openapi, conv.ExampleOptions{
		SchemaNames: []string{"StrictTypes"},
		MaxDepth:    5,
		Seed:        33333,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Examples, "StrictTypes")

	var strict map[string]interface{}
	err = json.Unmarshal(result.Examples["StrictTypes"], &strict)
	require.NoError(t, err)

	age := int(strict["age"].(float64))
	assert.GreaterOrEqual(t, age, 0)
	assert.LessOrEqual(t, age, 150)

	score := strict["score"].(float64)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)

	username := strict["username"].(string)
	assert.GreaterOrEqual(t, len(username), 3)
	assert.LessOrEqual(t, len(username), 20)

	tags := strict["tags"].([]interface{})
	assert.GreaterOrEqual(t, len(tags), 2)
	assert.LessOrEqual(t, len(tags), 10)

	email := strict["email"].(string)
	assert.Contains(t, email, "@")

	website := strict["website"].(string)
	assert.Contains(t, website, "://")

	birthday := strict["birthday"].(string)
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, birthday)

	createdAt := strict["createdAt"].(string)
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}T`, createdAt)

	userId := strict["userId"].(string)
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, userId)
}
