package camera

import (
	"fmt"
	"webcam/internal/i18n"
)

// CompareCameras prints a side-by-side comparison of available camera parameters.
func CompareCameras(cameras []Config) {
	if len(cameras) == 0 {
		fmt.Println(i18n.T("No cameras found"))
		return
	}

	base := cameras[0]
	others := cameras[1:]

	fmt.Println(i18n.T("\nCamera Comparison"))
	fmt.Println("===============================================================")

	fmt.Printf("%-20s %-15s", i18n.T("Name"), base.Name)

	for _, cam := range others {
		fmt.Printf(" %-15s", cam.Name)
	}
	fmt.Println()

	fmt.Println("---------------------------------------------------------------")

	fmt.Printf("%-20s %-15s", i18n.T("Resolution"), fmt.Sprintf("%dx%d", base.Width, base.Height))
	for _, cam := range others {
		fmt.Printf(" %-15s", fmt.Sprintf("%dx%d", cam.Width, cam.Height))
	}
	fmt.Println()

	fmt.Printf("%-20s %-15.2f", "FPS", base.FPS)
	for _, cam := range others {
		fmt.Printf(" %-15.2f", cam.FPS)
	}
	fmt.Println()

	fmt.Printf("%-20s %-15s", i18n.T("Format"), base.Format)
	for _, cam := range others {
		fmt.Printf(" %-15s", cam.Format)
	}
	fmt.Println()

	fmt.Println("===============================================================")
}

// PrintCameraList prints available cameras with their type (internal/external).
func PrintCameraList(cameras []Config) {
	fmt.Println(i18n.T("Available cameras:"))
	for i, cam := range cameras {
		fmt.Printf("[%d] - %s\n", i+1, cam.Name)
	}
}
