package util

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLevels(t *testing.T) {
	// create a buffer, a logger, and send the logs to the buffer
	var buf bytes.Buffer
	lg := NewLogger()
	lg.SetOutput(&buf)

	// The default log level is Info, so this will not be seen
	lg.Debugf("Test 1")

	// Update log level to Debug so the next msg is logged
	lg.SetLvl(DebugLevel)
	lg.Debugf("Test 2")

	// Raise level to Err (test is flaky without sleep here)
	lg.SetLvl(ErrorLevel)
	//time.Sleep(time.Second)

	// Info and Debug msgs should now be missed
	lg.Debug("Test 3")
	lg.Info("Test 4")

	// Error should be captured
	lg.Error("Test 5")

	// get output
	output := buf.String()

	// Results should contain test 2 and 5
	assert.Contains(t, output, "Test 2", "Test 2 should be in the output")
	assert.Contains(t, output, "Test 5", "Test 5 should be in the output")

	// Tests 1,3, and 4 should all not be there
	assert.NotContains(t, output, "Test 1", "Test 1 should not be in the output.")
	assert.NotContains(t, output, "Test 3", "Test 3 should not be in the output.")
	assert.NotContains(t, output, "Test 4", "Test 4 should not be in the output.")

}
