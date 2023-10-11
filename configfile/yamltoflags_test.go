package configfile

import (
	"flag"
	"testing"
)

func TestParseYaml(t *testing.T) {
	flagSet := flag.NewFlagSet("testing", flag.PanicOnError)

	flag1 := flagSet.String("flag1", "", "boem should go here")
	flag2 := flagSet.String("flag2", "", "babies should go here")
	flagInt := flagSet.Int("flagInt", -1, "33 should go here")
	ParseConfigFileToFlagset("testdata/config.yaml", flagSet)

	if *flag1 != "boem" {
		t.Errorf("expected flag1 to contain boem, it has %q", *flag1)
	}
	if *flag2 != "babies" {
		t.Errorf("expected flag2 to contain babies, it has %q", *flag2)
	}
	if *flagInt != 33 {
		t.Errorf("expected flagInt to contain 33, it has %d", *flagInt)
	}
}
