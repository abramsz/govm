package download

import (
	"fmt"
	"io"
	"time"
)

// progressReader wraps an io.Reader and prints download progress to w.
type progressReader struct {
	reader    io.Reader
	total     int64
	current   int64
	lastPrint time.Time
	out       io.Writer
}

func newProgressReader(r io.Reader, total int64, out io.Writer) *progressReader {
	return &progressReader{
		reader:    r,
		total:     total,
		lastPrint: time.Now(),
		out:       out,
	}
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	p.current += int64(n)

	if time.Since(p.lastPrint) > 300*time.Millisecond || err == io.EOF {
		p.print()
		p.lastPrint = time.Now()
	}

	return n, err
}

func (p *progressReader) print() {
	if p.total <= 0 {
		mb := float64(p.current) / (1024 * 1024)
		fmt.Fprintf(p.out, "\r  %.1f MB downloaded", mb)
		return
	}

	pct := float64(p.current) / float64(p.total) * 100
	mb := float64(p.current) / (1024 * 1024)
	totalMB := float64(p.total) / (1024 * 1024)
	fmt.Fprintf(p.out, "\r  %.1f / %.1f MB (%.0f%%)", mb, totalMB, pct)
}
