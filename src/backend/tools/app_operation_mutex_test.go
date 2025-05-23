//go:build fast

package tools

import (
	"github.com/ocelot-cloud/shared/assert"
	"testing"
	"time"
)

var (
	operation1 = "operation1"
	operation2 = "operation2"
)

func TestTryMutex_TryLockUnlock(t *testing.T) {
	tm := ProvideAppOperationMutex()

	assert.Nil(t, tm.TryLock(operation1))
	assert.Equal(t, operation1, currentOperation)
	err := tm.TryLock(operation2)
	assert.NotNil(t, err)
	assert.Equal(t, concurrentOperationErrorMessage, err.Error())

	tm.Unlock()
	assert.Equal(t, "", currentOperation)
	assert.Nil(t, tm.TryLock(operation2))

	tm.Unlock()
	tm.Unlock() // assert that it won't throw a fatal panic as it would when using a sync.Mutex directly
}

func TestTryMutex_LockAndWait(t *testing.T) {
	tm := ProvideAppOperationMutex()

	assert.Nil(t, tm.TryLock(operation1))
	assert.Equal(t, operation1, currentOperation)

	go tm.Lock(operation2)
	assert.Equal(t, operation1, currentOperation)
	tm.Unlock()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, operation2, currentOperation)
	tm.Unlock()
	assert.Equal(t, "", currentOperation)
}
