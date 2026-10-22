package camera

import (
	"fmt"
	"strconv"
)

// SelectCamera prompts user to select a camera by its number in the list (starting from 1) with input validation loop.
func SelectCamera(cameras []Config) (*Config, bool) {
	var input string

	for {
		fmt.Printf("Enter camera number [1-%d] or 'q' to quit: ", len(cameras))
		fmt.Scanln(&input)

		if input == "q" || input == "Q" {
			return nil, false
		}

		number, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Invalid input. Please enter a number.")
			continue
		}

		cam, found := findCameraByNumber(cameras, number)
		if !found {
			fmt.Println("Camera not found. Try again.")
			continue
		}

		return cam, true
	}
}

// findCameraByNumber returns the camera at the given 1-based position in the list.
func findCameraByNumber(cameras []Config, number int) (*Config, bool) {
	if number < 1 || number > len(cameras) {
		return nil, false
	}
	return &cameras[number-1], true
}
