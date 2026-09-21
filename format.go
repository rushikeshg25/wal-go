package walgo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/rushikeshg25/wal-go/pb"
	"google.golang.org/protobuf/proto"
	"hash/crc32"
	"io"
	"os"
)

const maxRecord = 64 << 20

var magic = []byte{'W', 'A', 'L', 'G', 1, 0, 16, 0}

func checksum(seq uint64, data []byte) uint32 {
	h := crc32.NewIEEE()
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], seq)
	h.Write(b[:])
	h.Write(data)
	return h.Sum32()
}
func writeFull(f *os.File, b []byte, off int64) error {
	for len(b) > 0 {
		n, e := f.WriteAt(b, off)
		b = b[n:]
		off += int64(n)
		if e != nil {
			return e
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
func scan(f *os.File, repair bool) (records []*pb.WAL_Entry, base, last uint64, end int64, err error) {
	stat, e := f.Stat()
	if e != nil {
		err = e
		return
	}
	size := stat.Size()
	var h [16]byte
	if size < 16 {
		err = errors.New("truncated WAL header")
		return
	}
	if _, err = f.ReadAt(h[:], 0); err != nil {
		return
	}
	if !bytes.Equal(h[:8], magic) {
		err = errors.New("unsupported WAL format; legacy files require explicit migration")
		return
	}
	base = binary.LittleEndian.Uint64(h[8:])
	last = base
	end = 16
	for end < size {
		var header [4]byte
		remaining := size - end
		if remaining >= 4 {
			if _, err = f.ReadAt(header[:], end); err != nil {
				return
			}
		}
		n := int64(binary.LittleEndian.Uint32(header[:]))
		if n > maxRecord {
			err = fmt.Errorf("oversized WAL record at %d", end)
			return
		}
		if remaining < 4 || n > remaining-4 {
			if !repair {
				err = io.ErrUnexpectedEOF
				return
			}
			if err = f.Truncate(end); err == nil {
				err = f.Sync()
			}
			return
		}
		data := make([]byte, n)
		if n > 0 {
			if _, err = f.ReadAt(data, end+4); err != nil {
				return
			}
		}
		entry := new(pb.WAL_Entry)
		if err = proto.Unmarshal(data, entry); err != nil {
			return
		}
		if last == ^uint64(0) || entry.LogSequenceNumber != last+1 || entry.CRC != checksum(entry.LogSequenceNumber, entry.Data) {
			err = fmt.Errorf("checksum or sequence corruption at %d", end)
			return
		}
		records = append(records, entry)
		last = entry.LogSequenceNumber
		end += 4 + n
	}
	return
}
