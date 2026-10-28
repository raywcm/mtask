package daemon

import (
	"os"
	"path/filepath"
	"syscall"
)

const logLimit = 8 * 1024 * 1024

// Two generations bound disk usage per task to approximately 16 MiB.
type outputLog struct {
	file *os.File
	path string
	size int64
}

func newOutputLog(path string) (*outputLog, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	return &outputLog{file: f, path: path, size: info.Size()}, nil
}
func (l *outputLog) Write(p []byte) (int, error) {
	if l.size+int64(len(p)) > logLimit {
		if err := l.file.Close(); err != nil {
			return 0, err
		}
		if err := os.Rename(l.path, l.path+".1"); err != nil {
			return 0, err
		}
		f, err := os.OpenFile(filepath.Clean(l.path), os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
		if err != nil {
			return 0, err
		}
		l.file = f
		l.size = 0
	}
	n, err := l.file.Write(p)
	l.size += int64(n)
	return n, err
}
func (l *outputLog) Close() error { return l.file.Close() }
