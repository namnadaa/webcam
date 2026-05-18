package camera

import (
	"fmt"
	"strconv"
)

// SelectCamera prompts user to select a valid camera index with input validation loop.
func SelectCamera(cameras []Config) (*Config, bool) {
	var input string

	for {
		fmt.Print("Enter camera index or 'q' to quit: ")
		fmt.Scanln(&input)

		if input == "q" || input == "Q" {
			return nil, false
		}

		index, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Invalid input. Please enter a number.")
			continue
		}

		cam, found := findCameraByIndex(cameras, index)
		if !found {
			fmt.Println("Camera not found. Try again.")
			continue
		}

		return cam, true
	}
}

// findCameraByIndex searches for a camera by its index and returns it if found.
func findCameraByIndex(cameras []Config, index int) (*Config, bool) {
	for i := range cameras {
		if cameras[i].Index == index {
			return &cameras[i], true
		}
	}
	return nil, false
}
