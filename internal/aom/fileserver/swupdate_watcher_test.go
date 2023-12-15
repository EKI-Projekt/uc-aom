// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package fileserver_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"os"
	"testing"
	"u-control/uc-aom/internal/aom/fileserver"
)

const mockSockAddr = "/tmp/mockswupdateprog"

func mockProgressMsg(t *testing.T) []byte {
	msg := fileserver.ProgressMsg{
		Status: fileserver.SUCCESS,
	}
	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.LittleEndian, msg)
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func mockSWUpdateSocket(t *testing.T) {
	if err := os.RemoveAll(mockSockAddr); err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("unix", mockSockAddr)
	if err != nil {
		t.Fatal("listen error:", err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			t.Fatal("accept error:", err)
		}
		mockedProgressMsg := mockProgressMsg(t)
		_, err = conn.Write(mockedProgressMsg)
		if err != nil {
			t.Fatal(err)
		}
		err = conn.Close()
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if err := os.RemoveAll(mockSockAddr); err != nil {
		t.Fatal(err)
	}
}

func TestSWUpdateWatcher_Connection(t *testing.T) {
	// arrange
	go func() {
		mockSWUpdateSocket(t)
	}()
	watcher := fileserver.NewSWUpdateWatcher(mockSockAddr)

	// act
	err := watcher.Connect()

	// assert
	if err != nil {
		t.Errorf("Could not connect to socket: %v", err)
	}
}

func TestSWUpdateWatcher_ConnectionError(t *testing.T) {
	// arrange
	go func() {
		mockSWUpdateSocket(t)
	}()
	watcher := fileserver.NewSWUpdateWatcher("invalidPath")

	// act
	err := watcher.Connect()

	// assert
	if err == nil {
		t.Error("Expected a connection error")
	}
}

func TestSWUpdateWatcher_ConnectionClosed(t *testing.T) {
	// arrange
	go func() {
		mockSWUpdateSocket(t)
	}()
	watcher := fileserver.NewSWUpdateWatcher(mockSockAddr)
	err := watcher.Connect()
	if err != nil {
		t.Fatal("Failed to connect to the mock socket")
	}

	msgChan, errChan := watcher.ListenOnStatus()
	status := <-msgChan // needed to read the message channel as well otherwise the execution is blocked
	err = <-errChan

	// assert
	if err == nil || err != io.EOF {
		t.Errorf("Expected an io.EOF error after the socket closes the client connection")
	}

	if status != fileserver.SUCCESS {
		t.Errorf("Expected a success message from the socket")
	}
}

func TestSWUpdateWatcher_Messages(t *testing.T) {
	// arrange
	go func() {
		mockSWUpdateSocket(t)
	}()
	watcher := fileserver.NewSWUpdateWatcher(mockSockAddr)
	err := watcher.Connect()

	if err != nil {
		t.Errorf("Could not connect to socket %v", err)
	}

	// act
	msgChan, errChan := watcher.ListenOnStatus()
	status := <-msgChan
	e := <-errChan

	// assert
	if e != nil && e != io.EOF { // ignore EOF since the mocked socket closes the connection after sending progress msg
		t.Errorf("Failed reading socket messages:%v", e)
	}

	if status != fileserver.SUCCESS {
		t.Errorf("Expected a success message from the socket")
	}
}
