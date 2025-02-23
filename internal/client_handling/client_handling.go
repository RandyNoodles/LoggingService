package clienthandling

//

import (
	"LoggingService/config"
	abuseprevention "LoggingService/internal/abuse_prevention"
	"LoggingService/internal/logwriter"
	"encoding/json"
	"fmt"
	"net"

	"github.com/xeipuuv/gojsonschema"
)

type ClientHandler struct {
	schema        []byte
	errorSettings config.ErrorSettings
	logWriter     *logwriter.LogWriter
	logPath       string
	errlogPath    string
}

// Construct new ClientHandler (compose along with new LogWriter)
func New(settings config.Config) *ClientHandler {
	return &ClientHandler{
		schema:        settings.ProtocolSettings.IncomingMessageSchema,
		errorSettings: settings.ErrorHandling,
		logWriter:     logwriter.New(settings.LogfileSettings, settings.RequiredFieldSettings),
		logPath:       settings.LogfileSettings.Path,
		errlogPath:    settings.ErrorHandling.ErrorLogPath,
	}
}

// Main go routine client handler function
func (handler *ClientHandler) HandleClient(conn net.Conn, abusePrevention *abuseprevention.AbusePreventionTracker) {

	defer conn.Close()

	//Get client IP
	clientIp := conn.RemoteAddr().String()

	//Check if IP is banned
	result := abusePrevention.CheckIPBlacklist(clientIp)
	if result != nil {
		fmt.Println(result)
		return
	}

	//Read the message stream into memory
	buffer := make([]byte, 4196)
	bytesRead, err := conn.Read(buffer)
	if bytesRead == 0 {
		fmt.Println(err)
		return
	}

	message := buffer[:bytesRead]

	////Check message against json schema
	err = handler.ValidateMessage(message, handler.schema)
	if err != nil {
		fmt.Println(err)
		return
		//LOCK () Increment blacklist
		//If redirect_to_error_log -> log error
	}

	//Parse json into map
	var parsedMessage map[string]interface{}
	err = json.Unmarshal(message, &parsedMessage)
	if err != nil {
		fmt.Println(err)
		return
	}

	//Check source_id blacklist
	clientId, _ := parsedMessage["source_id"].(string)
	result = abusePrevention.CheckSourceIDBlacklist(clientId)
	if result != nil {
		fmt.Println(result)
	}

	//Format log
	formattedLog, err := handler.logWriter.FormatLogEntry(parsedMessage, clientIp)
	if err != nil {
		fmt.Println(err)
	}

	//Write log to file
	err = handler.logWriter.WriteLogToFile(formattedLog, handler.logPath)
	if err != nil {
		fmt.Println(err)
	}

	var response []byte = []byte(`{"success": true, "message": "test 1-2"}`)
	conn.Write(response)
}

func (handler *ClientHandler) ValidateMessage(data []byte, schema []byte) error {

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
