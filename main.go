package main

import (
	"bufio"
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

	// the receiving thread, which keeps our node's state updated
	go func() {
		for {
			select {
			case <-c.Closed:
				return
			default:
				err := c.ReceiveCommand()
				if err != nil {
					fmt.Fprintf(os.Stderr, "WARN: while receiving: %v\n", err)
				}
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()

	// the sending thread, which propagates our node's state across the network
	go func() {
		for {
			select {
			case <-c.Closed:
				return
			default:
				err := c.Broadcast()
				if err != nil {
					fmt.Fprintf(os.Stderr, "WARN: while broadcasting: %v\n", err)
				}
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()

	// exports the current state to a json file `state.json`
	go func() {
		for {
			select {
			case <-c.Closed:
				return
			default:
				jsonClient, err := c.DebugJSON()
				if err != nil {
					panic(fmt.Errorf("error while exporting to JSON: %v", err))
				}
				err = os.WriteFile("state.json", jsonClient, 0o644)
				if err != nil {
					panic(fmt.Errorf("error while writing JSON file: %v", err))
				}
				time.Sleep(time.Second * 2)
			}
		}
	}()

	// take user input to add to the DHT
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Type something to add it to the DHT: ")
		if scanner.Scan() {
			input := scanner.Text()
			if input == "exit" {
				break
			}
			c.DebugAddToDHT(input)
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "error while reading user input: %v", err)
			break
		}
	}

	err = c.Close()
	if err != nil {
		panic(fmt.Errorf("error while closing: %v", err))
	}

	<-c.Closed
	close(c.Closed)
}
