package service

import (
	"os"

	pkgsftp "github.com/pkg/sftp"
)

type guardedFile struct {
	file  *os.File
	check func() error
}

func (f *guardedFile) ReadAt(p []byte, offset int64) (int, error) {
	if err := f.check(); err != nil {
		return 0, err
	}
	return f.file.ReadAt(p, offset)
}

func (f *guardedFile) WriteAt(p []byte, offset int64) (int, error) {
	if err := f.check(); err != nil {
		return 0, err
	}
	return f.file.WriteAt(p, offset)
}

func (f *guardedFile) Close() error { return f.file.Close() }

type guardedList struct {
	pkgsftp.ListerAt
	check func() error
}

func (l *guardedList) ListAt(items []os.FileInfo, offset int64) (int, error) {
	if err := l.check(); err != nil {
		return 0, err
	}
	return l.ListerAt.ListAt(items, offset)
}
