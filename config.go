package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/Fornaxian/log"
)

// Manager is in charge of finding and reading configuration files
type Manager struct {
	confPaths     []string
	fileName      string
	defaultConfig string
	Conf          interface{}
}

// ErrNoConfigFound returned by ReloadConfig if no config file cound be found
type ErrNoConfigFound struct{}

func (ErrNoConfigFound) Error() string {
	return "no config files found at the configured locations"
}

// New prepares a new configuration manager which can be used to read properties
// from a TOML config file. The confDir param can be used to set a custom config
// directory. If it's left empty the default config locations will be used. When
// no config files can be found on the system a new one will be generated in the
// current working directory, and the program will exit. If it fails to write
// the config file it will print instructions for the user to create a new
// config file and exit with an error status.
//
// Params:
// - defaultConfig: The default configuration file in TOML format. If a config
//                  file is found at any of the configured locations, but it's
//                  missing some tags the default values from this config will
//                  be used. Note that if no config files are found, all the
//                  properties will be the defaults.
// - confDir:       A directory which will be searched for a configuration file.
//                  If empty the default system directories will be searched for
//                  configuration files.
// - fileName:      The name of the configuration file, only files with this
//                  name will be attempted to be parsed.
// - conf:          This has to be a pointer to a struct with TOML annotations.
//                  https://github.com/BurntSushi/toml/blob/master/README.md#examples
// - autoload:      Setting this to true will automatically load the config file
//                  before returning this function. If it fails to load the
//                  config file it will print instructions for the user to
//                  stdout and exit the program. If you wish to handle this
//                  behaviour yourself you should set autoload to false and call
//                  Manager.LoadConfig manually.
func New(
	defaultConf, confDir, fileName string,
	config interface{},
	autoload bool,
) (*Manager, error) {
	var err error
	var c = &Manager{
		confPaths: []string{
			confDir + "/" + fileName,
			fileName, // In the working directory
			fmt.Sprintf("%s/.config/%s", os.Getenv("HOME"), fileName),
			"/usr/local/etc/" + fileName,
			"/etc/" + fileName,
		},
		fileName:      fileName,
		defaultConfig: defaultConf,
		Conf:          config,
	}

	// Read the default configuration. The values entered in the config file
	// will overwrite the defaults
	_, err = toml.Decode(defaultConf, c.Conf)
	if err != nil {
		return nil, fmt.Errorf("failed to decode default config: %s", err)
	}

	if !autoload {
		return c, nil
	}

	err = c.LoadConfig()
	if _, ok := err.(ErrNoConfigFound); ok {
		log.Info("No configuration files were found, a new one will be " +
			"generated in the present working directory")
		err = ioutil.WriteFile(fileName, []byte(defaultConf), 0644)
		if err != nil {
			log.Warn(
				"A default config file could not be created in this directory "+
					"for the following reason: %s.\n\n"+
					"Please manually create a configuration file in one "+
					"of the following places:\n", err, fileName,
			)
			for _, cd := range c.confPaths {
				fmt.Println(cd)
			}
			os.Exit(1)
		}
		os.Exit(0)
	}

	log.Info("Successfully loaded configuration file")
	return c, nil
}

// LoadConfig tries every configuration file configured in the Manager until it
// finds one it can read. If no configuration files can be read it will return
// an ErrNoConfigFound error. If error is nil the config was loaded
// successfully. This function can be called multiple times to reload the config
// file from disk.
func (c *Manager) LoadConfig() error {
	var confStr []byte
	var err error

	for _, cd := range c.confPaths {
		if cd == "" {
			continue
		}

		log.Debug("Trying configuration file '%s'", cd)
		confStr, err = ioutil.ReadFile(cd)
		if err != nil {
			log.Debug("No config found at '%s' (%s)", cd, err)
			continue
		}
		// Reading succeeded, now try decoding

		_, err = toml.Decode(string(confStr), c.Conf)
		if err != nil {
			log.Warn("Unable to decode config file at '%s': %s", cd, err)
			continue
		}

		// We did it
		return nil
	}

	return ErrNoConfigFound{}
}
