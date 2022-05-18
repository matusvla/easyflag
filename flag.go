package easyflag

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
)

const (
	helpArg      = "-help"
	helpArgShort = "-h"

	requiredValue = "required"
)

// Extender is an interface that can be implemented by the type passed to the ParseAndLoad function.
// It can be used for additional validation or modification of the CLI arguments
type Extender interface {
	Extend() error
}

/*
ParseAndLoad takes a pointer to a structure and fills it from the user defined CLI flags according to the `flag` fields metadata.

If the params type or any of its fields implements the Extender interface then its Extend method will be called at the end of the setup.
This can be used for the validation or modification of the field values.

In case of an error during the flag parsing, the passed structure is set to its zero value and the error is returned.
*/
func ParseAndLoad(params interface{}) (retErr error) {
	rv := reflect.ValueOf(params)
	if rv.Kind() != reflect.Ptr || rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
		return &InvalidParamsError{reflect.TypeOf(params)}
	}

	defer func() {
		if retErr != nil {
			pEl := rv.Elem()
			pEl.Set(reflect.Zero(pEl.Type()))
		}
	}()

	fb := newFlagBuilder()
	if err := fb.setUpFlags(params); err != nil {
		return err
	}

	passedArgs := os.Args[1:] // first argument is a command name - we skip it
	if err := fb.parseFlags(passedArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		return err
	}

	if err := fb.runExtensionFunctions(); err != nil {
		return err
	}

	return fb.validate()
}

// InvalidParamsError is an error returned in case that the `params` argument passed to the ParseAndLoad function is not a pointer to a structure.
type InvalidParamsError struct {
	Type reflect.Type
}

// Error prints the description of the InvalidParamsError.
func (e *InvalidParamsError) Error() string {
	const outputFmt = "flags parse: got %s"
	if e.Type == nil {
		return fmt.Sprintf(outputFmt, "<nil>")
	}

	if e.Type.Kind() != reflect.Ptr {
		return fmt.Sprintf(outputFmt, "non-pointer "+e.Type.String())
	}
	return fmt.Sprintf(outputFmt, e.Type.String())
}
