package file

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	"github.com/stretchr/testify/assert"
)

const (
	frontends = `
[frontends]
  [frontends.frontend1]
  backend = "backend2"
    [frontends.frontend1.routes.test_1]
    rule = "Host:test.localhost"
  [frontends.frontend2]
  backend = "backend1"
  passHostHeader = true
  entrypoints = ["https"] # overrides defaultEntryPoints
    [frontends.frontend2.routes.test_1]
    rule = "Host:{subdomain:[a-z]+}.localhost"
  [frontends.frontend3]
  entrypoints = ["http", "https"] # overrides defaultEntryPoints
  backend = "backend2"
    rule = "Path:/test"
    `

	backends = `
[backends]
  [backends.backend1]
    [backends.backend1.circuitbreaker]
      expression = "NetworkErrorRatio() > 0.5"
    [backends.backend1.servers.server1]
    url = "http://172.17.0.2:80"
    weight = 10
    [backends.backend1.servers.server2]
    url = "http://172.17.0.3:80"
    weight = 1
  [backends.backend2]
    [backends.backend2.LoadBalancer]
      method = "drr"
    [backends.backend2.servers.server1]
    url = "http://172.17.0.4:80"
    weight = 1
    [backends.backend2.servers.server2]
    url = "http://172.17.0.5:80"
    weight = 2
    `

	frontend1 = `
[frontends]
  [frontends.frontend1]
  backend = "backend1"
    [frontends.frontend1.routes.test_1]
    rule = "Host:test.localhost"
    `

	backend1 = `
  [backends.backend1]
    [backends.backend1.circuitbreaker]
      expression = "NetworkErrorRatio() > 0.5"
    [backends.backend1.servers.server1]
    url = "http://172.17.0.2:80"
    weight = 10
    [backends.backend1.servers.server2]
    url = "http://172.17.0.3:80"
    weight = 1
    `
)

func TestProvideSingleFile(t *testing.T) {
	tempDir := createTempDir(t, "testsingle")
	defer os.RemoveAll(tempDir)

	tempFileName := path.Join(tempDir, "temp1.toml")
	tempFile := createFile(t, tempFileName)
	tempFile.WriteString(frontends + backends)
	tempFile.Close()

	provider := new(Provider)
	c := make(chan types.ConfigMessage)

	var wg sync.WaitGroup
	wg.Add(1)

	numFrontends := 3
	numBackends := 2

	go func() {
		for {
			data := <-c
			assert.Equal(t, "file", data.ProviderName)
			assert.Equal(t, numFrontends, len(data.Configuration.Frontends))
			assert.Equal(t, numBackends, len(data.Configuration.Backends))
			wg.Done()
		}

	}()

	provider.Filename = tempFile.Name()
	provider.Watch = true
	provider.Provide(c, safe.NewPool(context.Background()), nil)

	// Wait for initial message to be tested
	wg.Wait()

	// Now test again with single frontend and backend
	numFrontends = 1
	numBackends = 1
	wg.Add(1)
	tempFile = createFile(t, tempFileName)
	tempFile.WriteString(frontend1)
	tempFile.WriteString(backend1)
	tempFile.Close()
	wg.Wait()
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

	provider := new(Provider)
	c := make(chan types.ConfigMessage)

	var wg sync.WaitGroup
	wg.Add(1)

	numFrontends := 3
	numBackends := 2

	go func() {
		for {
			data := <-c
			assert.Equal(t, "file", data.ProviderName)
			assert.Equal(t, numFrontends, len(data.Configuration.Frontends))
			assert.Equal(t, numBackends, len(data.Configuration.Backends))
			wg.Done()

		}
	}()

	provider.Directory = tempDir
	provider.Watch = true
	provider.Provide(c, safe.NewPool(context.Background()), nil)

	// Wait for initial config message to be tested
	wg.Wait()

	// Now remove the backends file
	numFrontends = 3
	numBackends = 0
	wg.Add(1)
	os.Remove(tempFile2.Name())
	wg.Wait()

	// Now remove the frontends file
	numFrontends = 0
	numBackends = 0
	wg.Add(1)
	os.Remove(tempFile1.Name())
	wg.Wait()

}

func createFile(t *testing.T, file string) *os.File {
	f, err := os.Create(file)
	if err != nil {
		t.Error(err)
	}
	return f
}

func createTempDir(t *testing.T, dir string) string {
	d, err := ioutil.TempDir("", dir)
	if err != nil {
		t.Error(err)
	}
	return d
}
