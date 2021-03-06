// Copyright (c) 2016 Intel Corporation. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package posix

import (
	"crypto/sha1"
	"hash"
	"io"

	"github.com/pkg/errors"
)

type (
	// ChecksumWriter wraps an io.WriterAt and updates the checksum
	// with every write.
	ChecksumWriter interface {
		WriteAt([]byte, int64) (int, error)
		Sum() []byte
	}

	// Sha1HashWriter implements ChecksumWriter and uses the SHA1
	// algorithm to calculate the file checksum
	Sha1HashWriter struct {
		dest  io.WriterAt
		cksum hash.Hash
	}

	// NoopHashWriter implements ChecksumWriter but doesn't
	// actually calculate a checksum
	NoopHashWriter struct {
		dest io.WriterAt
	}
)

// NewSha1HashWriter returns a new Sha1HashWriter
func NewSha1HashWriter(dest io.WriterAt) ChecksumWriter {
	return &Sha1HashWriter{
		dest:  dest,
		cksum: sha1.New(),
	}
}

// WriteAt updates the checksum and writes the byte slice at offset
func (hw *Sha1HashWriter) WriteAt(b []byte, off int64) (int, error) {
	_, err := hw.cksum.Write(b)
	if err != nil {
		return 0, errors.Wrap(err, "updating checksum failed")
	}
	return hw.dest.WriteAt(b, off)
}

// Sum returns the checksum
func (hw *Sha1HashWriter) Sum() []byte {
	return hw.cksum.Sum(nil)
}

// NewNoopHashWriter returns a new NoopHashWriter
func NewNoopHashWriter(dest io.WriterAt) ChecksumWriter {
	return &NoopHashWriter{
		dest: dest,
	}
}

// WriteAt writes the byte slice at offset
func (hw *NoopHashWriter) WriteAt(b []byte, off int64) (int, error) {
	return hw.dest.WriteAt(b, off)
}

// Sum returns a dummy checksum
func (hw *NoopHashWriter) Sum() []byte {
	return []byte{}
}
