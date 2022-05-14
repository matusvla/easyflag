package cli

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

// Extender is an interface that can be implemented by the type passed to the ParseAndLoadFlags function.
// It can be used for the additional validation or modification of the CLI parameters
type Extender interface {
	Extend() error
}

/*
ParseAndLoadFlags takes a pointer to a structure and fills it from the CLI flags according to the `flag` meta tags
defined on the level of structure's fields.

Example of the input structure:
	type Params struct {
		Str       string        `flag:"str|Testing string||required"`
		Str2      string        `flag:"str2|Testing string2|Str2 default|"`
		Boo       bool          `flag:"boo|Testing boolean|true|"`
		Number    int           `flag:"num|Testing number|123|"`
		ExtNumber int           `flag:"extnum|Extension testing number|"`
		Number64  int64         `flag:"num64|Testing number|1234|"`
		UNumber   uint          `flag:"unum|Testing number||required"`
		UNumber64 uint64        `flag:"unum64|Testing number|123456|"`
		Float64   float64       `flag:"fnum64|Testing number|123.456|"`
		Dur       time.Duration `flag:"dur|Testing number|10m|"`
	}

The value of the `flag` metadata consists of four parts separated by the '|' character. Only the first value is mandatory
The first value is the name of the matching CLI flag. Bool flags can be passed without an explicit value which sets the underlying field to true.
The second value is the flag's description.
The third value is the default value of this flag.
The fourth value is used to specify that a flag is required. If this is specified, the default value is ignored.

There are two default flags -h and -help. If a user provides one of these, the program only prints the information about
the available flags and finishes.

If the Params type or any of its fields implements the Extender interface then its Extend method will be called at the end of the setup.
This can be used for the validation or modification of the field values.

In case of an error during the flag parsing, the passed structure is set to its zero value and the error is returned
*/
func ParseAndLoadFlags(params interface{}) (retErr error) {
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
	err := fb.parseFlags(passedArgs)
	if err != nil {
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

// InvalidParamsError is an error returned in case that the provided CLI Params is not a pointer to a structure.
type InvalidParamsError struct {
	Type reflect.Type
}

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
