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
	"sync"

	"github.com/eiannone/keyboard"
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

	//Waitgroup to wrap up all client handlers before shutdown
	var wg sync.WaitGroup

	//Channel to receive shutdown signal
	quit := make(chan error, 5)
	//Watch for keypress to shutdown server
	go WatchForShutdownKey(quit, addressString)

	for {

		select {
		//If shutdown signal received
		case err := <-quit:
			if err != nil {
				fmt.Println("Shutdown error:", err)
			} else {
				fmt.Println("\nShutdown signal received.")
			}

			// Stop accepting new connections
			fmt.Println("Closing listener...")
			listener.Close()

			// Wait for all running client handlers to finish
			wg.Wait()

			fmt.Println("Server shut down successfully.")
			return

		default:
			fmt.Println("server blocked")
			conn, err := listener.Accept()
			fmt.Println("server unblocked")
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}

			wg.Add(1)
			go func(conn net.Conn) {
				defer wg.Done()
				handler.HandleClient(conn)
			}(conn)
		}
	}
}

// ////////////////////////////////////////////////////////////////
// Async watch for shutdown keypress and return value via channel
func WatchForShutdownKey(quit chan error, serverAddress string) {

	if err := keyboard.Open(); err != nil {
		quit <- err
	}

	defer keyboard.Close()
	for {
		char, _, err := keyboard.GetKey()
		if err != nil {
			quit <- err
		}
		if char == 'q' || char == 'Q' {
			fmt.Println("Keypress received")
			break
		}
	}

	//Exit server on next loopq
	quit <- nil

	//Send dummy request to unblock listener.Accept()
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		quit <- fmt.Errorf("error sending unblocking connection: %s", err.Error())
		return
	}
	if conn != nil {
		conn.Close()
	}
}
