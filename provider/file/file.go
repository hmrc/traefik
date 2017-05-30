package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	"gopkg.in/fsnotify.v1"
)

var _ provider.Provider = (*Provider)(nil)

// Provider holds configurations of the provider.
type Provider struct {
	provider.BaseProvider `mapstructure:",squash"`
	Directory             string `description:"Load configuration from one or more .toml files in a directory"`
}

// Provide allows the file provider to provide configurations to traefik
// using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- types.ConfigMessage, pool *safe.Pool, constraints types.Constraints) error {
	var configuration *types.Configuration
	var err error
	var watchDir string

	if p.Directory != "" {
		watchDir = p.Directory
		configuration, err = loadFileConfigFromDirectory(p.Directory)
	} else {
		watchDir = filepath.Dir(p.Filename)
		configuration, err = loadFileConfig(p.Filename)
	}

	if err != nil {
		return err
	}

	if err := p.addWatcher(pool, watchDir, configurationChan, p.watcherCallback); err != nil {
		return err
	}

	sendConfigToChannel(configurationChan, configuration)
	return nil
}

func (p *Provider) addWatcher(pool *safe.Pool, directory string, configurationChan chan<- types.ConfigMessage, callback func(chan<- types.ConfigMessage, fsnotify.Event)) error {
	// Debounce used to ensure that multiple watcher events in a short space of time don't trigger multiple config reloads
	var debouncePeriod = 1 * time.Second
	var event *fsnotify.Event

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating file watcher: %s", err)
	}

	// Process events
	pool.Go(func(stop chan bool) {
		defer watcher.Close()
		for {
			select {
			case <-stop:
				return
			case evt := <-watcher.Events:
				event = &evt
			case <-time.After(debouncePeriod):
				if event != nil {
					callback(configurationChan, *event)
					event = nil
				}
			case err := <-watcher.Errors:
				log.Errorf("Watcher event error: %s", err)
			}
		}
	})
	err = watcher.Add(directory)
	if err != nil {
		return fmt.Errorf("error adding file watcher: %s", err)
	}

	return nil
}

func sendConfigToChannel(configurationChan chan<- types.ConfigMessage, configuration *types.Configuration) {
	configurationChan <- types.ConfigMessage{
		ProviderName:  "file",
		Configuration: configuration,
	}
}

func loadFileConfig(filename string) (*types.Configuration, error) {
	configuration := new(types.Configuration)
	if _, err := toml.DecodeFile(filename, configuration); err != nil {
		return nil, fmt.Errorf("error reading configuration file: %s", err)
	}
	return configuration, nil
}

func loadFileConfigFromDirectory(directory string) (*types.Configuration, error) {
	var fileList []os.FileInfo
	var err error

	if fileList, err = ioutil.ReadDir(directory); err != nil {
		return nil, fmt.Errorf("unable to read directory %s: %v", directory, err)
	}

	configuration := &types.Configuration{Frontends: make(map[string]*types.Frontend),
		Backends: make(map[string]*types.Backend)}

	for _, file := range fileList {
		if !strings.HasSuffix(file.Name(), ".toml") {
			continue
		}

		var c *types.Configuration
		if c, err = loadFileConfig(path.Join(directory, file.Name())); err != nil {
			return nil, err
		}

		for k, v := range c.Backends {
			if _, exists := configuration.Backends[k]; exists {
				log.Warnf("Backend %s already configured, skipping", k)
			} else {
				configuration.Backends[k] = v
			}
		}

		for k, v := range c.Frontends {
			if _, exists := configuration.Frontends[k]; exists {
				log.Warnf("Frontend %s already configured, skipping", k)
			} else {
				configuration.Frontends[k] = v
			}

		}
	}

	return configuration, nil
}

func (p *Provider) watcherCallback(configurationChan chan<- types.ConfigMessage, event fsnotify.Event) {
	var configuration *types.Configuration
	var err error

	if p.Directory != "" {
		configuration, err = loadFileConfigFromDirectory(p.Directory)
	} else {
		configuration, err = loadFileConfig(p.Filename)
	}

	if err != nil {
		log.Errorf("Error occurred during watcher callback: %s", err)
		return
	}

	sendConfigToChannel(configurationChan, configuration)
}
