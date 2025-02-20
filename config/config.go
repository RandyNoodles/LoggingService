package config

type LogFileConfig struct {
	ValidLevels             []string `json:"valid_levels"`
	PathType                string   `json:"path_type"`
	Path                    string   `json:"path"`
	Format                  string   `json:"format"`
	NonSyslogFieldOrder     []string `json:"non_syslog_field_order"`
	PlaintextFieldDelimiter string   `json:"plaintext_field_delimiter"`
	PlaintextEntryDelimiter string   `json:"plaintext_entry_delimiter"`
	MaxSizeKB               int      `json:"max_size_kb"`
	RotationMinutes         int      `json:"rotation_minutes"`
}

type ProtocolConfig struct {
	Format                       string   `json:"format"`
	IncomingJSONSchema           string   `json:"incoming_json_schema"`
	MessagesPerMinute            int      `json:"messages_per_minute"`
	BadMessageBlacklistThreshold int      `json:"bad_message_blacklist_threshold"`
	BlacklistedIPs               []string `json:"blacklisted_ips"`
	BlacklistPermanent           bool     `json:"blacklist_permanent"`
	BlacklistDurationMinutes     int      `json:"blacklist_duration_minutes"`
}

type ErrorHandlingConfig struct {
	MissingRequiredField string `json:"missing_required_field"`
	ExtraField           string `json:"extra_field"`
	FieldExceedsLength   string `json:"field_exceeds_length"`
	InvalidMessage       string `json:"invalid_message"`
	ErrorLogPath         string `json:"error_log_path"`
}

type Config struct {
	LogFileSettings  LogFileConfig       `json:"logfile_settings"`
	ProtocolSettings ProtocolConfig      `json:"protocol_settings"`
	ErrorHandling    ErrorHandlingConfig `json:"error_handling"`
}
