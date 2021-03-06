// Copyright (c) 2016 Intel Corporation. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package posix

import "io"

// CopyAt copies n bytes at offset of src file to same offset at destination.
//
// Like ReadAt, CopyAt returns non-nil error when writen < n, and at end of file
// that error is io.EOF.
// Implementation inspired by io.Copy()
func CopyAt(dst io.WriterAt, src io.ReaderAt, offset uint64, n uint64) (written uint64, err error) {
	var bufSize uint64 = 1024 * 1024
	if bufSize > n {
		bufSize = n
	}
	buf := make([]byte, bufSize)
	for written < n {
		nr, er := src.ReadAt(buf, int64(offset+written))
		if nr > 0 {
			nw, ew := dst.WriteAt(buf[0:nr], int64(offset+written))
			if nw > 0 {
				written += uint64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
		}
		if er == io.EOF && written == n {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return
}
