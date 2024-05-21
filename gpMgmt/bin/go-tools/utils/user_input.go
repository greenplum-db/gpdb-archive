package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func AskUserYesNo(waittime time.Duration) bool {
	// Create a channel to receive user input
	userInput := make(chan string)

	// Create a timer channel that sends a signal after the timeout duration
	timer := time.After(waittime * time.Second)

	// Create a goroutine to read user input
	go func() {
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("ReadString error")
		}
		userInput <- strings.ToLower(strings.TrimSpace(input))
	}()

	// Wait for either user input or timeout
	select {
	case input := <-userInput:
		if input == "yes" || input == "y" {
			fmt.Println("You entered 'yes'.")
			return true
		} else if input == "no" || input == "n" {
			fmt.Println("You entered 'no'.")
			return false
		} else {
			fmt.Println("Invalid input. Please enter 'yes' or 'no'.")
		}
	case <-timer:
		fmt.Println("no user input received. Default input is 'no'.")
		return false
	}
	return false
}
