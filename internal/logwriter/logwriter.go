package logwriter

import (
	"LoggingService/config"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var logFileMutex sync.Mutex
var errFileMutex sync.Mutex

// Log format strings to be converted to int
type logFormat int

const (
	Json logFormat = iota
	Plaintext
	Error
)

// TimeFormats maps layout names to their corresponding time format strings.
var TimeFormats = map[string]string{
	"ANSIC":       time.ANSIC,
	"UnixDate":    time.UnixDate,
	"RubyDate":    time.RubyDate,
	"RFC822":      time.RFC822,
	"RFC822Z":     time.RFC822Z,
	"RFC850":      time.RFC850,
	"RFC1123":     time.RFC1123,
	"RFC1123Z":    time.RFC1123Z,
	"RFC3339":     time.RFC3339,
	"RFC3339Nano": time.RFC3339Nano,
	"Kitchen":     time.Kitchen,
}

type LogWriter struct {
	format              logFormat
	fieldDelimiter      string
	entryDelimiter      string
	columnOrder         []string
	timestampFormat     string
	isTimestampIncluded bool
	isSourceIpIncluded  bool
}

func New(logSettings config.LogfileSettings, timeAndIPSetting config.RequiredFieldSettings) *LogWriter {

	var convertedFormat logFormat
	if logSettings.Format == "json" {
		convertedFormat = Json
	} else if logSettings.Format == "plaintext" {
		convertedFormat = Plaintext
	} else {
		convertedFormat = Error
	}

	return &LogWriter{
		format:              convertedFormat,
		fieldDelimiter:      logSettings.PlaintextFieldDelimiter,
		entryDelimiter:      logSettings.PlaintextEntryDelimiter,
		columnOrder:         logSettings.ColumnOrder,
		timestampFormat:     timeAndIPSetting.TimestampFormat,
		isTimestampIncluded: timeAndIPSetting.TimestampIncludeInLogs,
		isSourceIpIncluded:  timeAndIPSetting.SourceIPIncludeInLogs,
	}
}

func TestLogfilePaths(log string, errorLog string) (bool, error) {

	//Attempt to open or create Logfile
	f, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, fmt.Errorf("failed to open or create file %q: %w", log, err)
	}
	f.Close()

	//Attempt to open or create Error Log
	f, err = os.OpenFile(errorLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, fmt.Errorf("failed to open or create file %q: %w", errorLog, err)
	}
	f.Close()

	return true, nil
}

func (lw *LogWriter) WriteErrorToFile(message string, category string, path string) error {
	errFileMutex.Lock()
	defer errFileMutex.Unlock()

	// Open the file in append mode. Create it if it doesn't exist.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Format the log entry with a timestamp.
	logEntry := fmt.Sprintf("ERROR: %s: %s: %s\n", category, time.Now().Format(time.RFC3339), message)
	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write error to file: %w", err)
	}
	return nil
}

func (lw *LogWriter) WriteLogToFile(logEntry string, path string) error {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	// Open the file in append mode. Create it if it doesn't exist.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to log to file: %w", err)
	}

	return nil
}

func (lw *LogWriter) FormatLogEntry(log map[string]interface{}, clientIp string) (string, error) {
	if lw.isTimestampIncluded {
		log["timestamp"] = time.Now().Format(TimeFormats[lw.timestampFormat])
	}

	if lw.isSourceIpIncluded {
		log["source_ip"] = clientIp

	}

	//If simple JSON formatting, just re-marshal the map with the new fields added.
	if lw.format == Json {
		formattedLog, err := json.Marshal(log)
		if err != nil {
			return "", err
		}
		return string(formattedLog), nil
	}

	//Else, format it using the delimiters
	var sb strings.Builder
	for _, column := range lw.columnOrder {
		sb.WriteString(fmt.Sprintf("%v%s", log[column], lw.fieldDelimiter))
	}
	sb.WriteString(lw.entryDelimiter)
	return sb.String(), nil
}
