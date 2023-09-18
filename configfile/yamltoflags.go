package configfile

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

func ParseConfigFileToFlagset(fn string, flags *flag.FlagSet) error {
	yamlData, err := os.ReadFile(fn)
	// Nonexistent config file is OK.
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		log.Fatalf("error reading config file %q: %v", fn, err)
	}

	parsed := make(map[interface{}]interface{})
	err = yaml.Unmarshal(yamlData, &parsed)
	if err != nil {
		log.Fatalf("could not parse config file %q: %v", fn, err)
	}

	flags.VisitAll(func(flag *flag.Flag) {
		if value, ok := parsed[flag.Name]; ok {
			flags.Set(flag.Name, fmt.Sprintf("%v", value))
		}
	})

	return nil
}

func ParseConfigFile(fn string) {
	ParseConfigFileToFlagset(fn, flag.CommandLine)
}
