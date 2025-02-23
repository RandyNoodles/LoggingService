package main

import (
	"LoggingService/config"
	abusePrevention "LoggingService/internal/abuse_prevention"
	clienthandling "LoggingService/internal/client_handling"
	"LoggingService/internal/logwriter"
	"fmt"
	"log"
	"net"
	"time"
)

func main() {

	var intSecs uint32 = uint32(time.Now().Unix())
	fmt.Println(intSecs)

	//Load config settings
	config, err := config.ParseConfigFile()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("config.json loaded.")

	//Init logwriter
	handler := clienthandling.New(*config)

	success, err := logwriter.TestLogfilePaths(config.LogfileSettings.Path, config.ErrorHandling.ErrorLogPath)
	if !success {
		fmt.Println(err)
	}

	//Init abuse prevention system
	ap := abusePrevention.New(config.ProtocolSettings)

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
		go handler.HandleClient(conn, ap)
	}
}
