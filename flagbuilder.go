package easyflag

import (
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type flagBuilder struct {
	flagSet  *flag.FlagSet
	required map[string]interface{} // map[flag name]pointers to the required fields to be able to check if they have been filled after the initialization
	extFns   []func() error
}

func newFlagBuilder() *flagBuilder {
	return &flagBuilder{
		required: make(map[string]interface{}),
		flagSet:  flag.NewFlagSet("", flag.ContinueOnError),
	}
}

func (fb *flagBuilder) setUpFlags(params interface{}) error {
	cliV := reflect.ValueOf(params).Elem()
	cliT := reflect.TypeOf(params).Elem()

	for i := 0; i < cliV.NumField(); i++ {
		fld := cliV.Field(i)
		fldT := cliT.Field(i)
		flagMetadataStr := fldT.Tag.Get("flag")

		// recursion for the underlying structures
		if fld.Kind() == reflect.Struct {
			if err := fb.setUpFlags(fld.Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		// skipping the fields without the `flag` field tag
		if flagMetadataStr == "" {
			continue
		}

		var err error
		switch tpe := fld.Interface().(type) {
		case string:
			err = parseAndAttachFlagData(fb, fld, flagMetadataStr, func(s string) (string, error) { return s, nil }, fb.flagSet.StringVar)

		case bool:
			err = parseAndAttachFlagData(fb, fld, flagMetadataStr, strconv.ParseBool, fb.flagSet.BoolVar)

		case int:
			err = parseAndAttachFlagData(fb, fld, flagMetadataStr, strconv.Atoi, fb.flagSet.IntVar)

		case int64:
			err = parseAndAttachFlagData(fb, fld, flagMetadataStr, func(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 64)
			}, fb.flagSet.Int64Var)

		case uint:
			err = parseAndAttachFlagData(fb, fld, flagMetadataStr, func(s string) (uint, error) {
				result, err := strconv.ParseUint(s, 10, 32)
				return uint(result), err
			}, fb.flagSet.UintVar)

		case uint64:
			err = parseAndAttachFlagData(fb, fld, flagMetadataStr, func(s string) (uint64, error) {
				return strconv.ParseUint(s, 10, 64)
			}, fb.flagSet.Uint64Var)

		case float64:
			err = parseAndAttachFlagData(fb, fld, flagMetadataStr, func(s string) (float64, error) {
				return strconv.ParseFloat(s, 64)
			}, fb.flagSet.Float64Var)

		case time.Duration:
			err = parseAndAttachFlagData(fb, fld, flagMetadataStr, time.ParseDuration, fb.flagSet.DurationVar)

		default:
			return fmt.Errorf("unsupported flag type: %T", tpe)
		}
		if err != nil {
			return err
		}
	}
	if e, ok := params.(Extender); ok {
		fb.extFns = append(fb.extFns, e.Extend)
	}
	return nil
}

func (fb *flagBuilder) parseFlags(args []string) error {
	return fb.flagSet.Parse(args)
}

func (fb *flagBuilder) validate() error {
	var missing []string
	for key, val := range fb.required {
		fld := reflect.ValueOf(val).Elem()
		if fld.IsZero() {
			missing = append(missing, key)
		}
	}
	switch len(missing) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("missing required flag %q or its value", strings.Join(missing, ", "))
	default:
		return fmt.Errorf("missing required flags %q or their values", strings.Join(missing, ", "))
	}
}

// runExtensionFunctions recursively runs all the relevant extension functions found during the flag collection process
func (fb *flagBuilder) runExtensionFunctions() error {
	for _, extFn := range fb.extFns {
		if err := extFn(); err != nil {
			return fmt.Errorf("extension running failed: %w", err)
		}
	}
	return nil
}

func parseAndAttachFlagData[T any](
	fb *flagBuilder,
	fld reflect.Value,
	flagMetadata string,
	parseFn func(string) (T, error),
	attachFn func(p *T, name string, value T, usage string),
) error {
	fm, err := parseFlagMetadata(flagMetadata)
	if err != nil {
		return err
	}
	var defaultVal T
	if fm.defaultVal != "" {
		var err error
		defaultVal, err = parseFn(fm.defaultVal)
		if err != nil {
			return err
		}
	}
	if n := fmt.Sprintf("-%s", fm.name); n == helpArg || n == helpArgShort {
		return fmt.Errorf("reserved flag %s overwriting not allowed", n)
	}
	addr := fld.Addr().Interface().(*T)

	attachFn(addr, fm.name, defaultVal, fm.usage)
	if fm.isRequired {
		fb.required[fm.name] = addr
	}
	return nil
}

type flagMetadata struct {
	name       string
	usage      string
	defaultVal string
	isRequired bool
}

func parseFlagMetadata(flagMetadataStr string) (flagMetadata, error) {
	metadataParts := strings.Split(flagMetadataStr, "|")
	name := strings.TrimSpace(metadataParts[0])
	var (
		usage, defaultVal string
		isRequired        bool
	)
	if len(metadataParts) > 1 {
		usage = strings.TrimSpace(metadataParts[1])
	}
	if len(metadataParts) > 2 {
		defaultVal = strings.TrimSpace(metadataParts[2])
	}
	if len(metadataParts) > 3 {
		switch val := metadataParts[3]; val {
		case requiredValue:
			defaultVal = "" // if it is required, we ignore default value
			isRequired = true
		case "":
		default:
			return flagMetadata{}, fmt.Errorf("unsupported value %q in the fourth metadata part", val)
		}
	}
	return flagMetadata{name, usage, defaultVal, isRequired}, nil
}
