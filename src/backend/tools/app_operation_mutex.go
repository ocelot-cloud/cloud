package tools

import (
	"errors"
	"sync"
)

const concurrentOperationErrorMessage = "another app operation is ongoing; try again later"

var AppOperationMutex = ProvideAppOperationMutex()

type OperationMutex struct {
	mu     sync.Mutex
	locked bool
}

func ProvideAppOperationMutex() *OperationMutex {
	return &OperationMutex{}
}

var currentOperation string

func (tm *OperationMutex) TryLock(operation string) error {
	Logger.Debug("Trying to lock app operation mutex for operation: %s", operation)
	success := tm.mu.TryLock()
	if !success {
		Logger.Info("Could not lock app operation mutex. Ongoing operation '%s' is blocking the attempted operation '%s'", currentOperation, operation)
		Logger.Info(concurrentOperationErrorMessage)
		return errors.New(concurrentOperationErrorMessage)
	}
	currentOperation = operation
	tm.locked = true
	return nil
}

func (tm *OperationMutex) Unlock() {
	Logger.Trace("Unlocking app operation mutex")
	if tm.locked {
		currentOperation = ""
		tm.locked = false
		tm.mu.Unlock()
	}
}

func (tm *OperationMutex) Lock(operation string) {
	Logger.Debug("Waiting to lock app operation mutex for operation: %s", operation)
	tm.mu.Lock()
	currentOperation = operation
	tm.locked = true
}
