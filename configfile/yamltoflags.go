package configfile

import (
	"os"
	"flag"
	"log"
	"fmt"

	"gopkg.in/yaml.v3"
)

func ParseConfigFileToFlagset(fn string, flags *flag.FlagSet) error {
	yamlData, err := os.ReadFile(fn)
	// Nonexistent config file is OK.
	if err == os.ErrNotExist {
		log.Printf("config file doesn't exist => skipping")
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