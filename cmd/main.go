package main

import (
	"LoggingService/config"
	"fmt"
	"log"
)

func main() {

	var columns config.ColumnOrder

	columns.SetColumnOrder([]string{"1", "2", "3"})

	fmt.Println(columns.GetColumnOrder())

	//Load config settings
	config, err := config.ParseConfigFile()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("config.json loaded.")
	fmt.Println(config) //THIS IS JUST TO PREVENT IT GETTING DELETED BY AUTO FORMATTING

	//Load incoming_message_schema into memory, make sure it has timestamp and source_id
	//Load in field ordering slice from field keys

	//Make sure our logFile exists/can be written to

	//Init abuse prevention system

	//Init listener

	//Listen for incoming messages

	//WHEN MESSAGE IS RECEIVED
	//Check if its valid
	//Note sender & time via abuse prevention -> make sure it doesn't violate standards
	//Write to logfile

}
