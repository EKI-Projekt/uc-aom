// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package fileserver

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	connectionAttempts = 5
)

type RecoveryStatus uint64

// progress msg status enums
const (
	IDLE RecoveryStatus = iota
	START
	RUN
	SUCCESS
	FAILURE
	DOWNLOAD
	DONE
	SUBPROCESS
	PROGRESS
)

type SourceType uint64

// progress msg source enums
const (
	SOURCE_UNKNOWN SourceType = iota
	SOURCE_WEBSERVER
	SOURCE_SURICATTA
	SOURCE_DOWNLOADER
	SOURCE_LOCAL
	SOURCE_CHUNKS_DOWNLOADER
)

// ProgressMsg is the struct wrapping the swupdate progress message
// https://sbabic.github.io/swupdate/progress.html
type ProgressMsg struct {
	Magic         uint32
	Status        RecoveryStatus
	DwlPercent    uint32
	DwlBytes      uint64
	NSteps        uint32
	CurStep       uint32
	CurPercentage uint32
	CurImage      [256]byte
	HndName       [64]byte
	Source        SourceType
	InfoLen       uint32
	Info          [2048]byte
}

type SWUpdateWatcher interface {
	Connect() error
	ListenOnStatus() (<-chan RecoveryStatus, <-chan error)
}

// swUpdateWatcher connects to the swupdate unix socket
// and send the incoming progress messages through a channel
type swUpdateWatcher struct {
	SocketPath string
	conn       net.Conn
}

// NewSWUpdateWatcher returns a new SWUpdateWatcher
func NewSWUpdateWatcher(socketPath string) *swUpdateWatcher {
	return &swUpdateWatcher{
		SocketPath: socketPath,
	}
}

// Connect connects to the swupdate socket
func (u *swUpdateWatcher) Connect() error {
	sockAddr := u.SocketPath
	var err error
	var connection net.Conn
	for i := 0; i < connectionAttempts; i++ {
		connection, err = net.Dial("unix", sockAddr)
		if connection != nil {
			u.conn = connection
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		log.Errorf("%s", err.Error())
		return err
	}

	return nil
}

// ListenOnStatus listens to the incoming messages from the swupdate socket
func (u *swUpdateWatcher) ListenOnStatus() (<-chan RecoveryStatus, <-chan error) {
	msgChan := make(chan RecoveryStatus)
	errChan := make(chan error)
	go func() {
		for {
			buf := make([]byte, 2416)
			mLen, err := u.conn.Read(buf)
			if err != nil {
				errChan <- err
			}

			if mLen > 0 {
				msg := ProgressMsg{}
				b := bytes.NewBuffer(buf)
				binary.Read(b, binary.LittleEndian, &msg)
				msgChan <- msg.Status
			}
		}
	}()
	return msgChan, errChan
}
