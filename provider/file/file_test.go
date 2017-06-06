package file

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	"github.com/stretchr/testify/assert"
)

const (
	frontends = `
[frontends]
  [frontends.frontend1]
  backend = "backend1"
  [frontends.frontend2]
  backend = "backend2"
    `

	backends = `
[backends]
  [backends.backend1]
    [backends.backend1.servers.server1]
    url = "http://172.17.0.2:80"
  [backends.backend2]
    [backends.backend2.servers.server1]
    url = "http://172.17.0.4:80"
    `

	frontend1 = `
[frontends]
  [frontends.frontend1]
  backend = "backend1"
    `

	backend1 = `
  [backends.backend1]
    [backends.backend1.servers.server1]
    url = "http://172.17.0.2:80"
    `
)

func TestProvideSingleFile(t *testing.T) {
	tempDir := createTempDir(t, "testsingle")
	defer os.RemoveAll(tempDir)

	tempFileName := path.Join(tempDir, "temp1.toml")
	tempFile := createFile(t, tempFileName)
	tempFile.WriteString(frontends + backends)
	tempFile.Close()

	c := make(chan types.ConfigMessage)
	signal := make(chan interface{})

	numFrontends := 2
	numBackends := 2

	go func() {
		for {
			data := <-c
			assert.Equal(t, "file", data.ProviderName)
			assert.Equal(t, numFrontends, len(data.Configuration.Frontends))
			assert.Equal(t, numBackends, len(data.Configuration.Backends))
			signal <- nil
		}

	}()

	provider := &Provider{
		BaseProvider: provider.BaseProvider{
			Filename: tempFile.Name(),
			Watch:    true,
		},
	}
	provider.Provide(c, safe.NewPool(context.Background()), nil)

	// Wait for initial message to be tested
	waitForSignal(t, signal, 2*time.Second)

	// Now test again with single frontend and backend
	numFrontends = 1
	numBackends = 1

	tempFile = createFile(t, tempFileName)
	tempFile.WriteString(frontend1)
	tempFile.WriteString(backend1)
	tempFile.Close()

	waitForSignal(t, signal, 2*time.Second)
}

func TestProvideDirectory(t *testing.T) {
	tempDir := createTempDir(t, "testdir")
	defer os.RemoveAll(tempDir)

	tempFileName1 := path.Join(tempDir, "temp1.toml")
	tempFileName2 := path.Join(tempDir, "temp2.toml")

	tempFile1 := createFile(t, tempFileName1)
	tempFile2 := createFile(t, tempFileName2)
	tempFile1.WriteString(frontends)
	tempFile2.WriteString(backends)
	tempFile1.Close()
	tempFile2.Close()

	c := make(chan types.ConfigMessage)
	signal := make(chan interface{})

	numFrontends := 2
	numBackends := 2

	go func() {
		for {
			data := <-c
			assert.Equal(t, "file", data.ProviderName)
			assert.Equal(t, numFrontends, len(data.Configuration.Frontends))
			assert.Equal(t, numBackends, len(data.Configuration.Backends))
			signal <- nil
		}
	}()

	provider := &Provider{
		BaseProvider: provider.BaseProvider{
			Watch: true,
		},
		Directory: tempDir,
	}
	provider.Provide(c, safe.NewPool(context.Background()), nil)

	// Wait for initial config message to be tested
	waitForSignal(t, signal, 2*time.Second)

	// Now remove the backends file
	numFrontends = 2
	numBackends = 0

	os.Remove(tempFile2.Name())
	waitForSignal(t, signal, 2*time.Second)

	// Now remove the frontends file
	numFrontends = 0
	numBackends = 0
	os.Remove(tempFile1.Name())
	waitForSignal(t, signal, 2*time.Second)

}

func createFile(t *testing.T, file string) *os.File {
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func createTempDir(t *testing.T, dir string) string {
	d, err := ioutil.TempDir("", dir)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func waitForSignal(t *testing.T, signal chan interface{}, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-signal:

	case <-timer.C:
		t.Fatal("Timed out waiting for assertions to be tested")
	}
}
