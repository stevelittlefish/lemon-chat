package server

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// modelErrorLog is the always-on sink for failed model requests. Unlike the
// verbose token log (every request, opt-in), it writes only when a request
// fails, so a live process can leave it on and the file only ever contains
// failures. The underlying file is opened lazily on the first failure and
// writes are serialised, so it is safe to share the same sink across every
// concurrent request. Each failure arrives as one already-formatted Write from
// the provider, so a timestamped header is all this adds.
type modelErrorLog struct {
	path string
	mu   sync.Mutex
	file *os.File
	off  bool // opening failed once; stop trying
}

func newModelErrorLog(dataDir string) *modelErrorLog {
	return &modelErrorLog{path: filepath.Join(dataDir, "model_errors.log")}
}

func (m *modelErrorLog) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.off {
		return len(p), nil
	}
	if m.file == nil {
		f, err := os.OpenFile(m.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			m.off = true
			log.Printf("model error log: could not open %s: %v", m.path, err)
			return len(p), nil
		}
		m.file = f
	}
	fmt.Fprintf(m.file, "\n########## %s ##########", time.Now().Format(time.RFC3339))
	if _, err := m.file.Write(p); err != nil {
		log.Printf("model error log: write failed: %v", err)
	}
	return len(p), nil
}

// Close closes the underlying file if it was ever opened.
func (m *modelErrorLog) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.file != nil {
		err := m.file.Close()
		m.file = nil
		return err
	}
	return nil
}

var _ io.Writer = (*modelErrorLog)(nil)
