package walgo

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type WAL struct {
	directory           string
	currentSegment      *os.File
	lock                sync.Mutex
	lastSequenceNo      uint64
	bufWriter           *bufio.Writer
	syncTimer           *time.Timer
	shouldFsync         bool
	maxFileSize         int64
	maxSegments         int
	currentSegmentIndex int
	ctx                 context.Context
	cancel              context.CancelFunc
}

const (
	segmentFilenamePrefix = "segment"
	syncInterval = 200*time.Millisecond
)


func OpenWAL(directory string, enableFsync bool, maxFileSize int64, maxSegments int) (*WAL, error){
	if err:=os.MkdirAll(directory,0755);err!=nil{
		return nil,err
	}

	files,err:=filepath.Glob(filepath.Join(directory,segmentFilenamePrefix+"*"))
	if err!=nil{
		return nil,err
	}

	lastSegmentId:=0
	if len(files)>0{
		lastSegmentId,err=findLastSegmentIndexinFiles(files)
		if err!=nil{
			return nil,err
		}
	}else{
		file,err:=createSegmentFile(directory,0)
		if err!=nil{
			return nil,err
		}

		if err:=file.Close();err!=nil{
			return nil,err
		}
	}

	filePath:=filepath.Join(directory,fmt.Sprintf("%s%d",segmentFilenamePrefix,lastSegmentId))
	file,err:=os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err!=nil{
		return nil,err
	}

	ctx,cancel:=context.WithCancel(context.Background())

	wal:=&WAL{
		directory: directory,
		currentSegment: file,
		lastSequenceNo: 0,
		bufWriter: bufio.NewWriter(file),
		syncTimer: time.NewTimer(syncInterval),
		shouldFsync: enableFsync,
		maxFileSize: maxFileSize,
		maxSegments: maxSegments,
		currentSegmentIndex: lastSegmentId,
		ctx: ctx,
		cancel: cancel,
	}

	if wal.lastSequenceNo,err=wal.getLastSequenceNo();err!=nil{
		return nil,err
	}


    
	return wal,nil
	
}

func (wal *WAL) getLastSequenceNo() (uint64,error){
	entry, err := wal.getLastEntryInLog()
	if err != nil {
		return 0, err
	}

	if entry != nil {
		return entry.GetLogSequenceNumber(), nil
	}
	return 0, nil
}

func (wal *WAL) getLastEntryInLog() (*WAL_Entry, error) {
	file, err := os.OpenFile(wal.currentSegment.Name(), os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var previousSize int32
	var offset int64
	var entry *WAL_Entry

	for {
		var size int32
		if err := binary.Read(file, binary.LittleEndian, &size); err != nil {
			if err == io.EOF {
				// End of file reached, read the last entry at the saved offset.
				if offset == 0 {
					return entry, nil
				}

				if _, err := file.Seek(offset, io.SeekStart); err != nil {
					return nil, err
				}

				// Read the entry data.
				data := make([]byte, previousSize)
				if _, err := io.ReadFull(file, data); err != nil {
					return nil, err
				}

				entry, err = unmarshalAndVerifyEntry(data)
				if err != nil {
					return nil, err
				}

				return entry, nil
			}
			return nil, err
		}

		// Get current offset
		offset, err = file.Seek(0, io.SeekCurrent)
		previousSize = size

		if err != nil {
			return nil, err
		}

		// Skip to the next entry.
		if _, err := file.Seek(int64(size), io.SeekCurrent); err != nil {
			return nil, err
		}
	}
}