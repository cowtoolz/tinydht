package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cowtools/tinydht/node"
)

func main() {
	c, err := node.Start()
	if err != nil {
		panic(fmt.Errorf("failed to start client: %v\n", err))
	}
	defer func() {
		err := c.Close()
		if err != nil {
			panic(fmt.Errorf("error while closing: %v", err))
		}
	}()

	// exports the current state to json, viewable in the browser
	go func() {
		for {
			time.Sleep(5 * time.Second)
			jsonClient, err := c.DebugJSON()
			if err != nil {
				panic(fmt.Errorf("error while exporting to JSON: %v", err))
			}
			err = os.WriteFile("state.json", jsonClient, 0o644)
			if err != nil {
				panic(fmt.Errorf("error while writing JSON file: %v", err))
			}
		}
	}()

	// the receiving thread, which keeps our node's state updated
	go func() {
		for {
			err := c.ReceiveCommand()
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARN: while receiving command: %v\n", err)
			}
		}
	}()

	// the sending thread, which propagates our node's state across the network
	go func() {
		for {
			err := c.Broadcast()
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARN: while receiving command: %v\n", err)
			}
			time.Sleep(time.Second * 5)
		}
	}()

	// take user input to add to the DHT
	go func() {
		for {
			var input string
			fmt.Print("Type something to add it to the DHT: ")
			_, err := fmt.Scanln(&input)
			if err != nil {
				panic(fmt.Errorf("error while reading user input: %v", err))
			}
			c.DebugAddToDHT(input)
		}
	}()

	<-c.Closed
}
