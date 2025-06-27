package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type ProgressBar struct {
	total     int
	current   int
	startTime time.Time
	mu        sync.Mutex
	width     int
	lastPrint time.Time
}

func New(total int) *ProgressBar {
	return &ProgressBar{
		total:     total,
		current:   0,
		startTime: time.Now(),
		width:     50,
		lastPrint: time.Now(),
	}
}

func (p *ProgressBar) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current++

	if time.Since(p.lastPrint) > 100*time.Millisecond || p.current == p.total {
		p.render()
		p.lastPrint = time.Now()
	}
}

func (p *ProgressBar) render() {
	percentage := float64(p.current) / float64(p.total)
	filled := int(percentage * float64(p.width))

	// Handle case where current exceeds total (duration-based tests)
	if filled > p.width {
		filled = p.width
	}

	remaining := p.width - filled
	if remaining < 0 {
		remaining = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", remaining)

	elapsed := time.Since(p.startTime)
	var eta time.Duration
	if p.current > 0 {
		eta = time.Duration(float64(elapsed)*(float64(p.total)/float64(p.current)) - float64(elapsed))
	}

	var rps float64
	if elapsed.Seconds() > 0 {
		rps = float64(p.current) / elapsed.Seconds()
	}

	fmt.Printf("\r[%s] %d/%d (%.1f%%) | %.1f req/s | Elapsed: %v | ETA: %v",
		bar,
		p.current,
		p.total,
		percentage*100,
		rps,
		elapsed.Round(time.Second),
		eta.Round(time.Second),
	)
}

func (p *ProgressBar) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.render()
	fmt.Println()
}
