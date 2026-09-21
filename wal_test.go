package walgo

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"testing"
)

func TestRotationRetentionRestart(t *testing.T) {
	dir := t.TempDir()
	w, e := WALInit(dir, 128, 2)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 100; i++ {
		if e = w.WriteLog(bytes.Repeat([]byte{byte(i)}, 20)); e != nil {
			t.Fatal(e)
		}
	}
	if e = w.Close(); e != nil {
		t.Fatal(e)
	}
	w, e = WALInit(dir, 128, 2)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if w.lastSequenceNo != 100 {
		t.Fatal(w.lastSequenceNo)
	}
	w.WriteLog([]byte("after"))
	records, e := w.ReadAll()
	if e != nil || records[len(records)-1].LogSequenceNumber != 101 {
		t.Fatal(records, e)
	}
	list, _ := ids(dir)
	if len(list) > 2 {
		t.Fatal("retention")
	}
}
func TestTailAndCorruption(t *testing.T) {
	dir := t.TempDir()
	w, _ := WALInit(dir, 1024, 3)
	w.WriteLog([]byte("good"))
	path := w.path(0)
	w.Close()
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	f.Write([]byte{1, 2})
	f.Close()
	w, e := WALInit(dir, 1024, 3)
	if e != nil {
		t.Fatal(e)
	}
	r, e := w.ReadAll()
	if e != nil || len(r) != 1 {
		t.Fatal(r, e)
	}
	w.Close()
	data, _ := os.ReadFile(path)
	data[len(data)-1] ^= 1
	os.WriteFile(path, data, 0600)
	if w, e = WALInit(dir, 1024, 3); e == nil {
		w.Close()
		t.Fatal("accepted corruption")
	}
}
func TestConcurrentWritesClose(t *testing.T) {
	w, _ := WALInit(t.TempDir(), 1024, 100)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if e := w.WriteLog([]byte("x")); e != nil {
					t.Error(e)
				}
			}
		}()
	}
	wg.Wait()
	r, e := w.ReadAll()
	if e != nil || len(r) != 200 {
		t.Fatal(len(r), e)
	}
	w.Close()
	w.Close()
	if !errors.Is(w.WriteLog(nil), ErrClosed) {
		t.Fatal("closed write")
	}
}
func TestEmptySegmentPreservesSequence(t *testing.T) {
	dir := t.TempDir()
	w, _ := WALInit(dir, 128, 1)
	w.WriteLog([]byte("a"))
	w.lock.Lock()
	old := w.currentFile
	if e := w.create(1); e != nil {
		t.Fatal(e)
	}
	old.Close()
	w.retain()
	w.lock.Unlock()
	w.Close()
	w, e := WALInit(dir, 128, 1)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if w.lastSequenceNo != 1 {
		t.Fatal("lost base sequence")
	}
	w.WriteLog([]byte("b"))
	r, _ := w.ReadAll()
	if len(r) != 1 || r[0].LogSequenceNumber != 2 {
		t.Fatal(r)
	}
}
