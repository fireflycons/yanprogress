package main

import "time"

func main() {
	// Example usage
	bar := NewProgressBar(100, 500*time.Millisecond)
	bar.SetStatus("Downloading File")
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
}
