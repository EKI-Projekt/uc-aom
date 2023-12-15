// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"sync"
)

type TransactionScheduler struct {
	// Protect access to the one and only transaction, tx.
	mu sync.RWMutex
	tx *Tx
}

// Returns a new TransactionScheduler
func NewTransactionScheduler() *TransactionScheduler {
	return &TransactionScheduler{
		mu: sync.RWMutex{},
		tx: nil,
	}
}

// Creates a new transaction
func (s *TransactionScheduler) CreateTransaction(ctx context.Context, service *Service) (*Tx, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.IsTransactionOpen() {
		return nil, errors.New("TransactionScheduler: Transaction already open")
	}

	s.tx = BeginTx(ctx, service)
	return s.tx, nil
}

// Return true if transaction is open/in progress
func (s *TransactionScheduler) IsTransactionOpen() bool {
	return s.tx != nil && !s.tx.IsDone()
}

// Return the transaction of the TransactionScheduler
func (s *TransactionScheduler) GetTransaction() *Tx {
	return s.tx
}
