/*
* FILE : 			client_handling.go
* FIRST VERSION : 	2025-02-23
* DESCRIPTION :
		HandleClient() is the main function to handle incoming log messages.

		Usage:
		- clientHandling.New(*config.Config) to instantiate a client handler
		- Use go routines to call clientHandling.HandleClient()

		Mutexes will handle concurrency issues between log writing and access
		to abuse prevention mechanisms.
*/

package clienthandling

import (
	"LoggingService/config"
	abuseprevention "LoggingService/internal/abuse_prevention"
	"LoggingService/internal/logwriting"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

var abusePreventionMutex sync.Mutex

type ClientHandler struct {
	schema          []byte
	errorSettings   config.ErrorSettings
	logWriter       *logwriting.LogWriter
	abusePrevention *abuseprevention.AbusePreventionTracker
	logPath         string
	errlogPath      string
}

// Construct new ClientHandler (compose along with new LogWriter)
func New(settings config.Config) *ClientHandler {
	return &ClientHandler{
		schema:          settings.ProtocolSettings.IncomingMessageSchema,
		errorSettings:   settings.ErrorHandling,
		logWriter:       logwriting.New(settings.LogfileSettings),
		abusePrevention: abuseprevention.New(settings.ProtocolSettings),
		logPath:         settings.LogfileSettings.Path,
		errlogPath:      settings.ErrorHandling.ErrorLogPath,
	}
}

// Main go routine client handler function
func (h *ClientHandler) HandleClient(conn net.Conn) {

	defer conn.Close()

	//Read the message stream into memory
	buffer := make([]byte, 4196)
	bytesRead, err := conn.Read(buffer)
	if bytesRead == 0 {
		h.logWriter.WriteErrorToFile(err.Error(), "internal:HandleClient():conn.Read()", h.errlogPath)
		h.sendResponse(conn, false, "internal server error")
		return
	}

	//Truncate trailing '\00' chars
	message := buffer[:bytesRead]

	//Get client IP
	clientAddress := strings.Split(conn.RemoteAddr().String(), ":")

	clientIp := clientAddress[0]

	err = h.ValidateMessage(message, clientIp)
	if err != nil {
		h.sendResponse(conn, false, err.Error())
	}

	//Parse json into map
	var parsedMessage map[string]interface{}
	err = json.Unmarshal(message, &parsedMessage)
	if err != nil {
		h.logWriter.WriteErrorToFile(err.Error(), "internal:HandleClient():json.Unmarshal()", h.errlogPath)
		h.sendResponse(conn, false, "internal server error")
		return
	}

	//Format log
	formattedLog, err := h.logWriter.FormatLogEntry(parsedMessage, clientIp)
	if err != nil {
		h.logWriter.WriteErrorToFile(err.Error(), "internal: internal:HandleClient():FormatLogEntry()", h.errlogPath)
		h.sendResponse(conn, false, "internal server error")
		return
	}

	//Write log to file
	err = h.logWriter.WriteLogToFile(formattedLog, h.logPath)
	if err != nil {
		h.logWriter.WriteErrorToFile(err.Error(), "internal:HandleClient():WriteLogToFile()", h.errlogPath)
		h.sendResponse(conn, false, "internal server error")
		return
	}

	//Send "Success" response to client
	response := []byte(`{"success": true, "message": "log received"}`)

	bytesWritten, err := conn.Write(response)
	if bytesWritten == 0 || err != nil {
		errorMessage := fmt.Sprintf("Unable to respond to client on connection: %s", conn.RemoteAddr())
		h.logWriter.WriteErrorToFile(errorMessage, "internal:sendResponse():conn.Write()", h.errlogPath)
	}
}

// Runs all abuse prevention stuff & validates client message against schema
func (h *ClientHandler) ValidateMessage(data []byte, clientIp string) error {

	abusePreventionMutex.Lock()
	defer abusePreventionMutex.Unlock()

	//Check if IP is banned
	result := h.abusePrevention.CheckIPBlacklist(clientIp)
	if result != nil {
		return result
	}

	//Log message in rate limiter, check if rate has been exceeded.
	err := h.abusePrevention.CheckIPRateLimiter(clientIp)
	if err != nil {
		return err
	}

	//Check message against json schema
	err = h.CompareAgainstSchema(data, h.schema)
	if err != nil {
		//Are they banned now? If so let them know.
		banMessage := h.abusePrevention.IncrementBadFormatCount(clientIp)
		if banMessage != nil {
			return banMessage
		} else { //Else, just send back the formatting errors
			//Log it as well, if that's what config says
			if h.errorSettings.InvalidMessage == "redirect_to_error_log" {
				h.logWriter.WriteErrorToFile(err.Error(), "invalid message format", h.errlogPath)
			}
			return err
		}
	}
	return nil
}

// Validate the incoming message against the json schema referenced in config.json
func (handler *ClientHandler) CompareAgainstSchema(data []byte, schema []byte) error {

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
