// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// ErrTxDone is returned by any operation that is performed on a transaction
// that has already been committed or rolled back.
var ErrTxDone = errors.New("uc-aom: transaction has already been committed or rolled back")

// The operation that is being performed by the open transation.
type Operation int

const (
	Unspecified Operation = iota
	Installing  Operation = iota
	Deleting    Operation = iota
	Updating    Operation = iota
	Configuring Operation = iota
)

type AffectedAddOn struct {
	Name      string
	Title     string
	Operation Operation
}

// Tx is an in-progress aom transaction.
//
// A transaction must end with a call to Commit or Rollback.
//
// After a call to Commit or Rollback, all operations on the
// transaction fail with ErrTxDone.
//
// The statements prepared for a transaction by calling
// the transaction's Prepare or Stmt methods are closed
// by the call to Commit or Rollback.
type Tx struct {
	service *Service

	// done transitions from 0 to 1 exactly once, on Commit
	// or Rollback. once done, all operations fail with
	// ErrTxDone.
	// Use atomic operations on value when checking value.
	done int32

	// cancel is called after done transitions from 0 to 1.
	cancel func()

	// ctx lives for the life of the transaction.
	ctx context.Context

	// used to roll back any failed actions.
	rollbackHooks []func()

	// the AddOn affected by this transaction.
	// it is a lazy immutable property, which once set cannot be overwritten.
	mu       sync.RWMutex
	affected *AffectedAddOn
}

func BeginTx(ctx context.Context, service *Service) *Tx {
	ctx, cancel := context.WithCancel(ctx)

	tx := &Tx{
		service:       service,
		cancel:        cancel,
		ctx:           ctx,
		rollbackHooks: make([]func(), 0),
	}

	go tx.awaitDone()
	return tx
}

// Done returns a channel that's closed when work done
func (tx *Tx) Done() <-chan struct{} {
	return tx.ctx.Done()
}

// Returns the AddOn identified by name if it is involved
// in this transaction, otherwise nil.
func (tx *Tx) AffectedAddOn(name string) *AffectedAddOn {
	if tx.IsDone() {
		return nil
	}

	tx.mu.RLock()
	defer tx.mu.RUnlock()

	if tx.affected != nil && tx.affected.Name == name {
		return tx.affected
	}

	return nil
}

// Subscribe a rollback hook for this transition.
func (tx *Tx) SubscribeRollbackHook(rollback func()) {
	tx.mu.Lock()
	tx.rollbackHooks = append(tx.rollbackHooks, rollback)
	tx.mu.Unlock()
}

// awaitDone blocks until the context in Tx is canceled and rolls back
// the transaction if it's not already done.
func (tx *Tx) awaitDone() {
	// Wait for either the transaction to be committed or rolled
	// back, or for the associated context to be closed.
	<-tx.ctx.Done()

	// Discard and close the connection used to ensure the
	// transaction is closed and the resources are released.  This
	// rollback does nothing if the transaction has already been
	// committed or rolled back.
	tx.rollback()
}

// Set the addon which in the context of this transition.
func (tx *Tx) setAddOnContext(name string, title string, op Operation) {
	if tx.IsDone() {
		return
	}

	tx.mu.Lock()
	if tx.affected == nil {
		tx.affected = &AffectedAddOn{Operation: op, Name: name, Title: title}
	}
	tx.mu.Unlock()
}

// Returns whether this transaction has been committed or rolled back.
func (tx *Tx) IsDone() bool {
	return atomic.LoadInt32(&tx.done) != 0
}

// close must only be called by Tx.rollback or Tx.Commit while
// tx is already canceled and won't be executed concurrently.
func (tx *Tx) close() {
	tx.mu.Lock()
	tx.affected = nil
	tx.rollbackHooks = nil
	tx.mu.Unlock()
}

// Commit commits the transaction.
func (tx *Tx) Commit() error {
	// Check context first to avoid transaction leak.
	// If put it behind tx.done CompareAndSwap statement, we can't ensure
	// the consistency between tx.done and the real COMMIT operation.
	select {
	default:
	case <-tx.ctx.Done():
		if atomic.LoadInt32(&tx.done) == 1 {
			return ErrTxDone
		}
		return tx.ctx.Err()
	}
	if !atomic.CompareAndSwapInt32(&tx.done, 0, 1) {
		return ErrTxDone
	}

	tx.cancel()
	tx.close()
	return nil
}

// Rollback aborts the transaction.
func (tx *Tx) Rollback() error {
	return tx.rollback()
}

// rollback aborts the transaction.
func (tx *Tx) rollback() error {
	if !atomic.CompareAndSwapInt32(&tx.done, 0, 1) {
		return ErrTxDone
	}

	// Apply rollbacks in reverse order.
	tx.mu.RLock()
	for i := len(tx.rollbackHooks) - 1; i >= 0; i-- {
		tx.rollbackHooks[i]()
	}
	tx.mu.RUnlock()

	tx.cancel()
	tx.close()
	return nil
}
