package golang_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnionJSONRoundTripMultipleVariants validates marshal/unmarshal with 3+ variants
func TestUnionJSONRoundTripMultipleVariants(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
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
`)

	result, err := schema.Convert(openapi, schema.ConvertOptions{
		GoPackagePath: "test/types",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto",
	})
	require.NoError(t, err)

	tmpDir := t.TempDir()

	typesDir := filepath.Join(tmpDir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(typesDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	testProg := `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"test/types"
)

func main() {
	// Test Dog variant
	dogJSON := []byte(` + "`" + `{"petType":"dog","bark":"woof"}` + "`" + `)
	var pet1 types.Pet
	if err := json.Unmarshal(dogJSON, &pet1); err != nil {
		fmt.Fprintf(os.Stderr, "dog unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if pet1.Dog == nil {
		fmt.Fprintf(os.Stderr, "expected Dog to be set\n")
		os.Exit(1)
	}
	if pet1.Dog.Bark != "woof" {
		fmt.Fprintf(os.Stderr, "expected bark=woof, got %s\n", pet1.Dog.Bark)
		os.Exit(1)
	}

	// Test Cat variant
	catJSON := []byte(` + "`" + `{"petType":"cat","meow":"purr"}` + "`" + `)
	var pet2 types.Pet
	if err := json.Unmarshal(catJSON, &pet2); err != nil {
		fmt.Fprintf(os.Stderr, "cat unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if pet2.Cat == nil {
		fmt.Fprintf(os.Stderr, "expected Cat to be set\n")
		os.Exit(1)
	}
	if pet2.Cat.Meow != "purr" {
		fmt.Fprintf(os.Stderr, "expected meow=purr, got %s\n", pet2.Cat.Meow)
		os.Exit(1)
	}

	// Test Bird variant
	birdJSON := []byte(` + "`" + `{"petType":"bird","chirp":"tweet"}` + "`" + `)
	var pet3 types.Pet
	if err := json.Unmarshal(birdJSON, &pet3); err != nil {
		fmt.Fprintf(os.Stderr, "bird unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if pet3.Bird == nil {
		fmt.Fprintf(os.Stderr, "expected Bird to be set\n")
		os.Exit(1)
	}
	if pet3.Bird.Chirp != "tweet" {
		fmt.Fprintf(os.Stderr, "expected chirp=tweet, got %s\n", pet3.Bird.Chirp)
		os.Exit(1)
	}

	// Test marshal Dog
	marshaled1, err := json.Marshal(&pet1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dog marshal error: %v\n", err)
		os.Exit(1)
	}
	var dogMap map[string]interface{}
	json.Unmarshal(marshaled1, &dogMap)
	if dogMap["petType"] != "dog" || dogMap["bark"] != "woof" {
		fmt.Fprintf(os.Stderr, "dog marshal incorrect: %s\n", string(marshaled1))
		os.Exit(1)
	}

	// Test marshal Cat
	marshaled2, err := json.Marshal(&pet2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cat marshal error: %v\n", err)
		os.Exit(1)
	}
	var catMap map[string]interface{}
	json.Unmarshal(marshaled2, &catMap)
	if catMap["petType"] != "cat" || catMap["meow"] != "purr" {
		fmt.Fprintf(os.Stderr, "cat marshal incorrect: %s\n", string(marshaled2))
		os.Exit(1)
	}

	// Test marshal Bird
	marshaled3, err := json.Marshal(&pet3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bird marshal error: %v\n", err)
		os.Exit(1)
	}
	var birdMap map[string]interface{}
	json.Unmarshal(marshaled3, &birdMap)
	if birdMap["petType"] != "bird" || birdMap["chirp"] != "tweet" {
		fmt.Fprintf(os.Stderr, "bird marshal incorrect: %s\n", string(marshaled3))
		os.Exit(1)
	}

	fmt.Println("OK")
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte(testProg), 0644)
	require.NoError(t, err)

	modFile := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(modFile, []byte("module test\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "test program failed:\n%s", string(output))
	assert.Contains(t, string(output), "OK")
}

// TestUnionJSONCaseInsensitive validates case-insensitive discriminator matching
func TestUnionJSONCaseInsensitive(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`)

	result, err := schema.Convert(openapi, schema.ConvertOptions{
		GoPackagePath: "test/types",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto",
	})
	require.NoError(t, err)

	tmpDir := t.TempDir()

	typesDir := filepath.Join(tmpDir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(typesDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	testProg := `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"test/types"
)

func main() {
	// Test lowercase "dog"
	dogJSON1 := []byte(` + "`" + `{"petType":"dog","bark":"woof"}` + "`" + `)
	var pet1 types.Pet
	if err := json.Unmarshal(dogJSON1, &pet1); err != nil {
		fmt.Fprintf(os.Stderr, "lowercase dog unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if pet1.Dog == nil {
		fmt.Fprintf(os.Stderr, "lowercase dog: expected Dog to be set\n")
		os.Exit(1)
	}

	// Test uppercase "DOG"
	dogJSON2 := []byte(` + "`" + `{"petType":"DOG","bark":"woof"}` + "`" + `)
	var pet2 types.Pet
	if err := json.Unmarshal(dogJSON2, &pet2); err != nil {
		fmt.Fprintf(os.Stderr, "uppercase DOG unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if pet2.Dog == nil {
		fmt.Fprintf(os.Stderr, "uppercase DOG: expected Dog to be set\n")
		os.Exit(1)
	}

	// Test mixed case "DoG"
	dogJSON3 := []byte(` + "`" + `{"petType":"DoG","bark":"woof"}` + "`" + `)
	var pet3 types.Pet
	if err := json.Unmarshal(dogJSON3, &pet3); err != nil {
		fmt.Fprintf(os.Stderr, "mixed DoG unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if pet3.Dog == nil {
		fmt.Fprintf(os.Stderr, "mixed DoG: expected Dog to be set\n")
		os.Exit(1)
	}

	// Test Cat with different cases
	catJSON1 := []byte(` + "`" + `{"petType":"cat","meow":"purr"}` + "`" + `)
	var pet4 types.Pet
	if err := json.Unmarshal(catJSON1, &pet4); err != nil {
		fmt.Fprintf(os.Stderr, "lowercase cat unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if pet4.Cat == nil {
		fmt.Fprintf(os.Stderr, "lowercase cat: expected Cat to be set\n")
		os.Exit(1)
	}

	catJSON2 := []byte(` + "`" + `{"petType":"CAT","meow":"purr"}` + "`" + `)
	var pet5 types.Pet
	if err := json.Unmarshal(catJSON2, &pet5); err != nil {
		fmt.Fprintf(os.Stderr, "uppercase CAT unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if pet5.Cat == nil {
		fmt.Fprintf(os.Stderr, "uppercase CAT: expected Cat to be set\n")
		os.Exit(1)
	}

	fmt.Println("OK")
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte(testProg), 0644)
	require.NoError(t, err)

	modFile := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(modFile, []byte("module test\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "test program failed:\n%s", string(output))
	assert.Contains(t, string(output), "OK")
}

// TestUnionJSONUnknownDiscriminator validates error on unknown discriminator value
func TestUnionJSONUnknownDiscriminator(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`)

	result, err := schema.Convert(openapi, schema.ConvertOptions{
		GoPackagePath: "test/types",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto",
	})
	require.NoError(t, err)

	tmpDir := t.TempDir()

	typesDir := filepath.Join(tmpDir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(typesDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	testProg := `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"test/types"
)

func main() {
	// Test unknown discriminator value "bird"
	unknownJSON := []byte(` + "`" + `{"petType":"bird","chirp":"tweet"}` + "`" + `)
	var pet types.Pet
	err := json.Unmarshal(unknownJSON, &pet)
	if err == nil {
		fmt.Fprintf(os.Stderr, "expected error for unknown discriminator, got nil\n")
		os.Exit(1)
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "unknown") {
		fmt.Fprintf(os.Stderr, "error should contain 'unknown', got: %s\n", errMsg)
		os.Exit(1)
	}
	if !strings.Contains(errMsg, "petType") {
		fmt.Fprintf(os.Stderr, "error should mention discriminator field 'petType', got: %s\n", errMsg)
		os.Exit(1)
	}

	fmt.Println("OK")
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte(testProg), 0644)
	require.NoError(t, err)

	modFile := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(modFile, []byte("module test\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "test program failed:\n%s", string(output))
	assert.Contains(t, string(output), "OK")
}

// TestUnionJSONEmptyVariant validates error when no variant is set
func TestUnionJSONEmptyVariant(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
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
        bark:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: string
`)

	result, err := schema.Convert(openapi, schema.ConvertOptions{
		GoPackagePath: "test/types",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto",
	})
	require.NoError(t, err)

	tmpDir := t.TempDir()

	typesDir := filepath.Join(tmpDir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(typesDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	testProg := `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"test/types"
)

func main() {
	// Create empty Pet with no variant set
	var pet types.Pet
	_, err := json.Marshal(&pet)
	if err == nil {
		fmt.Fprintf(os.Stderr, "expected error for empty variant, got nil\n")
		os.Exit(1)
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Pet") {
		fmt.Fprintf(os.Stderr, "error should mention type 'Pet', got: %s\n", errMsg)
		os.Exit(1)
	}
	if !strings.Contains(errMsg, "no variant") {
		fmt.Fprintf(os.Stderr, "error should mention 'no variant', got: %s\n", errMsg)
		os.Exit(1)
	}

	fmt.Println("OK")
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte(testProg), 0644)
	require.NoError(t, err)

	modFile := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(modFile, []byte("module test\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "test program failed:\n%s", string(output))
	assert.Contains(t, string(output), "OK")
}

// TestUnionJSONNestedObjects validates unions with nested object variants
func TestUnionJSONNestedObjects(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
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
        cvv:
          type: string
    BankTransfer:
      type: object
      properties:
        paymentType:
          type: string
        accountNumber:
          type: string
        routingNumber:
          type: string
`)

	result, err := schema.Convert(openapi, schema.ConvertOptions{
		GoPackagePath: "test/types",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto",
	})
	require.NoError(t, err)

	tmpDir := t.TempDir()

	typesDir := filepath.Join(tmpDir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(typesDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	testProg := `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"test/types"
)

func main() {
	// Test CreditCard variant with multiple fields
	ccJSON := []byte(` + "`" + `{"paymentType":"creditcard","cardNumber":"1234-5678-9012-3456","cvv":"123"}` + "`" + `)
	var payment1 types.Payment
	if err := json.Unmarshal(ccJSON, &payment1); err != nil {
		fmt.Fprintf(os.Stderr, "creditcard unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if payment1.CreditCard == nil {
		fmt.Fprintf(os.Stderr, "expected CreditCard to be set\n")
		os.Exit(1)
	}
	if payment1.CreditCard.CardNumber != "1234-5678-9012-3456" {
		fmt.Fprintf(os.Stderr, "expected cardNumber=1234-5678-9012-3456, got %s\n", payment1.CreditCard.CardNumber)
		os.Exit(1)
	}
	if payment1.CreditCard.Cvv != "123" {
		fmt.Fprintf(os.Stderr, "expected cvv=123, got %s\n", payment1.CreditCard.Cvv)
		os.Exit(1)
	}

	// Test BankTransfer variant with multiple fields
	btJSON := []byte(` + "`" + `{"paymentType":"banktransfer","accountNumber":"9876543210","routingNumber":"123456789"}` + "`" + `)
	var payment2 types.Payment
	if err := json.Unmarshal(btJSON, &payment2); err != nil {
		fmt.Fprintf(os.Stderr, "banktransfer unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if payment2.BankTransfer == nil {
		fmt.Fprintf(os.Stderr, "expected BankTransfer to be set\n")
		os.Exit(1)
	}
	if payment2.BankTransfer.AccountNumber != "9876543210" {
		fmt.Fprintf(os.Stderr, "expected accountNumber=9876543210, got %s\n", payment2.BankTransfer.AccountNumber)
		os.Exit(1)
	}
	if payment2.BankTransfer.RoutingNumber != "123456789" {
		fmt.Fprintf(os.Stderr, "expected routingNumber=123456789, got %s\n", payment2.BankTransfer.RoutingNumber)
		os.Exit(1)
	}

	// Test marshal CreditCard
	marshaled1, err := json.Marshal(&payment1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creditcard marshal error: %v\n", err)
		os.Exit(1)
	}
	var ccMap map[string]interface{}
	json.Unmarshal(marshaled1, &ccMap)
	if ccMap["paymentType"] != "creditcard" {
		fmt.Fprintf(os.Stderr, "creditcard marshal incorrect paymentType: %s\n", string(marshaled1))
		os.Exit(1)
	}
	if ccMap["cardNumber"] != "1234-5678-9012-3456" {
		fmt.Fprintf(os.Stderr, "creditcard marshal incorrect cardNumber: %s\n", string(marshaled1))
		os.Exit(1)
	}
	if ccMap["cvv"] != "123" {
		fmt.Fprintf(os.Stderr, "creditcard marshal incorrect cvv: %s\n", string(marshaled1))
		os.Exit(1)
	}

	// Test marshal BankTransfer
	marshaled2, err := json.Marshal(&payment2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "banktransfer marshal error: %v\n", err)
		os.Exit(1)
	}
	var btMap map[string]interface{}
	json.Unmarshal(marshaled2, &btMap)
	if btMap["paymentType"] != "banktransfer" {
		fmt.Fprintf(os.Stderr, "banktransfer marshal incorrect paymentType: %s\n", string(marshaled2))
		os.Exit(1)
	}
	if btMap["routingNumber"] != "123456789" {
		fmt.Fprintf(os.Stderr, "banktransfer marshal incorrect routingNumber: %s\n", string(marshaled2))
		os.Exit(1)
	}

	fmt.Println("OK")
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte(testProg), 0644)
	require.NoError(t, err)

	modFile := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(modFile, []byte("module test\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "test program failed:\n%s", string(output))
	assert.Contains(t, string(output), "OK")
}

// TestUnionJSONMultipleFields validates struct with multiple union-typed fields
func TestUnionJSONMultipleFields(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
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
`)

	result, err := schema.Convert(openapi, schema.ConvertOptions{
		GoPackagePath: "test/types",
		PackageName:   "testpkg",
		PackagePath:   "github.com/example/proto",
	})
	require.NoError(t, err)

	tmpDir := t.TempDir()

	typesDir := filepath.Join(tmpDir, "types")
	err = os.MkdirAll(typesDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(typesDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	testProg := `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"test/types"
)

func main() {
	// Test Order with both union fields
	orderJSON := []byte(` + "`" + `{"orderId":"123","payment":{"paymentType":"creditcard","cardNumber":"1234"},"shipping":{"shippingType":"express","deliveryTime":"2 hours"}}` + "`" + `)
	var order types.Order
	if err := json.Unmarshal(orderJSON, &order); err != nil {
		fmt.Fprintf(os.Stderr, "order unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if order.OrderId != "123" {
		fmt.Fprintf(os.Stderr, "expected orderId=123, got %s\n", order.OrderId)
		os.Exit(1)
	}

	// Verify payment union
	if order.Payment == nil {
		fmt.Fprintf(os.Stderr, "expected Payment to be set\n")
		os.Exit(1)
	}
	if order.Payment.CreditCard == nil {
		fmt.Fprintf(os.Stderr, "expected CreditCard to be set\n")
		os.Exit(1)
	}
	if order.Payment.CreditCard.CardNumber != "1234" {
		fmt.Fprintf(os.Stderr, "expected cardNumber=1234, got %s\n", order.Payment.CreditCard.CardNumber)
		os.Exit(1)
	}

	// Verify shipping union
	if order.Shipping == nil {
		fmt.Fprintf(os.Stderr, "expected Shipping to be set\n")
		os.Exit(1)
	}
	if order.Shipping.Express == nil {
		fmt.Fprintf(os.Stderr, "expected Express to be set\n")
		os.Exit(1)
	}
	if order.Shipping.Express.DeliveryTime != "2 hours" {
		fmt.Fprintf(os.Stderr, "expected deliveryTime=2 hours, got %s\n", order.Shipping.Express.DeliveryTime)
		os.Exit(1)
	}

	// Test marshal Order
	marshaled, err := json.Marshal(&order)
	if err != nil {
		fmt.Fprintf(os.Stderr, "order marshal error: %v\n", err)
		os.Exit(1)
	}

	var orderMap map[string]interface{}
	json.Unmarshal(marshaled, &orderMap)
	if orderMap["orderId"] != "123" {
		fmt.Fprintf(os.Stderr, "order marshal incorrect orderId: %s\n", string(marshaled))
		os.Exit(1)
	}

	payment, ok := orderMap["payment"].(map[string]interface{})
	if !ok {
		fmt.Fprintf(os.Stderr, "order marshal missing payment: %s\n", string(marshaled))
		os.Exit(1)
	}
	if payment["paymentType"] != "creditcard" {
		fmt.Fprintf(os.Stderr, "order marshal incorrect paymentType: %s\n", string(marshaled))
		os.Exit(1)
	}

	shipping, ok := orderMap["shipping"].(map[string]interface{})
	if !ok {
		fmt.Fprintf(os.Stderr, "order marshal missing shipping: %s\n", string(marshaled))
		os.Exit(1)
	}
	if shipping["shippingType"] != "express" {
		fmt.Fprintf(os.Stderr, "order marshal incorrect shippingType: %s\n", string(marshaled))
		os.Exit(1)
	}

	fmt.Println("OK")
}
`

	testFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(testFile, []byte(testProg), 0644)
	require.NoError(t, err)

	modFile := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(modFile, []byte("module test\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "test program failed:\n%s", string(output))
	assert.Contains(t, string(output), "OK")
}
