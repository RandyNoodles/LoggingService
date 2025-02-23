package main

import (
	"LoggingService/config"
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

	fmt.Println(string(config.ProtocolSettings.IncomingMessageSchema)) //THIS IS JUST TO PREVENT IT GETTING DELETED BY AUTO FORMATTING

	//Make sure our logFile exists/can be written to

	//Init abuse prevention system

	//Init listener

	portString := fmt.Sprintf(":%d", config.ServerSettings.Port)
	listener, err := net.Listen(config.ServerSettings.IpAddress, portString)
	if err != nil {
		log.Fatal("Error starting TCP listener: ", err)
	}
	defer listener.Close()
	fmt.Printf("TCP listener starting at %s%s", config.ServerSettings.IpAddress, portString)

	// for {
	// 	conn, err := listener.Accept()
	// 	if err != nil {
	// 		fmt.Println("Error accepting connection:", err)
	// 		continue
	// 	}

	// 	//Use goroutine to handle connection
	// 	//go handleConnection(conn)
	// }

	//Listen for incoming messages

	//WHEN MESSAGE IS RECEIVED
	//Check if its valid
	//Note sender & time via abuse prevention -> make sure it doesn't violate standards
	//Write to logfile

}
