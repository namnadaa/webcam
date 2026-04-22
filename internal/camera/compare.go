package camera

import "fmt"

// CompareInternalExternal compares internal camera with all external cameras.
func CompareInternalExternal(cameras []Config) {
	var internal *Config
	var externals []Config

	for i := range cameras {
		cam := &cameras[i]

		if cam.IsExternal {
			externals = append(externals, *cam)
		} else {
			internal = cam
		}
	}

	if internal == nil {
		fmt.Println("Internal camera not found")
		return
	}

	if len(externals) == 0 {
		fmt.Println("External cameras not found")
		return
	}

	fmt.Println("\nCamera Comparison")
	fmt.Println("===============================================================")

	fmt.Printf("%-20s %-15s", "Parameter", "Internal")
	for i := range externals {
		fmt.Printf(" %-15s", fmt.Sprintf("External %d", externals[i].Index))
	}
	fmt.Println()

	fmt.Println("---------------------------------------------------------------")

	internalRes := fmt.Sprintf("%dx%d", internal.Width, internal.Height)
	fmt.Printf("%-20s %-15s", "Resolution", internalRes)

	for _, ext := range externals {
		res := fmt.Sprintf("%dx%d", ext.Width, ext.Height)
		fmt.Printf(" %-15s", res)
	}
	fmt.Println()

	fmt.Printf("%-20s %-15.2f", "FPS", internal.FPS)

	for _, ext := range externals {
		fmt.Printf(" %-15.2f", ext.FPS)
	}
	fmt.Println()

	fmt.Printf("%-20s %-15s", "Format", internal.Format)

	for _, ext := range externals {
		fmt.Printf(" %-15s", ext.Format)
	}
	fmt.Println()

	fmt.Println("===============================================================")
}
