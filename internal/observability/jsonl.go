package observability

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	mu      sync.Mutex
	dataDir string
	enabled func() bool
}

func New(dataDir string, enabled func() bool) *Logger {
	return &Logger{dataDir: dataDir, enabled: enabled}
}

func (l *Logger) RunUsage(event string, fields map[string]any) error {
	return l.write("run-usage.jsonl", event, fields)
}

func (l *Logger) ChannelCall(event string, fields map[string]any) error {
	return l.write("channel-calls.jsonl", event, fields)
}

func (l *Logger) write(fileName, event string, fields map[string]any) error {
	if l == nil || l.enabled == nil || !l.enabled() {
		return nil
	}
	if fields == nil {
		fields = map[string]any{}
	}
	fields["event"] = event
	fields["ts"] = time.Now().UTC().Format(time.RFC3339Nano)

	content, err := json.Marshal(fields)
	if err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(l.dataDir, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(l.dataDir, fileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(content, '\n'))
	return err
}
