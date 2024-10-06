// Package yanprogress provides Yet ANother Progressbar!
//
// This is a simple one that provides a bar when the total value is known, or a spinner when it is not.
// It also detects whether the output device is a real tty. If it is not, then output is line by line suitable for log files
// or environments that capture the terminal streams e.g. containers or CI systems.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type (
	ProgressBar struct {
		max           uint64          // Maximum value (the 100% mark)
		current       uint64          // The current progress value
		spinnerPos    int             // Position of spinner if using spinner
		interval      time.Duration   // Interval between redraws
		dest          io.Writer       // destination eg stdout/stderr
		wg            *sync.WaitGroup // Waits on final redraw on completion
		lock          *sync.Mutex     // Sync calls to redraw
		start         time.Time       // Time when the bar was started
		status        string          // Status description (if any)
		isRunning     bool            // true after Start has been called
		statusChanged bool            // Set if SetStatus is called
		isaTTY        bool            // Set by constructor to know if we have a real TTY
		done          chan struct{}   // Channel to signal completion
	}

	optionFunc func(*ProgressBar)
)

// Spinner charaters
var spinners = func() []string {
	if runtime.GOOS != "windows" {
		return []string{"⠋", "⠙", "⠚", "⠒", "⠂", "⠂", "⠒", "⠲", "⠴", "⠦", "⠖", "⠒", "⠐", "⠐", "⠒", "⠓", "⠋"}
	} else {
		return []string{"\\", "|", "/", "-"}
	}
}()

// WithWriter sets the output stream for the bar (default os.Stderr)
func WithWriter(w io.Writer) optionFunc {
	return func(p *ProgressBar) {
		p.dest = w
	}
}

// NewProgressBar creates a new progress bar.
//
// Set max to zero for an unbounded spinner, else it is the value that represents 100% complete.
func NewProgressBar(max uint64, redrawInterval time.Duration, opts ...optionFunc) *ProgressBar {
	p := &ProgressBar{
		max:      max,
		interval: redrawInterval,
		start:    time.Now(),
		done:     make(chan struct{}),
		dest:     os.Stderr,
		wg:       &sync.WaitGroup{},
		lock:     &sync.Mutex{},
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

// Inc increments the progress bar's value by one.
func (p *ProgressBar) Inc() {
	atomic.AddUint64(&p.current, 1)
}

// Set sets the progress bar's value to a specific value.
func (p *ProgressBar) Set(value uint64) {
	atomic.StoreUint64(&p.current, value)
}

// Start hides the cursor and starts drawing the progress bar.
func (p *ProgressBar) Start() {

	fmt.Fprintf(p.dest, "\n")
	if p.isaTTY {
		cursorHide()
	}

	// Emit new lines to avoid overwriting existing terminal content
	p.prepareNewLines()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
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

// Complete marks the progress as complete and shows the cursor.
//
// It also cancels the goroutine responsible for redrawing the bar.
func (p *ProgressBar) Complete() {

	p.isRunning = false

	if p.max > 0 {
		// Set the current progress to the max explicitly
		atomic.StoreUint64(&p.current, uint64(p.max))
	}

	// Redraw one last time at exactly 100%
	//p.redraw()

	close(p.done)
	p.wg.Wait()

	if p.isaTTY {
		fmt.Fprintln(p.dest)
		cursorShow() // Show the cursor on completion
	}
}

// SetStatus sets the status line for the progress bar.
//
// Use this to indicate to users what is currently happening.
func (p *ProgressBar) SetStatus(status string) {

	p.status = strings.ReplaceAll(strings.TrimSpace(status), "\n", " ")
	p.statusChanged = true

	if p.isRunning {
		p.redraw()
	}
}

// prepareNewLines emits enough new lines to ensure the progress bar doesn't overwrite existing terminal content
func (p *ProgressBar) prepareNewLines() {
	fmt.Fprintln(p.dest)
	fmt.Fprintln(p.dest)
}

// redraw dynamically adjusts the bar's width and adapts to terminal resizing
func (p *ProgressBar) redraw() {

	p.lock.Lock()
	defer p.lock.Unlock()

	// Calculate progress percentage and iterations/second
	current := atomic.LoadUint64(&p.current)
	elapsed := time.Since(p.start).Seconds()
	speed := float64(current) / elapsed

	// Format the speed based on the condition (< 100 -> 1 decimal place, >= 100 -> whole number)
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
		p.renderTerminal(percentage, speedFormatted)
	} else {
		// non-terminal output device (like maybe a log file or redirected output)
		p.renderNonTerminal(percentage, speedFormatted)
	}
}

// renderTerminal handles redraw for tty device.
func (p *ProgressBar) renderTerminal(percentage int, speedFormatted string) {
	// Writing to a regualar terminal - we can move the cursor
	// Move the cursor up by the number of lines the progress bar takes up
	cursorMoveUp(2)

	// Get the current terminal width and dynamically adjust the bar size
	width := getTerminalWidth()

	// Reserve space for percentage and speed display
	availableWidth := width - 20

	if p.statusChanged {
		fmt.Fprint(p.dest, "\r"+stringRepeat(" ", width-1)+"\r")
		p.statusChanged = false
	}

	fmt.Fprintf(p.dest, "%s\n", ellipsize(p.status, width))

	if p.max > 0 {
		// Percentage bar
		bar := renderProgressBar(percentage, availableWidth)
		fmt.Fprintf(p.dest, "[%s] %3d%% (%s it/s)\n", bar, percentage, speedFormatted)

	} else {
		// Unbounded spinner
		spinner := spinners[p.spinnerPos]
		p.spinnerPos = (p.spinnerPos + 1) % len(spinners)
		fmt.Fprintf(p.dest, "%s (%s it/s)\n", spinner, speedFormatted)
	}
}

// renderNonTerminal handles redraw when output device is not a tty.
func (p *ProgressBar) renderNonTerminal(percentage int, speedFormatted string) {
	// When output is not tty, print to a new line
	if p.statusChanged {
		fmt.Fprintf(p.dest, "%s\n", p.status)
		p.statusChanged = false
	}

	if p.max > 0 {
		fmt.Fprintf(p.dest, "%3d%% ", percentage)
	}

	fmt.Fprintf(p.dest, "(%s it/s)\n", speedFormatted)
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

// ellipsize function truncates a string at the nearest word boundary to fit within the given width
func ellipsize(s string, width int) string {

	// Check if the string is shorter than or equal to the width
	if len(s) <= width {
		return s
	}

	if width <= 3 {
		return "..."
	}

	trimWidth := width - 3
	var result strings.Builder

	for _, word := range strings.Fields(s) {
		// Check if adding the next word would exceed the limit
		if result.Len()+len(word)+1 > trimWidth {
			break
		}
		if result.Len() > 0 {
			result.WriteByte(' ') // Add a space between words
		}
		result.WriteString(word)
	}

	// Append ellipsis
	result.WriteString("...")

	return result.String()
}
