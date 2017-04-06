package provider

import (
	"context"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"
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

var pool = safe.NewPool(context.Background())

func createFile(t *testing.T, file string) *os.File {
	f, err := os.Create(file)
	if err != nil {
		t.Error(err)
		return nil
	}
	return f
}

func createTempDir(t *testing.T, dir string) string {
	d, err := ioutil.TempDir("", dir)
	if err != nil {
		t.Error(err)
		return ""
	}
	return d
}

func TestFileProvideSingleFile(t *testing.T) {
	tempDir := createTempDir(t, "test")
	tempFileName := path.Join(tempDir, "temp1.toml")

	tempFile := createFile(t, tempFileName)
	tempFile.WriteString(frontends)
	tempFile.WriteString(backends)
	tempFile.Close()

	fileProvider := new(File)
	c := make(chan types.ConfigMessage)

	var wg sync.WaitGroup
	wg.Add(1)

	numBackends := 2
	numFrontends := 3

	go func() {
		for {
			data := <-c
			assert.Equal(t, "file", data.ProviderName)
			assert.Equal(t, numBackends, len(data.Configuration.Backends))
			assert.Equal(t, numFrontends, len(data.Configuration.Frontends))
			wg.Done()
		}

	}()

	fileProvider.Filename = tempFile.Name()
	fileProvider.Watch = true
	fileProvider.Provide(c, pool, nil)

	wg.Wait()

	numBackends = 1
	numFrontends = 1

	wg.Add(1)

	tempFile = createFile(t, tempFileName)
	tempFile.WriteString(frontend1)
	tempFile.WriteString(backend1)

	wg.Wait()

	os.Remove(tempFile.Name())
	os.Remove(tempDir)
}

func TestFileProvideDirectory(t *testing.T) {
	tempDir := createTempDir(t, "test")

	tempFileName1 := path.Join(tempDir, "temp1.toml")
	tempFileName2 := path.Join(tempDir, "temp2.toml")

	tempFile1 := createFile(t, tempFileName1)
	tempFile2 := createFile(t, tempFileName2)
	tempFile1.WriteString(frontends)
	tempFile2.WriteString(backends)
	tempFile1.Close()
	tempFile2.Close()

	file := new(File)
	c := make(chan types.ConfigMessage)

	var wg sync.WaitGroup
	wg.Add(1)

	numBackends := 2
	numFrontends := 3

	go func() {
		for {
			data := <-c
			assert.Equal(t, "file", data.ProviderName)
			assert.Equal(t, numBackends, len(data.Configuration.Backends))
			assert.Equal(t, numFrontends, len(data.Configuration.Frontends))
			wg.Done()
		}
	}()

	file.Directory = tempDir
	file.Watch = true
	file.Provide(c, pool, nil)

	wg.Wait()

	numBackends = 0
	numFrontends = 0

	wg.Add(1)

	os.Remove(tempFile1.Name())
	os.Remove(tempFile2.Name())

	wg.Wait()

	numBackends = 2
	numFrontends = 3

	os.Remove(tempDir)
}
