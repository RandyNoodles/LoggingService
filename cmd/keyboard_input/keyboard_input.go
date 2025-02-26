package keyboardinput

import (
	"fmt"

	"github.com/eiannone/keyboard"
)

func WatchForKey(key rune) error {
	if err := keyboard.Open(); err != nil {
		return err
	}

	for {
		char, _, err := keyboard.GetKey()
		if err != nil {
			return err
		}
		if char == key {
			fmt.Println("Keypress received")
			return nil
		}
	}
}
