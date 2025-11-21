package internal_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	schema "github.com/duh-rpc/openapi-schema.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationEcommerce validates complete e-commerce domain model
func TestIntegrationEcommerce(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: E-Commerce API
  version: 1.0.0
components:
  schemas:
    PaymentMethod:
      oneOf:
        - $ref: '#/components/schemas/CreditCard'
        - $ref: '#/components/schemas/PayPal'
        - $ref: '#/components/schemas/BankTransfer'
      discriminator:
        propertyName: type
    CreditCard:
      type: object
      properties:
        type:
          type: string
        cardNumber:
          type: string
        expiryDate:
          type: string
        cvv:
          type: string
    PayPal:
      type: object
      properties:
        type:
          type: string
        email:
          type: string
          format: email
    BankTransfer:
      type: object
      properties:
        type:
          type: string
        accountNumber:
          type: string
        routingNumber:
          type: string
    Order:
      type: object
      properties:
        orderId:
          type: string
        totalAmount:
          type: number
          format: double
        paymentMethod:
          $ref: '#/components/schemas/PaymentMethod'
        createdAt:
          type: string
          format: date-time
`)

	result, err := schema.Convert(openapi, schema.ConvertOptions{
		GoPackagePath: "test/ecommerce",
		PackageName:   "ecommerce",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)

	goCode := string(result.Golang)

	assert.Contains(t, goCode, "type PaymentMethod struct")
	assert.Contains(t, goCode, "CreditCard *CreditCard")
	assert.Contains(t, goCode, "PayPal *PayPal")
	assert.Contains(t, goCode, "BankTransfer *BankTransfer")
	assert.Contains(t, goCode, "type Order struct")
	assert.Contains(t, goCode, "PaymentMethod *PaymentMethod")
	assert.Contains(t, goCode, "OrderId string")
	assert.Contains(t, goCode, "TotalAmount float64")

	assert.NotNil(t, result.TypeMap)

	paymentInfo := result.TypeMap["PaymentMethod"]
	require.NotNil(t, paymentInfo)
	assert.Equal(t, schema.TypeLocationGolang, paymentInfo.Location)
	assert.Equal(t, "contains oneOf", paymentInfo.Reason)

	orderInfo := result.TypeMap["Order"]
	require.NotNil(t, orderInfo)
	assert.Equal(t, schema.TypeLocationGolang, orderInfo.Location)
	assert.True(t, strings.Contains(orderInfo.Reason, "references union type"))

	tmpDir := t.TempDir()

	ecommerceDir := filepath.Join(tmpDir, "ecommerce")
	err = os.MkdirAll(ecommerceDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(ecommerceDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	testProg := `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"test/ecommerce"
)

func main() {
	creditCardJSON := []byte(` + "`" + `{"type":"creditcard","cardNumber":"1234-5678-9012-3456","expiryDate":"12/25","cvv":"123"}` + "`" + `)
	var payment ecommerce.PaymentMethod
	if err := json.Unmarshal(creditCardJSON, &payment); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if payment.CreditCard == nil {
		fmt.Fprintf(os.Stderr, "expected CreditCard to be set\n")
		os.Exit(1)
	}
	if payment.CreditCard.CardNumber != "1234-5678-9012-3456" {
		fmt.Fprintf(os.Stderr, "expected cardNumber=1234-5678-9012-3456\n")
		os.Exit(1)
	}
	fmt.Println("SUCCESS")
}
`

	mainFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(mainFile, []byte(testProg), 0644)
	require.NoError(t, err)

	goMod := `module test

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "run", "main.go")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "go run failed: %s", string(output))
	assert.Contains(t, string(output), "SUCCESS")
}

