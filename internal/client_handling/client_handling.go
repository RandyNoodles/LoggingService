package clienthandling

//

import (
	"LoggingService/config"
	abuseprevention "LoggingService/internal/abuse_prevention"
	"LoggingService/internal/logwriter"
	"encoding/json"
	"fmt"
	"net"
	"strings"

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
	clientAddress := strings.Split(conn.RemoteAddr().String(), ":")

	fmt.Printf("Client address: %s\n", conn.RemoteAddr())

	clientIp := clientAddress[0]

	//Check if IP is banned
	result := abusePrevention.CheckIPBlacklist(clientIp)
	if result != nil {
		handler.sendResponse(conn, false, result.Error())
		return
	}

	// //Log message in rate limiter, check if rate has been exceeded.
	// err := abusePrevention.CheckIPRateLimiter(clientIp)
	// if err != nil {
	// 	handler.sendResponse(conn, false, result.Error())
	// 	return
	// }

	//Read the message stream into memory
	buffer := make([]byte, 4196)
	bytesRead, err := conn.Read(buffer)
	if bytesRead == 0 {
		handler.logWriter.WriteErrorToFile(err.Error(), "internal:HandleClient():conn.Read()", handler.errlogPath)
		handler.sendResponse(conn, false, "internal server error")
		return
	}

	message := buffer[:bytesRead]

	//Check message against json schema
	err = handler.ValidateMessage(message, handler.schema)
	if err != nil {
		//Are they banned now? If so let them know.
		banMessage := abusePrevention.IncrementBadFormatCount(clientIp)
		if banMessage != nil {
			handler.sendResponse(conn, false, banMessage.Error())
		} else { //Else, just send back the formatting errors
			handler.sendResponse(conn, false, err.Error())
		}
		if handler.errorSettings.InvalidMessage == "redirect_to_error_log" {
			handler.logWriter.WriteErrorToFile(err.Error(), "invalid message format", handler.errlogPath)
		}
		return
	}

	//Parse json into map
	var parsedMessage map[string]interface{}
	err = json.Unmarshal(message, &parsedMessage)
	if err != nil {
		handler.logWriter.WriteErrorToFile(err.Error(), "internal:HandleClient():json.Unmarshal()", handler.errlogPath)
		handler.sendResponse(conn, false, "internal server error")
		return
	}

	//Check source_id blacklist
	clientId, _ := parsedMessage["source_id"].(string)
	result = abusePrevention.CheckSourceIDBlacklist(clientId)
	if result != nil {
		handler.sendResponse(conn, false, result.Error())
		return
	}

	// //Log message in rate limiter, check if rate has been exceeded.
	// err = abusePrevention.CheckSourceRateLimiter(parsedMessage["source_id"].(string))
	// if err != nil {
	// 	handler.sendResponse(conn, false, result.Error())
	// 	return
	// }

	//Format log
	formattedLog, err := handler.logWriter.FormatLogEntry(parsedMessage, clientIp)
	if err != nil {
		handler.logWriter.WriteErrorToFile(err.Error(), "internal: internal:HandleClient():FormatLogEntry()", handler.errlogPath)
		handler.sendResponse(conn, false, "internal server error")
		return
	}

	//Write log to file
	err = handler.logWriter.WriteLogToFile(formattedLog, handler.logPath)
	if err != nil {
		handler.logWriter.WriteErrorToFile(err.Error(), "internal:HandleClient():WriteLogToFile()", handler.errlogPath)
		handler.sendResponse(conn, false, "internal server error")
		return
	}

	//Send "Success" response to client
	response := []byte(`{"success": true, "message": "log received"}`)

	bytesWritten, err := conn.Write(response)
	if bytesWritten == 0 || err != nil {
		errorMessage := fmt.Sprintf("Unable to respond to client on connection: %s", conn.RemoteAddr())
		handler.logWriter.WriteErrorToFile(errorMessage, "internal:sendResponse():conn.Write()", handler.errlogPath)
	}
}

// Validate the incoming message against the json schema referenced in config.json
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

func (handler *ClientHandler) sendResponse(conn net.Conn, success bool, message string) {

	response := fmt.Sprintf("{\"success\": %v, \"message\": \"%s\"}", success, message)

	bytesWritten, err := conn.Write([]byte(response))

	if bytesWritten == 0 || err != nil {
		errorMessage := fmt.Sprintf("Unable to respond to client on connection: %s", conn.RemoteAddr())
		handler.logWriter.WriteErrorToFile(errorMessage, "internal:sendResponse():conn.Write()", handler.errlogPath)
	}
}
