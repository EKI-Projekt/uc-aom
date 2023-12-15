// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service_test

import (
	"context"
	"sync"
	"testing"
	"u-control/uc-aom/internal/aom/server"
	"u-control/uc-aom/internal/aom/service"
)

func initTransactionScheduler(t *testing.T) (*service.TransactionScheduler, *service.Service) {
	t.Helper()
	_, multiService, _ := server.NewServerUsingServiceMultiComponentMock()
	serviceMock := multiService.NewServiceUsingServiceMultiComponentMock()

	uut := service.NewTransactionScheduler()
	return uut, serviceMock
}

func TestCreateTransaction(t *testing.T) {
	// Arrange
	contextMock := context.Background()
	uut, serviceMock := initTransactionScheduler(t)

	// Act
	tx, err := uut.CreateTransaction(contextMock, serviceMock)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tx == nil {
		t.Fatalf("Expected transaction but got none.")
	}
}

func TestCreateTransactionAlreadyExist(t *testing.T) {
	// Arrange
	contextMock := context.Background()
	uut, serviceMock := initTransactionScheduler(t)

	// Act
	blockingTx, err := uut.CreateTransaction(contextMock, serviceMock)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if blockingTx == nil {
		t.Fatalf("Expected transaction but got nil")
	}

	tx, err := uut.CreateTransaction(contextMock, serviceMock)

	// Assert
	if err == nil {
		t.Fatalf("Expected error but got nil")
	}

	if tx != nil {
		t.Fatalf("Expected empty transaction but got '%v'", tx)
	}
}

func TestCreateTransactionIsClosed(t *testing.T) {
	// Arrange
	var waitGroup sync.WaitGroup

	contextMock := context.Background()
	uut, serviceMock := initTransactionScheduler(t)

	// Act
	tx, err := uut.CreateTransaction(contextMock, serviceMock)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Assert
	// assert transaction after context was closed
	waitGroup.Add(1)
	go func() {
		select {
		case <-tx.Done():
			tx = uut.GetTransaction()
			if tx.IsDone() != true {
				t.Errorf("Expected transaction to be closed")
			}
			waitGroup.Done()
		}
	}()

	// close context
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	waitGroup.Wait()
}

func TestCreateTransactionAfterOneWasClosed(t *testing.T) {
	// Arrange
	var waitGroup sync.WaitGroup

	uut, serviceMock := initTransactionScheduler(t)

	// Act
	firstTx, err := uut.CreateTransaction(context.Background(), serviceMock)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if firstTx == nil {
		t.Fatalf("Expected transaction but got nil")
	}

	// Assert
	// assert after closing
	waitGroup.Add(1)
	go func() {
		select {
		case <-firstTx.Done():
			secondTx, err := uut.CreateTransaction(context.Background(), serviceMock)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if secondTx == nil {
				t.Errorf("Expected transaction but got nil")
			}
			secondTx.Commit()
			waitGroup.Done()
		}
	}()

	// closing context
	err = firstTx.Commit()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	waitGroup.Wait()
}

func TestIsTransactionOpenFalse(t *testing.T) {
	// Arrange
	uut := service.NewTransactionScheduler()

	// Act
	result := uut.IsTransactionOpen()

	// Assert
	expectedResult := false
	if result != expectedResult {
		t.Fatalf("Expected result '%v', but got '%v'.", expectedResult, result)
	}
}

func TestIsTransactionOpenFalseAfterClosing(t *testing.T) {
	// Arrange
	var waitGroup sync.WaitGroup

	contextMock := context.Background()
	uut, serviceMock := initTransactionScheduler(t)
	tx, err := uut.CreateTransaction(contextMock, serviceMock)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Act
	resultBeforeClose := uut.IsTransactionOpen()
	expectedResultBeforeClose := true
	if resultBeforeClose != expectedResultBeforeClose {
		t.Fatalf("Expected result '%v', but got '%v'.", expectedResultBeforeClose, resultBeforeClose)
	}

	// Assert
	// assert after closing
	waitGroup.Add(1)
	go func() {
		select {
		case <-tx.Done():
			resultAfterClose := uut.IsTransactionOpen()
			expectedResultAfterClose := false
			if resultAfterClose != expectedResultAfterClose {
				t.Errorf("Expected result '%v', but got '%v'.", expectedResultAfterClose, resultAfterClose)
			}
			waitGroup.Done()
		}
	}()

	// closing context
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	waitGroup.Wait()
}

func TestIsTransactionOpenTrue(t *testing.T) {
	// Arrange
	contextMock := context.Background()
	uut, serviceMock := initTransactionScheduler(t)
	_, err := uut.CreateTransaction(contextMock, serviceMock)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Act
	result := uut.IsTransactionOpen()

	// Assert
	expectedResult := true
	if result != expectedResult {
		t.Fatalf("Expected result '%v', but got '%v'.", expectedResult, result)
	}
}

func TestGetTransaction(t *testing.T) {
	// Arrange
	contextMock := context.Background()
	uut, serviceMock := initTransactionScheduler(t)

	// Act
	tx, err := uut.CreateTransaction(contextMock, serviceMock)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tx == nil {
		t.Fatalf("Expected transaction but got none.")
	}

	resultTx := uut.GetTransaction()
	if tx != resultTx {
		t.Fatalf("Expected transaction '%v' but got '%v'", tx, resultTx)
	}
}
