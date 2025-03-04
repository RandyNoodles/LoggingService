/*
* FILE : 			config.go
* FIRST VERSION : 	2025-02-22
* DESCRIPTION :
			Parses config.json, as well as the user-defined message format
		defined in "protocol_settings": "incoming_message_schema".

		Calling ParseConfigFile():
		Returns an error if config or incoming_message_schema are invalid.
		Else returns a pointer to a set of structs containing all parsed config data.
*/

package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed config_schema.json
var configValidationSchema []byte //Embed config schema into binary to avoid user tampering in real-world scenario

// Holds all three config sections from parsed config.json file
type Config struct {
	ServerSettings   ServerSettings   `json:"server_settings"`
	LogfileSettings  LogfileSettings  `json:"logfile_settings"`
	ProtocolSettings ProtocolSettings `json:"protocol_settings"`
	ErrorHandling    ErrorSettings    `json:"error_handling"`
}

// Where to boot up the server
type ServerSettings struct {
	IpAddress string `json:"ip"`
	Port      int    `json:"port"`
}

// Settings for logfile configuration
type LogfileSettings struct {
	Path                    string   `json:"path"`
	Format                  string   `json:"format"`
	PlaintextFieldDelimiter string   `json:"plaintext_field_delimiter"`
	PlaintextEntryDelimiter string   `json:"plaintext_entry_delimiter"`
	ColumnOrder             []string `json:"column_order"`
	TimestampFormat         string   `json:"timestamp_format"`
}

// Settings for Protocol & abuse prevention
type ProtocolSettings struct {
	IncomingMessageSchemaPath    string   `json:"incoming_json_schema"`
	IpMessagesPerMinute          int      `json:"messages_per_ip_per_minute"`
	BadMessageBlacklistThreshold int      `json:"bad_message_blacklist_threshold"`
	BlacklistedIPs               []string `json:"blacklisted_ips"`
	BlacklistPermanent           bool     `json:"blacklist_permanent"`
	BlacklistDurationSeconds     int      `json:"blacklist_duration_seconds"`
	IncomingMessageSchema        []byte
}

// Settings for error handling
type ErrorSettings struct {
	ExtraField     string `json:"extra_field"`
	InvalidMessage string `json:"invalid_message"`
	ErrorLogPath   string `json:"error_log_path"`
}

func ParseConfigFile() (*Config, error) {

	var config Config //Return value

	//Read in config.json
	data, err := os.ReadFile("../config.json")
	if err != nil {
		return nil, err
	}

	//Load the embedded schema for config.json
	schemaLoader := gojsonschema.NewStringLoader(string(configValidationSchema))
	configDataLoader := gojsonschema.NewStringLoader(string(data))

	//Compare config data against embedded config schema
	result, err := gojsonschema.Validate(schemaLoader, configDataLoader)

	if err != nil {
		return nil, err
	}

	//If config.json was valid, parse into structs
	if result.Valid() {
		err = json.Unmarshal(data, &config) // Parse the JSON file into our config structs
		if err != nil {
			fmt.Print(string(data))
			return nil, err
		}
		//Else, return schema validation errors.
	} else {
		var errorMessages string
		for _, err := range result.Errors() {
			errorMessages += fmt.Sprintf("- %s\n", err)
		}
		return nil, fmt.Errorf("config.json failed to validate against schema: \n%s", errorMessages)
	}

	//Parse incoming_message_schema.json
	err = config.parseIncomingMessageSchema()
	if err != nil {
		return nil, err
	}

	return &config, err
}

// Parse incoming_message_schema.json found in ProtocolSettings
func (obj *Config) parseIncomingMessageSchema() error {

	data, err := os.ReadFile(obj.ProtocolSettings.IncomingMessageSchemaPath)

	if err != nil {
		return err
	}

	//Load incoming_message_schema to an object for validation.
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return err
	}

	//Ensure "properties" object is present & populated.
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return errors.New(`"properties" key not found in the incoming_message_schema.json file`)
	}
	if len(props) == 0 {
		return errors.New(`"properties" object in incoming_message_schema.json file cannot be empty. At minimum "source_id" is required`)
	}

	//Ensure all columns in column_ordering are found in the properties of incoming_message_schema.json
	err = validateColumnOrdering(obj.LogfileSettings.ColumnOrder, props)
	if err != nil {
		return err
	}

	//Looks good, save the incoming_message_schema.
	obj.ProtocolSettings.IncomingMessageSchema = data
	return nil
}

// Ensure all columns in column_ordering (config.json) exist in the incoming_message_schema.json
// Columns "timestamp" and "source_ip" are also allowed.
func validateColumnOrdering(columnOrder []string, props map[string]interface{}) error {

	for _, col := range columnOrder {

		if _, exists := props[col]; !exists {
			if col == "timestamp" || col == "source_ip" {
				continue
			}

			return fmt.Errorf("config.json>>logfile_settings>>column_order contains column not found in incoming_message_schema: %s", col)
		}
	}
	return nil
}
