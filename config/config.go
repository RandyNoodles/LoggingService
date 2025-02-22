package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed config_schema.json
var configValidationSchema []byte //Embed config schema into binary to avoid user tampering

// Holds all three config sections from parsed config.json file
type Config struct {
	LogfileSettings  LogfileSettings  `json:"logfile_settings"`
	ProtocolSettings ProtocolSettings `json:"protocol_settings"`
	ErrorHandling    ErrorHandling    `json:"error_handling"`
}

// Settings for logfile configuration
type LogfileSettings struct {
	PathType                string `json:"path_type"`
	Path                    string `json:"path"`
	Format                  string `json:"format"`
	PlaintextFieldDelimiter string `json:"plaintext_field_delimiter"`
	PlaintextEntryDelimiter string `json:"plaintext_entry_delimiter"`
	MaxSizeKB               int    `json:"max_size_kb"`
	RotationMinutes         int    `json:"rotation_minutes"`
	fieldOrder              []string
}

// Settings for Protocol & abuse prevention
type ProtocolSettings struct {
	IncomingJSONSchema           string   `json:"incoming_json_schema"`
	IpMessagesPerMinute          int      `json:"messages_per_ip_per_minute"`
	SourceMessagesPerMinute      int      `json:"messages_per_source_id_per_minute"`
	BadMessageBlacklistThreshold int      `json:"bad_message_blacklist_threshold"`
	BlacklistedIPs               []string `json:"blacklisted_ips"`
	BlacklistPermanent           bool     `json:"blacklist_permanent"`
	BlacklistDurationMinutes     int      `json:"blacklist_duration_minutes"`
}

// Settings for error handling
type ErrorHandling struct {
	MissingRequiredField string `json:"missing_required_field"`
	ExtraField           string `json:"extra_field"`
	InvalidMessage       string `json:"invalid_message"`
	ErrorLogPath         string `json:"error_log_path"`
}

func ParseConfigFile() (*Config, error) {

	data, err := os.ReadFile("../config.json")

	if err != nil {
		return nil, err
	}

	// Load the embedded config schema
	schemaLoader := gojsonschema.NewStringLoader(string(configValidationSchema))
	configDataLoader := gojsonschema.NewStringLoader(string(data))

	result, err := gojsonschema.Validate(schemaLoader, configDataLoader) //Compare config stuff against embedded config schema

	if err != nil { //Validator straight up failed - likely invalid JSON syntax or something?
		return nil, err
	}

	var config Config //Config struct to store return value

	if result.Valid() { // Schema is valid, continue
		err = json.Unmarshal(data, &config) // Parse the JSON file into our config structs
		if err != nil {
			fmt.Print(string(data))
			return nil, err
		}
	} else { // Schema validation failed, return the results.
		var errorMessages string
		for _, err := range result.Errors() {
			errorMessages += fmt.Sprintf("- %s\n", err)
		}
		return nil, fmt.Errorf("schema validation failed: \n%s", errorMessages)
	}

	return &config, err
}

func (obj *LogfileSettings) SetFieldOrder(names []string) {
	obj.fieldOrder = names
}

func (obj LogfileSettings) GetFieldOrder() []string {
	return obj.fieldOrder
}
