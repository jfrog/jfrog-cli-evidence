package tests

import (
	"io"
	"os"
	"testing"

	corelog "github.com/jfrog/jfrog-cli-core/v2/utils/log"
	coreTests "github.com/jfrog/jfrog-cli-core/v2/utils/tests"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/stretchr/testify/assert"
)

// RunCliCmdWithStdOutputAndErrOutput runs a CLI command and returns the combined
// stdout and stderr content.
//
// Use this when assertions treat the command output as plain text (Contains /
// NotContains). Prefer (*JfrogCli).RunCliCmdWithOutput when the returned string
// must remain structured stdout only (for example JSON Unmarshal), because log
// lines written to stderr would pollute the parse.
func RunCliCmdWithStdOutputAndErrOutput(t *testing.T, cli *coreTests.JfrogCli, args ...string) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	assert.NoError(t, err)

	previousStdout := os.Stdout
	previousStderr := os.Stderr
	previousLog := log.Logger

	// Point both standard streams at the same pipe. NewLogger(nil) sends
	// OutputLog to os.Stdout and Info/Error logs to os.Stderr, so both ends
	// up in the combined capture.
	os.Stdout = writer
	os.Stderr = writer
	log.SetLogger(log.NewLogger(corelog.GetCliLogLevel(), nil))

	errCh := make(chan error, 1)
	go func() {
		errCh <- cli.Exec(args...)
		// Closing the writer unblocks the reader after the command finishes.
		assert.NoError(t, writer.Close())
	}()

	content, readErr := io.ReadAll(reader)
	cmdErr := <-errCh

	os.Stdout = previousStdout
	os.Stderr = previousStderr
	log.SetLogger(previousLog)
	assert.NoError(t, reader.Close())
	assert.NoError(t, readErr)

	output := string(content)
	log.Debug(output)
	assert.NoError(t, cmdErr)
	return output
}
