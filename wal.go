package walgo

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/rushikeshg25/wal-go/pb"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrClosed = errors.New("wal: closed")
var ErrPoisoned = errors.New("wal: poisoned")

type WAL struct {
	directory               string
	currentFile             *os.File
	lock                    sync.Mutex
	lastSequenceNo          uint64
	maxFileSize             int64
	maxFiles, currentFileNo int
	end                     int64
	closed                  bool
	poison                  error
	stop, done              chan struct{}
}

func ids(dir string) ([]int, error) {
	entries, e := os.ReadDir(dir)
	if e != nil {
		return nil, e
	}
	out := []int{}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "wal-") {
			continue
		}
		n, e := strconv.Atoi(strings.TrimPrefix(entry.Name(), "wal-"))
		if e != nil || n < 0 || entry.IsDir() || entry.Name() != fmt.Sprintf("wal-%d", n) {
			return nil, fmt.Errorf("invalid segment %q", entry.Name())
		}
		out = append(out, n)
	}
	sort.Ints(out)
	return out, nil
}
func (w *WAL) path(id int) string { return filepath.Join(w.directory, fmt.Sprintf("wal-%d", id)) }
func syncDir(dir string) error {
	f, e := os.Open(dir)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
func (w *WAL) create(id int) error {
	f, e := os.OpenFile(w.path(id), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	h := make([]byte, 16)
	copy(h, magic)
	binary.LittleEndian.PutUint64(h[8:], w.lastSequenceNo)
	if e = writeFull(f, h, 0); e == nil {
		e = f.Sync()
	}
	if e == nil {
		e = syncDir(w.directory)
	}
	if e != nil {
		f.Close()
		return e
	}
	w.currentFile = f
	w.currentFileNo = id
	w.end = 16
	return nil
}
func WALInit(directory string, maxFileSize int64, maxFiles int) (*WAL, error) {
	if maxFileSize < 64 || maxFiles < 1 {
		return nil, errors.New("maxFileSize must be >=64 and maxFiles >=1")
	}
	if e := os.MkdirAll(directory, 0700); e != nil {
		return nil, e
	}
	list, e := ids(directory)
	if e != nil {
		return nil, e
	}
	w := &WAL{directory: directory, maxFileSize: maxFileSize, maxFiles: maxFiles, stop: make(chan struct{}), done: make(chan struct{})}
	for i, id := range list {
		f, e := os.OpenFile(w.path(id), os.O_RDWR, 0600)
		if e != nil {
			return nil, e
		}
		_, base, last, end, e := scan(f, i == len(list)-1)
		if e != nil {
			f.Close()
			return nil, e
		}
		if i > 0 && (id != list[i-1]+1 || base != w.lastSequenceNo) {
			f.Close()
			return nil, errors.New("noncontiguous WAL segments")
		}
		w.lastSequenceNo = last
		w.end = end
		w.currentFileNo = id
		if i == len(list)-1 {
			w.currentFile = f
		} else {
			f.Close()
		}
	}
	if len(list) == 0 {
		if e = w.create(0); e != nil {
			return nil, e
		}
	}
	if e = w.retain(); e != nil {
		w.currentFile.Close()
		return nil, e
	}
	go w.syncLoop()
	return w, nil
}
func (w *WAL) check() error {
	if w.closed {
		return ErrClosed
	}
	if w.poison != nil {
		return errors.Join(ErrPoisoned, w.poison)
	}
	return nil
}
func (w *WAL) fail(e error) error {
	if e != nil {
		w.poison = e
	}
	return e
}
func (w *WAL) retain() error {
	list, e := ids(w.directory)
	if e != nil {
		return e
	}
	removed := false
	for len(list) > w.maxFiles {
		if e = os.Remove(w.path(list[0])); e != nil {
			return e
		}
		list = list[1:]
		removed = true
	}
	if removed {
		return syncDir(w.directory)
	}
	return nil
}
func (w *WAL) append(entry *pb.WAL_Entry) error {
	if e := w.check(); e != nil {
		return e
	}
	if len(entry.Data) > maxRecord-64 || w.lastSequenceNo == ^uint64(0) {
		return errors.New("WAL record or sequence limit")
	}
	if entry.LogSequenceNumber != w.lastSequenceNo+1 || entry.CRC != checksum(entry.LogSequenceNumber, entry.Data) {
		return errors.New("invalid sequence or checksum")
	}
	data, e := proto.Marshal(entry)
	if e != nil {
		return e
	}
	if int64(len(data))+20 > w.maxFileSize {
		return errors.New("record exceeds segment capacity")
	}
	if w.end+4+int64(len(data)) > w.maxFileSize {
		if e = w.currentFile.Sync(); e != nil {
			return w.fail(e)
		}
		old := w.currentFile
		if e = w.create(w.currentFileNo + 1); e != nil {
			return w.fail(e)
		}
		if e = old.Close(); e != nil {
			return w.fail(e)
		}
		if e = w.retain(); e != nil {
			return w.fail(e)
		}
	}
	record := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(record, uint32(len(data)))
	copy(record[4:], data)
	if e = writeFull(w.currentFile, record, w.end); e != nil {
		return w.fail(e)
	}
	w.end += int64(len(record))
	w.lastSequenceNo = entry.LogSequenceNumber
	return nil
}
func (w *WAL) WriteLog(data []byte) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	seq := w.lastSequenceNo + 1
	return w.append(&pb.WAL_Entry{LogSequenceNumber: seq, Data: data, CRC: checksum(seq, data)})
}
func (w *WAL) WriteWALEntryToBuffer(entry *pb.WAL_Entry) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	if entry == nil {
		return errors.New("nil entry")
	}
	return w.append(entry)
}
func (w *WAL) Sync() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	if e := w.check(); e != nil {
		return e
	}
	return w.fail(w.currentFile.Sync())
}
func (w *WAL) syncLoop() {
	defer close(w.done)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			_ = w.Sync()
		}
	}
}
func (w *WAL) Close() error {
	w.lock.Lock()
	if w.closed {
		w.lock.Unlock()
		<-w.done
		return nil
	}
	w.closed = true
	close(w.stop)
	e := errors.Join(w.poison, w.currentFile.Sync(), w.currentFile.Close())
	w.lock.Unlock()
	<-w.done
	return e
}
func (w *WAL) ReadAllLogsFromCurrentFile() ([]*pb.WAL_Entry, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	if e := w.check(); e != nil {
		return nil, e
	}
	records, _, _, _, e := scan(w.currentFile, false)
	return records, e
}
func (w *WAL) ReadLogsFromFile(file *os.File) ([]*pb.WAL_Entry, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	if e := w.check(); e != nil {
		return nil, e
	}
	records, _, _, _, e := scan(file, false)
	return records, e
}
func (w *WAL) ReadAll() ([]*pb.WAL_Entry, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	if e := w.check(); e != nil {
		return nil, e
	}
	list, e := ids(w.directory)
	if e != nil {
		return nil, e
	}
	var out []*pb.WAL_Entry
	for _, id := range list {
		f, e := os.Open(w.path(id))
		if e != nil {
			return nil, e
		}
		records, _, _, _, e := scan(f, false)
		f.Close()
		if e != nil {
			return nil, e
		}
		out = append(out, records...)
	}
	return out, nil
}

// Repair only truncates a structurally incomplete tail in the current segment.
func (w *WAL) Repair() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	if e := w.check(); e != nil {
		return e
	}
	_, _, last, end, e := scan(w.currentFile, true)
	if e != nil {
		return w.fail(e)
	}
	w.lastSequenceNo = last
	w.end = end
	return nil
}
func InitExisingWAL(files []os.DirEntry, directory string) (int, error) {
	list, e := ids(directory)
	if e != nil {
		return 0, e
	}
	if len(list) == 0 {
		return 0, errors.New("no WAL segments")
	}
	return list[len(list)-1], nil
}
