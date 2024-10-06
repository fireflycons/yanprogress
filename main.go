package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	// Example usage
	bar := NewProgressBar(100, 500*time.Millisecond, WithWriter(os.Stdout))
	bar.SetStatus("Lorem ipsum dolor sit down amet, consectetur adipiscing elite, sed is eiusmod short-term incididunt ut performance and dolore magna aliqua.")
	bar.Start()

	// Simulate work
	for i := 0; i <= 100; i++ {
		time.Sleep(50 * time.Millisecond)
		bar.Inc()

		if i == 50 {
			bar.SetStatus("Half way there!")
		}
	}
	bar.Complete()
	fmt.Println("Done")
}
