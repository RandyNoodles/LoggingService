package main

import (
	"LoggingService/config"
	"fmt"
	"log"
)

func main() {
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

	//Listen for incoming messages

	//WHEN MESSAGE IS RECEIVED
	//Check if its valid
	//Note sender & time via abuse prevention -> make sure it doesn't violate standards
	//Write to logfile

}