// TestIntegrationLargeSchema validates schema with 25+ interconnected types
func TestIntegrationLargeSchema(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: Large Schema Test
  version: 1.0.0
components:
  schemas:
    Type1:
      type: object
      properties:
        id:
          type: string
        ref2:
          $ref: '#/components/schemas/Type2'
    Type2:
      type: object
      properties:
        id:
          type: string
        ref3:
          $ref: '#/components/schemas/Type3'
    Type3:
      type: object
      properties:
        id:
          type: string
        value:
          type: integer
    Type4:
      type: object
      properties:
        name:
          type: string
        tags:
          type: array
          items:
            type: string
    Type5:
      type: object
      properties:
        enabled:
          type: boolean
        count:
          type: integer
          format: int64
    Type6:
      type: object
      properties:
        price:
          type: number
          format: double
        currency:
          type: string
    Type7:
      type: object
      properties:
        createdAt:
          type: string
          format: date-time
        updatedAt:
          type: string
          format: date-time
    Type8:
      type: object
      properties:
        status:
          type: string
          enum:
            - active
            - inactive
            - pending
    Type9:
      type: object
      properties:
        ref1:
          $ref: '#/components/schemas/Type1'
        ref4:
          $ref: '#/components/schemas/Type4'
    Type10:
      type: object
      properties:
        items:
          type: array
          items:
            $ref: '#/components/schemas/Type5'
    Type11:
      type: object
      properties:
        nested:
          $ref: '#/components/schemas/Type6'
    Type12:
      type: object
      properties:
        flag1:
          type: boolean
        flag2:
          type: boolean
    Type13:
      type: object
      properties:
        data:
          type: string
        metadata:
          $ref: '#/components/schemas/Type7'
    Type14:
      type: object
      properties:
        ref8:
          $ref: '#/components/schemas/Type8'
    Type15:
      type: object
      properties:
        list:
          type: array
          items:
            type: integer
            format: int32
    Type16:
      type: object
      properties:
        ref9:
          $ref: '#/components/schemas/Type9'
        ref10:
          $ref: '#/components/schemas/Type10'
    Type17:
      type: object
      properties:
        value:
          type: number
          format: float
    Type18:
      type: object
      properties:
        ref11:
          $ref: '#/components/schemas/Type11'
    Type19:
      type: object
      properties:
        content:
          type: string
    Type20:
      type: object
      properties:
        ref12:
          $ref: '#/components/schemas/Type12'
        ref13:
          $ref: '#/components/schemas/Type13'
    Type21:
      type: object
      properties:
        identifier:
          type: string
    Type22:
      type: object
      properties:
        ref14:
          $ref: '#/components/schemas/Type14'
    Type23:
      type: object
      properties:
        array:
          type: array
          items:
            $ref: '#/components/schemas/Type15'
    Type24:
      type: object
      properties:
        description:
          type: string
    Type25:
      type: object
      properties:
        ref16:
          $ref: '#/components/schemas/Type16'
        ref17:
          $ref: '#/components/schemas/Type17'
    Type26:
      type: object
      properties:
        final:
          type: string
        ref25:
          $ref: '#/components/schemas/Type25'
`)

	result, err := schema.Convert(openapi, schema.ConvertOptions{
		PackageName: "largepkg",
		PackagePath: "github.com/example/proto/v1",
	})
	require.NoError(t, err)

	protoCode := string(result.Protobuf)

	assert.Contains(t, protoCode, "message Type1")
	assert.Contains(t, protoCode, "message Type10")
	assert.Contains(t, protoCode, "message Type20")
	assert.Contains(t, protoCode, "message Type26")

	assert.NotNil(t, result.TypeMap)
	assert.GreaterOrEqual(t, len(result.TypeMap), 25)

	for i := 1; i <= 26; i++ {
		typeName := fmt.Sprintf("Type%d", i)
		assert.Contains(t, result.TypeMap, typeName)
	}
}

// TestIntegrationUnionsWithNestedAndArrays validates unions + nested objects + arrays together
func TestIntegrationUnionsWithNestedAndArrays(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: Complex Integration
  version: 1.0.0
components:
  schemas:
    Notification:
      oneOf:
        - $ref: '#/components/schemas/EmailNotification'
        - $ref: '#/components/schemas/SmsNotification'
        - $ref: '#/components/schemas/PushNotification'
      discriminator:
        propertyName: notificationType
    EmailNotification:
      type: object
      properties:
        notificationType:
          type: string
        to:
          type: string
          format: email
        subject:
          type: string
        tags:
          type: array
          items:
            type: string
    SmsNotification:
      type: object
      properties:
        notificationType:
          type: string
        phoneNumber:
          type: string
        message:
          type: string
    PushNotification:
      type: object
      properties:
        notificationType:
          type: string
        deviceToken:
          type: string
        title:
          type: string
        body:
          type: string
    Campaign:
      type: object
      properties:
        campaignId:
          type: string
        name:
          type: string
        notifications:
          type: array
          items:
            $ref: '#/components/schemas/Notification'
        tags:
          type: array
          items:
            type: string
`)

	result, err := schema.Convert(openapi, schema.ConvertOptions{
		GoPackagePath: "test/notifications",
		PackageName:   "notifications",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)

	goCode := string(result.Golang)

	assert.Contains(t, goCode, "type Notification struct")
	assert.Contains(t, goCode, "EmailNotification *EmailNotification")
	assert.Contains(t, goCode, "SmsNotification *SmsNotification")
	assert.Contains(t, goCode, "PushNotification *PushNotification")
	assert.Contains(t, goCode, "type EmailNotification struct")
	assert.Contains(t, goCode, "Tags []string")
	assert.Contains(t, goCode, "type Campaign struct")
	assert.Contains(t, goCode, "Notifications []*Notification")

	assert.NotNil(t, result.TypeMap)

	notificationInfo := result.TypeMap["Notification"]
	require.NotNil(t, notificationInfo)
	assert.Equal(t, schema.TypeLocationGolang, notificationInfo.Location)
	assert.Equal(t, "contains oneOf", notificationInfo.Reason)

	campaignInfo := result.TypeMap["Campaign"]
	require.NotNil(t, campaignInfo)
	assert.Equal(t, schema.TypeLocationGolang, campaignInfo.Location)
	assert.True(t, strings.Contains(campaignInfo.Reason, "references union type"))

	tmpDir := t.TempDir()

	notificationsDir := filepath.Join(tmpDir, "notifications")
	err = os.MkdirAll(notificationsDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(notificationsDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	testProg := `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"test/notifications"
)

func main() {
	emailJSON := []byte(` + "`" + `{"notificationType":"emailnotification","to":"test@example.com","subject":"Test","body":"Hello","attachments":[]}` + "`" + `)
	var notification notifications.Notification
	if err := json.Unmarshal(emailJSON, &notification); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal error: %v\n", err)
		os.Exit(1)
	}
	if notification.EmailNotification == nil {
		fmt.Fprintf(os.Stderr, "expected EmailNotification to be set\n")
		os.Exit(1)
	}
	if notification.EmailNotification.To != "test@example.com" {
		fmt.Fprintf(os.Stderr, "expected to=test@example.com\n")
		os.Exit(1)
	}
	fmt.Println("SUCCESS")
}
`

	mainFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(mainFile, []byte(testProg), 0644)
	require.NoError(t, err)

	goMod := `module test

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "run", "main.go")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "go run failed: %s", string(output))
	assert.Contains(t, string(output), "SUCCESS")
}

// TestIntegrationMultipleUnionsWithReferences validates complex reference chains with unions
func TestIntegrationMultipleUnionsWithReferences(t *testing.T) {
	openapi := []byte(`openapi: 3.0.0
info:
  title: Multiple Unions Test
  version: 1.0.0
components:
  schemas:
    Storage:
      oneOf:
        - $ref: '#/components/schemas/LocalStorage'
        - $ref: '#/components/schemas/CloudStorage'
      discriminator:
        propertyName: storageType
    LocalStorage:
      type: object
      properties:
        storageType:
          type: string
        path:
          type: string
        capacity:
          type: integer
          format: int64
    CloudStorage:
      type: object
      properties:
        storageType:
          type: string
        provider:
          type: string
        bucket:
          type: string
        region:
          type: string
    Authentication:
      oneOf:
        - $ref: '#/components/schemas/ApiKeyAuth'
        - $ref: '#/components/schemas/OAuth2Auth'
      discriminator:
        propertyName: authType
    ApiKeyAuth:
      type: object
      properties:
        authType:
          type: string
        apiKey:
          type: string
    OAuth2Auth:
      type: object
      properties:
        authType:
          type: string
        clientId:
          type: string
        clientSecret:
          type: string
        tokenUrl:
          type: string
    ServiceConfig:
      type: object
      properties:
        configId:
          type: string
        storage:
          $ref: '#/components/schemas/Storage'
        authentication:
          $ref: '#/components/schemas/Authentication'
        backupStorage:
          $ref: '#/components/schemas/Storage'
    DeploymentPlan:
      type: object
      properties:
        planId:
          type: string
        services:
          type: array
          items:
            $ref: '#/components/schemas/ServiceConfig'
        defaultAuth:
          $ref: '#/components/schemas/Authentication'
    Organization:
      type: object
      properties:
        orgId:
          type: string
        name:
          type: string
        deployments:
          type: array
          items:
            $ref: '#/components/schemas/DeploymentPlan'
`)

	result, err := schema.Convert(openapi, schema.ConvertOptions{
		GoPackagePath: "test/config",
		PackageName:   "config",
		PackagePath:   "github.com/example/proto/v1",
	})
	require.NoError(t, err)

	goCode := string(result.Golang)

	assert.Contains(t, goCode, "type Storage struct")
	assert.Contains(t, goCode, "LocalStorage *LocalStorage")
	assert.Contains(t, goCode, "CloudStorage *CloudStorage")
	assert.Contains(t, goCode, "type Authentication struct")
	assert.Contains(t, goCode, "ApiKeyAuth *ApiKeyAuth")
	assert.Contains(t, goCode, "OAuth2Auth *OAuth2Auth")
	assert.Contains(t, goCode, "type ServiceConfig struct")
	assert.Contains(t, goCode, "Storage *Storage")
	assert.Contains(t, goCode, "Authentication *Authentication")
	assert.Contains(t, goCode, "BackupStorage *Storage")
	assert.Contains(t, goCode, "type DeploymentPlan struct")
	assert.Contains(t, goCode, "Services []*ServiceConfig")
	assert.Contains(t, goCode, "DefaultAuth *Authentication")
	assert.Contains(t, goCode, "type Organization struct")
	assert.Contains(t, goCode, "Deployments []*DeploymentPlan")

	assert.NotNil(t, result.TypeMap)

	storageType := result.TypeMap["Storage"]
	require.NotNil(t, storageType)
	assert.Equal(t, schema.TypeLocationGolang, storageType.Location)

	authType := result.TypeMap["Authentication"]
	require.NotNil(t, authType)
	assert.Equal(t, schema.TypeLocationGolang, authType.Location)

	serviceConfigType := result.TypeMap["ServiceConfig"]
	require.NotNil(t, serviceConfigType)
	assert.Equal(t, schema.TypeLocationGolang, serviceConfigType.Location)
	assert.True(t, strings.Contains(serviceConfigType.Reason, "references union type"))

	deploymentType := result.TypeMap["DeploymentPlan"]
	require.NotNil(t, deploymentType)
	assert.Equal(t, schema.TypeLocationGolang, deploymentType.Location)
	assert.True(t, strings.Contains(deploymentType.Reason, "references union type"))

	orgType := result.TypeMap["Organization"]
	require.NotNil(t, orgType)
	assert.Equal(t, schema.TypeLocationGolang, orgType.Location)
	assert.True(t, strings.Contains(orgType.Reason, "references union type"))

	tmpDir := t.TempDir()

	configDir := filepath.Join(tmpDir, "config")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(configDir, "types.go")
	err = os.WriteFile(goFile, result.Golang, 0644)
	require.NoError(t, err)

	goMod := `module test

go 1.21
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", string(output))
}
