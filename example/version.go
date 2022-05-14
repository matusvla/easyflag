package example

import (
	"fmt"
	"os"
)

// version variable set by LDFLAGS, see Makefile:

var BuildVersion string

type BuildVersionFlag struct {
	HasVersionPrintout bool `flag:"v|prints out version (in semver aka v1.4.78 format)"`
}

func (bvf *BuildVersionFlag) Extend() error {
	// special handling of -v flag
	if bvf.HasVersionPrintout {
		fmt.Printf("%v\n", BuildVersion)
		os.Exit(0)
	}
	return nil
}
