# AsyncAPI spec generator from Golang code

## Example 

Following Go code [(full example)](./example/basic/main.go)

```go
package amqp

import "time"

// @title System controller
// @version Mark 1
// @description System management service

type EmergencyCommand struct {
	// Confirmation code, written on a piece of paper under the main boss's keyboard
	ConfirmationCode string `json:"confirmation_code" example:"endgame"`

	Timeout int `json:"timeout" description:"Time in seconds until execution" example:"3" validate:"min=3"`
}

// EmergencyButton asyncApi
// @summary complete data destruction
// @description Deleting all existing data and resetting the system
// @payload amqp.EmergencyCommand
// @queue emergency
// @tags emergency
// @contentType application/json
func EmergencyButton(payload EmergencyCommand) {
	if payload.ConfirmationCode != "1234" {
		panic("incorrect confirmation code")
		return
	}
	time.Sleep(time.Duration(payload.Timeout) * time.Second)
	
	// todo `sudo rm -rf /`
}
```

After executing

```shell
asyncapigo -d ./example/basic --out ./example/basic/asyncapi.yaml
```

transform to...

```yaml
asyncapi: 2.4.0
info:
    title: System controller
    version: Mark 1
    description: System management service
channels:
    emergency:
        publish:
            message:
                oneOf:
                    - $ref: '#/components/messages/emergency.publish.EmergencyButton'
components:
    messages:
        emergency.publish.EmergencyButton:
            tags:
                - name: emergency
            payload:
                $ref: '#/components/schemas/amqp.EmergencyCommand'
            summary: complete data destruction
            description: Deleting all existing data and resetting the system
    schemas:
        amqp.EmergencyCommand:
            type: object
            properties:
                confirmation_code:
                    description: |
                        Confirmation code, written on a piece of paper under the main boss's keyboard
                    example: endgame
                    type: string
                timeout:
                    description: Time in seconds until execution
                    example: 3
                    type: integer
                    minimum: 3
```

## Documentation

### Info

Support following tags

 -  `title` 
 -  `version`
 -  `description`
 -  `termsOfService`
 -  `contact.name`
 -  `contact.url`
 -  `contact.email`
 -  `license.name`
 -  `license.url`

### Handlers

Support following tags

- `summary`
- `description`
- `header` multiple tag
- `tag`,`tags`
- `operation` subscribe / publish (default)
- `queue`
- `payload` type name with package (pkg.Type)
- `contentType`

### Struct field tags

- `validate` - min,max,gt,ls,required,oneOf
- `format`
- `example`
- `description` allowed in field comment too
- `required`

### Header parameters

- `required`
- `type`
- `description`
- `example`
- `format`