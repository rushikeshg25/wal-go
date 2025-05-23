package walgo

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"sync"
	"time"
)

type WAL struct {
	directory      string
	currentFile    *os.File
	lock           sync.Mutex
	lastSequenceNo uint64
	bufWriter      *bufio.Writer
	syncTimer      *time.Timer
	shouldFsync    bool
	maxFileSize    int64
	maxLogs        int
	currentFileNo  int
	ctx            context.Context
	cancel         context.CancelFunc
}

const (
	walFilenamePrefix = "wal-"
	syncInterval      = 200 * time.Millisecond
	maxSegmentSize    = 1024 * 1024 * 1024
)

func WALInit(directory string, maxFileSize int64, maxLogs int, shouldFsync bool) (*WAL, error) {
	wl := &WAL{
		directory:      directory,
		currentFile:    nil,
		lastSequenceNo: 0,
		bufWriter:      nil,
		syncTimer:      time.NewTimer(syncInterval),
		shouldFsync:    shouldFsync,
		maxFileSize:    maxFileSize,
		maxLogs:        maxLogs,
		currentFileNo:  0,
	}

	var file *os.File
	var err error

	if err = os.MkdirAll(directory, 0755); err != nil {
		return nil, err
	}

	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		file, err = os.Create(directory + "/" + walFilenamePrefix + strconv.Itoa(wl.currentFileNo))
		if err != nil {
			return nil, err
		}

	} else {
		file, err = readLastLogs(files, directory)
		if err != nil {
			return nil, err
		}
	}
	wl.currentFile = file
	wl.bufWriter = bufio.NewWriter(file)
	return wl, nil
}

func readLastLogs(files []os.DirEntry, directory string) (*os.File, error) {
	lastFile := files[len(files)-1].Name()
	file, err := os.OpenFile(directory+"/"+lastFile, os.O_RDWR, 0755)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (wl *WAL) WriteLog() {}

// func (wl *WAL) Close() error {}

// func (wl *WAL) Sync() error {}
