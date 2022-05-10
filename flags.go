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

	mandatoryValueIdent = "mandatory"
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
		Str       string        `flag:"str=|Testing string||mandatory"`
		Str2      string        `flag:"str2=|Testing string2|Str2 default|"`
		Boo       bool          `flag:"boo|Testing boolean|true|"`
		Number    int           `flag:"num=|Testing number|123|"`
		ExtNumber int           `flag:"extnum=|Extension testing number|"`
		Number64  int64         `flag:"num64=|Testing number|1234|"`
		UNumber   uint          `flag:"unum=|Testing number|12345|mandatory"`
		UNumber64 uint64        `flag:"unum64=|Testing number|123456|"`
		Float64   float64       `flag:"fnum64=|Testing number|123.456|"`
		Dur       time.Duration `flag:"dur=|Testing number|10m|"`
	}

The value of the `flag` metadata consists of four parts separated by the '|' character. Only the first value is mandatory
The first value is the name of the matching CLI flag. Use `name=` to denote arguments with value (e.g. `date=` would expect CLI argument `./a_program date=2020-11-07` whereas
`b` counts on argument being simply `./a_program -b`).
The second value is the flag's description.
The third value is the default value of this flag.
The fourth value is used to specify that a flag is mandatory. If this is specified, the default value is ignored.

There are two default flags -h and -help. If a user provides one of these, the program only prints the information about
the available flags and finishes.

If the Params type ar any of the fields that it consists of fulfills the Extender interface then its Extend method will be called at the end of the setup.
In case there is an error, the state of the passed structure is set to its zero value.
*/
func ParseAndLoadFlags(params interface{}) (retErr error) {
	rv := reflect.ValueOf(params)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &InvalidParseError{reflect.TypeOf(params)}
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
	isHelpRequest, err := fb.parseFlags(os.Args[1:]) // first argument is a command name - we skip it
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		return err
	}
	if isHelpRequest {
		os.Exit(0)
	}

	if err := fb.runExtensionFunctions(); err != nil {
		return err
	}

	return fb.validate()
}

// runExtensionFunctions recursively runs all the relevant extension functions found during the flag collection process
func (fb *flagBuilder) runExtensionFunctions() error {
	for _, extFn := range fb.extFns {
		if err := extFn(); err != nil {
			return fmt.Errorf("running flag extensions failed, %w", err)
		}
	}
	return nil
}

// InvalidParseError is an error returned in case that the provided CLI Params structure is of an unsupported type
type InvalidParseError struct {
	Type reflect.Type
}

func (e *InvalidParseError) Error() string {
	if e.Type == nil {
		return "flags parse: got nil"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "flags parse: got (non-pointer " + e.Type.String() + ")"
	}
	return "flags parse: got (nil " + e.Type.String() + ")"
}
