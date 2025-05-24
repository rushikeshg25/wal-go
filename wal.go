package walgo

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/rushikeshg25/wal-go/pb"
)

type WAL struct {
	directory      string
	currentFile    *os.File
	lock           sync.Mutex
	lastSequenceNo uint64
	bufWriter      *bufio.Writer
	syncTimer      *time.Timer
	maxFileSize    int64
	maxFiles       int
	currentFileNo  int
	ctx            context.Context
	cancel         context.CancelFunc
}

const (
	walFilenamePrefix = "wal-"
	syncInterval      = 200 * time.Millisecond
	maxSegmentSize    = 1024 * 1024 * 1024
)

func WALInit(directory string, maxFileSize int64, maxFiles int) (*WAL, error) {
	wl := &WAL{
		directory:      directory,
		currentFile:    nil,
		lastSequenceNo: 0,
		bufWriter:      nil,
		syncTimer:      time.NewTimer(syncInterval),
		maxFileSize:    maxFileSize,
		maxFiles:       maxFiles,
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
			wl.currentFile.Close()
			return nil, err
		}

	} else {
		num, err := InitExisingWAL(files, directory)
		if err != nil {
			return nil, err
		}
		file, err = os.OpenFile(directory+"/"+walFilenamePrefix+strconv.Itoa(num+1), os.O_RDWR, 0644)
		if err != nil {
			wl.currentFile.Close()
			return nil, err
		}
		wl.currentFileNo = num + 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	wl.currentFile = file
	wl.bufWriter = bufio.NewWriter(file)
	wl.ctx = ctx
	wl.cancel = cancel

	go wl.syncwithTimer()

	return wl, nil
}

func (wl *WAL) WriteLog(data []byte) error {

	if err := wl.checkCurrentFileSize(); err != nil {
		return err
	}

	wl.lock.Lock()
	defer wl.lock.Unlock()
	wl.lastSequenceNo++
	entry := &pb.WAL_Entry{
		LogSequenceNumber: wl.lastSequenceNo,
		Data:              data,
		CRC:               crc32.ChecksumIEEE(append(data, byte(wl.lastSequenceNo))),
	}
	return wl.WriteWALEntryToBuffer(entry)
}

func (wl *WAL) WriteWALEntryToBuffer(logEntry *pb.WAL_Entry) error {
	logEntryBytes := Marshal(logEntry)
	size := int32(len(logEntryBytes))
	if err := binary.Write(wl.bufWriter, binary.LittleEndian, size); err != nil {
		return err
	}
	_, err := wl.bufWriter.Write(logEntryBytes)
	return err
}

func (wl *WAL) Sync() error {
	err := wl.bufWriter.Flush()
	return err
}

func (wl *WAL) syncwithTimer() {
	for {
		select {
		case <-wl.syncTimer.C:
			wl.lock.Lock()
			err := wl.Sync()
			wl.lock.Unlock()
			if err != nil {
				fmt.Println("Sync failed")
			}
		case <-wl.ctx.Done():
			return
		}
	}
}

func (wl *WAL) checkCurrentFileSize() error {
	stat, err := wl.currentFile.Stat()
	if err != nil {
		return err
	}

	if stat.Size()+int64(wl.bufWriter.Buffered()) >= wl.maxFileSize {
		if err := wl.createNewWALFile(); err != nil {
			return err
		}
	}
	return nil
}

func (wl *WAL) createNewWALFile() error {
	if err := wl.Sync(); err != nil {
		return err
	}

	if err := wl.currentFile.Close(); err != nil {
		return err
	}

	wl.currentFileNo++
	file, err := os.Create(wl.directory + "/" + walFilenamePrefix + strconv.Itoa(wl.currentFileNo))
	if err != nil {
		return err
	}
	wl.currentFile = file
	wl.bufWriter = bufio.NewWriter(file)

	files, err := os.ReadDir(wl.directory)
	if err != nil {
		return err
	}
	if len(files) >= wl.maxFiles {
		err := wl.deleteOldestFile(files[0].Name())
		if err != nil {
			return err
		}
	}
	return nil
}

func (wl *WAL) deleteOldestFile(file string) error {
	err := os.Remove(wl.directory + "/" + file)
	return err
}

func (wl *WAL) Close() {
	wl.cancel()
	if err := wl.Sync(); err != nil {
		fmt.Println("Sync failed")
	}
	wl.currentFile.Close()
}

func (wl *WAL) ReadAllLogsFromCurrentFile() ([]*pb.WAL_Entry, error) {
	file, err := os.OpenFile(wl.directory+"/"+walFilenamePrefix+strconv.Itoa(wl.currentFileNo), os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	logs, err := wl.ReadLogsFromFile(file)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (wl *WAL) ReadLogsFromFile(file *os.File) ([]*pb.WAL_Entry, error) {
	var logs []*pb.WAL_Entry
	for {
		var size int32
		if err := binary.Read(file, binary.LittleEndian, &size); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		data := make([]byte, size)
		if _, err := io.ReadFull(file, data); err != nil {
			return nil, err
		}

		logEntry := UnMarshall(data)
		logs = append(logs, logEntry)
	}
	return logs, nil
}

func InitExisingWAL(files []os.DirEntry, directory string) (int, error) {
	lastFile := files[len(files)-1].Name()
	lastFileNo, err := strconv.Atoi(lastFile[len(lastFile)-1:])
	if err != nil {
		return 000, err
	}

	return lastFileNo, nil
}

func (wl *WAL) Repair() error {
	files, err := os.ReadDir(wl.directory)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		panic("No files found")
	} else {

	}

	return nil
}
