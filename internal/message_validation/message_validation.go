package messagevalidation

import (
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

func ValidateMessage(data []byte, schema []byte) error {

	schemaLoader := gojsonschema.NewStringLoader(string(schema))
	dataLoader := gojsonschema.NewStringLoader(string(data))

	result, err := gojsonschema.Validate(schemaLoader, dataLoader)

	if err != nil {
		return err
	}

	if !result.Valid() {
		var errorMessages string
		for _, err := range result.Errors() {
			errorMessages += fmt.Sprintf("- %s\n", err)
		}
		return fmt.Errorf("message failed to validate against schema:\n%s", errorMessages)
	}
	return nil
}
