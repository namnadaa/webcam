package camera

import "fmt"

// CompareCameras prints a side-by-side comparison of available camera parameters.
func CompareCameras(cameras []Config) {
	if len(cameras) == 0 {
		fmt.Println("No cameras found")
		return
	}

	base := cameras[0]
	others := cameras[1:]

	fmt.Println("\nCamera Comparison")
	fmt.Println("===============================================================")

	fmt.Printf("%-20s %-15s", "Name", base.Name)

	for _, cam := range others {
		fmt.Printf(" %-15s", cam.Name)
	}
	fmt.Println()

	fmt.Println("---------------------------------------------------------------")

	fmt.Printf("%-20s %-15s", "Resolution", fmt.Sprintf("%dx%d", base.Width, base.Height))
	for _, cam := range others {
		fmt.Printf(" %-15s", fmt.Sprintf("%dx%d", cam.Width, cam.Height))
	}
	fmt.Println()

	fmt.Printf("%-20s %-15.2f", "FPS", base.FPS)
	for _, cam := range others {
		fmt.Printf(" %-15.2f", cam.FPS)
	}
	fmt.Println()

	fmt.Printf("%-20s %-15s", "Format", base.Format)
	for _, cam := range others {
		fmt.Printf(" %-15s", cam.Format)
	}
	fmt.Println()

	fmt.Println("===============================================================")
}

// PrintCameraList prints available cameras with their type (internal/external).
func PrintCameraList(cameras []Config) {
	fmt.Println("Available cameras:")
	for _, cam := range cameras {
		fmt.Printf("[%d] - %s\n", cam.Index, cam.Name)
	}
}
