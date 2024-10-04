package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

type (
	ProgressBar struct {
		max         int64         // Maximum value (the 100% mark)
		lastTaskLen int           // Length of the last task text
		current     uint64        // The current progress value
		interval    time.Duration // Interval between redraws
		start       time.Time     // Time when the bar was started
		status      string        // Status description (if any)
		done        chan struct{} // Channel to signal completion
		lines       int           // Number of lines needed for the progress bar
		dest        io.Writer     // destination eg stdout/stderr
		isRunning   bool          // true after Start has been called
		taskChanged bool
		isaTTY      bool
	}

	optionFunc func(*ProgressBar)
)

// WithWriter sets the output stream for the bar (default os.Stderr)
func WithWriter(w io.Writer) optionFunc {
	return func(p *ProgressBar) {
		p.dest = w
	}
}

// NewProgressBar creates a new progress bar
func NewProgressBar(max int64, interval time.Duration, opts ...optionFunc) *ProgressBar {
	p := &ProgressBar{
		max:      max,
		interval: interval,
		start:    time.Now(),
		done:     make(chan struct{}),
		dest:     os.Stderr,
	}

	for _, o := range opts {
		o(p)
	}

	p.isaTTY = func() bool {
		if f, ok := p.dest.(*os.File); ok {
			return isatty(f.Fd())
		}

		return false
	}()

	return p
}

// Inc increments the progress bar by one.
func (p *ProgressBar) Inc() {
	atomic.AddUint64(&p.current, 1)
}

// Set sets the progress bar to a specific value.
func (p *ProgressBar) Set(value uint64) {
	atomic.StoreUint64(&p.current, value)
}

// Complete marks the progress as complete and shows the cursor.
func (p *ProgressBar) Complete() {

	p.isRunning = false

	if p.max > 0 {
		// Set the current progress to the max explicitly
		atomic.StoreUint64(&p.current, uint64(p.max))
	}

	// Redraw one last time at exactly 100%
	p.redraw()

	close(p.done)
	if p.isaTTY {
		if runtime.GOOS == "windows" {
			cursorMoveDown(3)
		}
		cursorShow() // Show the cursor on completion
	}
}

// SetStatus sets the status for the progress bar.
func (p *ProgressBar) SetStatus(status string) {

	p.status = strings.TrimSpace(status)

	if p.isRunning {
		// Clear the old task line before setting a new task
		if p.lastTaskLen > 0 && p.isaTTY {
			p.clearTaskLine()
		}
		p.redraw()
	}

	p.lastTaskLen = len(status)
	p.taskChanged = true
}

// Start hides the cursor and starts drawing the progress bar.
func (p *ProgressBar) Start() {

	// Emit new lines to avoid overwriting existing terminal content
	p.prepareNewLines()

	go func() {
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				p.redraw()
			case <-p.done:
				p.redraw() // Final redraw at completion
				return
			}
		}
	}()

	p.isRunning = true
}

// clearTaskLine clears the task line by moving the cursor up and overwriting it with spaces
func (p *ProgressBar) clearTaskLine() {
	if !p.isaTTY {
		return
	}
	// Move the cursor up to the task line and clear it
	cursorMoveUp(p.lines)
	fmt.Print("\r" + stringRepeat(" ", p.lastTaskLen)) // Clear the line by overwriting with spaces
	fmt.Print("\r")                                    // Move cursor back to the start of the line
	cursorMoveDown(p.lines)
}

// prepareNewLines emits enough new lines to ensure the progress bar doesn't overwrite existing terminal content
func (p *ProgressBar) prepareNewLines() {
	if p.status != "" {
		p.lines = 2 // 1 line for task, 1 line for progress bar
	} else {
		p.lines = 1 // 1 line for progress bar only
	}

	// Emit enough new lines to account for the space the progress bar will take
	for i := 0; i < p.lines; i++ {
		fmt.Println()
	}
}

// redraw dynamically adjusts the bar's width and adapts to terminal resizing
func (p *ProgressBar) redraw() {

	// Calculate progress percentage and iterations/second
	current := atomic.LoadUint64(&p.current)
	elapsed := time.Since(p.start).Seconds()
	speed := float64(current) / elapsed

	// Format the speed based on the condition (< 25 -> 1 decimal place, >= 25 -> whole number)
	speedFormatted := formatSpeed(speed)

	var percentage int

	if p.max > 0 {
		// Bounded progress bar
		percentage = int((float64(current) / float64(p.max)) * 100)

		// Clamp the percentage to be within [0, 100]
		if percentage < 0 {
			percentage = 0
		} else if percentage > 100 {
			percentage = 100
		}
	}

	if p.isaTTY {
		// Writing to a regualar terminal - we can move the cursor
		// Move the cursor up by the number of lines the progress bar takes up
		cursorMoveUp(p.lines)

		// Get the current terminal width and dynamically adjust the bar size
		width := getTerminalWidth()
		availableWidth := width - 20 // Reserve space for percentage and speed display

		if p.max > 0 {
			// Bounded progress bar
			percentage := int((float64(current) / float64(p.max)) * 100)

			// Clamp the percentage to be within [0, 100]
			if percentage < 0 {
				percentage = 0
			} else if percentage > 100 {
				percentage = 100
			}

			bar := renderProgressBar(percentage, availableWidth)
			if p.status != "" {
				fmt.Fprintf(p.dest, "%s\n", p.status)
			}

			fmt.Fprintf(p.dest, "[%s] %3d%% (%s it/s)\n", bar, percentage, speedFormatted)

		} else {

			// Unbounded spinner
			spinner := renderSpinner(current)
			if p.status != "" {
				fmt.Fprintf(p.dest, "%s\n", p.status)
			}
			fmt.Fprintf(p.dest, "%s (%s it/s)\n", spinner, speedFormatted)
		}
	} else {
		// non-terminal oputput device (like maybe a log file or redirected output)
		if p.taskChanged {
			fmt.Fprintf(p.dest, "%s\n", p.status)
			p.taskChanged = false
		}

		if p.max > 0 {
			fmt.Fprintf(p.dest, "%3d%% ", percentage)
		}

		fmt.Fprintf(p.dest, "(%s it/s)\n", speedFormatted)
	}

}

// Helper function to render a bounded progress bar
func renderProgressBar(percentage int, width int) string {
	filledWidth := (percentage * width) / 100
	emptyWidth := width - filledWidth

	// Add the '>' at the leading edge of the bar, and use '=' for filled parts
	bar := stringRepeat("=", filledWidth-1) + ">"
	if percentage == 100 {
		bar = stringRepeat("=", width) // Use only '=' when complete
	}
	return fmt.Sprintf("%s%s", bar, stringRepeat(" ", emptyWidth))
}

// Helper function to render spinner for unbounded progress
func renderSpinner(current uint64) string {
	spinners := []string{"⠋", "⠙", "⠚", "⠒", "⠂", "⠂", "⠒", "⠲", "⠴", "⠦", "⠖", "⠒", "⠐", "⠐", "⠒", "⠓", "⠋"}
	return spinners[current%uint64(len(spinners))]
}

// Helper function to repeat a string n times
func stringRepeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

// Helper function to format the iterations/second based on the speed value
func formatSpeed(speed float64) string {
	if speed < 100 {
		return fmt.Sprintf("%.1f", speed) // One decimal place if speed < 100
	}
	return fmt.Sprintf("%.0f", speed) // Whole number if speed >= 100
}
