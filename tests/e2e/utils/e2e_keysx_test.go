package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testStableFor = 30 * time.Millisecond
	testInterval  = 10 * time.Millisecond
	testTimeout   = 2 * time.Second
)

func TestWaitUntilStable_ReturnsOnceStable(t *testing.T) {
	calls := 0
	err := waitUntilStable(func() (bool, error) {
		calls++
		return true, nil
	}, testStableFor, testInterval, testTimeout)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, calls, 2, "a single sighting must not be enough")
}

func TestWaitUntilStable_MissRestartsWait(t *testing.T) {
	calls := 0
	err := waitUntilStable(func() (bool, error) {
		calls++
		// Found on the first two calls, missing on the third, found afterwards.
		return calls != 3, nil
	}, testStableFor, testInterval, testTimeout)

	assert.NoError(t, err)
	// Without the restart, the wait would end on the fourth call, 30ms after the first sighting.
	assert.GreaterOrEqual(t, calls, 5)
}

func TestWaitUntilStable_ErrorRestartsWait(t *testing.T) {
	calls := 0
	err := waitUntilStable(func() (bool, error) {
		calls++
		if calls == 3 {
			return false, errors.New("list failed")
		}
		return true, nil
	}, testStableFor, testInterval, testTimeout)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, calls, 5)
}

func TestWaitUntilStable_TimesOutWithLastError(t *testing.T) {
	listErr := errors.New("list failed")
	err := waitUntilStable(func() (bool, error) {
		return false, listErr
	}, testStableFor, testInterval, 50*time.Millisecond)

	assert.ErrorIs(t, err, listErr)
	assert.ErrorContains(t, err, "timed out after 50ms")
}

func TestWaitUntilStable_TimesOutWhenNeverFound(t *testing.T) {
	err := waitUntilStable(func() (bool, error) {
		return false, nil
	}, testStableFor, testInterval, 50*time.Millisecond)

	assert.EqualError(t, err, "timed out after 50ms")
}
