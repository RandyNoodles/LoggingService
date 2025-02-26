/*
* FILE : 			main.go
* FIRST VERSION : 	2025-02-22
* DESCRIPTION :
		Entry point for the logging service.

		Once systems are setup, runs listener in a loop which spawns new
		go routines to handle any incoming client requests.
*/

package main

import (
	"LoggingService/config"
	clienthandling "LoggingService/internal/client_handling"
	"LoggingService/internal/logwriting"
	"fmt"
	"log"
	"net"
)

func main() {

	//Load config settings
	config, err := config.ParseConfigFile()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("config.json loaded.")

	//Init client handler
	//Contains instances of abuse prevention and logwriter systems
	handler := clienthandling.New(*config)

	//Test logfile paths
	success, err := logwriting.TestLogfilePaths(config.LogfileSettings.Path, config.ErrorHandling.ErrorLogPath)
	if !success {
		fmt.Println(err)
	}

	//Init listener
	addressString := fmt.Sprintf("%s:%d", config.ServerSettings.IpAddress, config.ServerSettings.Port)
	listener, err := net.Listen("tcp", addressString)
	if err != nil {
		log.Fatal("Error starting TCP listener: ", err)
	}
	defer listener.Close()
	fmt.Printf("TCP listener starting at %s", addressString)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		//Use goroutine to handle connection
		go handler.HandleClient(conn)
	}
}
