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

func (fb *flagBuilder) setUpFlags(cliParams interface{}) (err error) {
	cliV := reflect.ValueOf(cliParams)
	cliT := reflect.TypeOf(cliParams)
	cliV = cliV.Elem()
	cliT = cliT.Elem()

	numFields := cliV.NumField()
	for i := 0; i < numFields; i++ {
		fld := cliV.Field(i)
		fldT := cliT.Field(i)
		flagMetadata := fldT.Tag.Get("flag")

		if fld.Kind() == reflect.Struct {
			if err = fb.setUpFlags(fld.Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		if flagMetadata == "" {
			continue
		}

		switch tpe := fld.Interface().(type) {
		case string:
			fd, err := parseFlagData(fld, flagMetadata, func(s string) (string, error) { return s, nil })
			if err != nil {
				return err
			}
			fb.flagSet.StringVar(fd.addr, fd.name, fd.defaultVal, fd.usage)
			addRequired(fb, fd)

		case bool:
			fd, err := parseFlagData(fld, flagMetadata, strconv.ParseBool)
			if err != nil {
				return err
			}
			fb.flagSet.BoolVar(fd.addr, fd.name, fd.defaultVal, fd.usage)
			addRequired(fb, fd)

		case int:
			fd, err := parseFlagData(fld, flagMetadata, strconv.Atoi)
			if err != nil {
				return err
			}
			fb.flagSet.IntVar(fd.addr, fd.name, fd.defaultVal, fd.usage)
			addRequired(fb, fd)

		case int64:
			fd, err := parseFlagData(fld, flagMetadata, func(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 64)
			})
			if err != nil {
				return err
			}
			fb.flagSet.Int64Var(fd.addr, fd.name, fd.defaultVal, fd.usage)
			addRequired(fb, fd)

		case uint:
			fd, err := parseFlagData(fld, flagMetadata, func(s string) (uint, error) {
				result, err := strconv.ParseUint(s, 10, 32)
				return uint(result), err
			})
			if err != nil {
				return err
			}
			fb.flagSet.UintVar(fd.addr, fd.name, fd.defaultVal, fd.usage)
			addRequired(fb, fd)

		case uint64:
			fd, err := parseFlagData(fld, flagMetadata, func(s string) (uint64, error) {
				return strconv.ParseUint(s, 10, 64)
			})
			if err != nil {
				return err
			}
			fb.flagSet.Uint64Var(fd.addr, fd.name, fd.defaultVal, fd.usage)
			addRequired(fb, fd)

		case float64:
			fd, err := parseFlagData(fld, flagMetadata, func(s string) (float64, error) {
				return strconv.ParseFloat(s, 64)
			})
			if err != nil {
				return err
			}
			fb.flagSet.Float64Var(fd.addr, fd.name, fd.defaultVal, fd.usage)
			addRequired(fb, fd)

		case time.Duration:
			fd, err := parseFlagData(fld, flagMetadata, time.ParseDuration)
			if err != nil {
				return err
			}
			fb.flagSet.DurationVar(fd.addr, fd.name, fd.defaultVal, fd.usage)
			addRequired(fb, fd)

		default:
			return fmt.Errorf("unsupported flag type: %T", tpe)
		}
		if err != nil {
			return err
		}
	}
	if e, ok := cliParams.(Extender); ok {
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

type baseFlagData struct {
	name       string
	usage      string
	isRequired bool
}

type flagDataInstance[T any] struct {
	*baseFlagData
	addr       *T
	defaultVal T
}

func (fdi *baseFlagData) value() string {
	return fdi.name
}

func parseFlagData[T any](fld reflect.Value, flagMetadata string, tParser func(string) (T, error)) (*flagDataInstance[T], error) {
	baseData, flagDefault := parseBaseFlagData(flagMetadata)
	var defaultVal T
	if flagDefault != "" {
		var err error
		defaultVal, err = tParser(flagDefault)
		if err != nil {
			return nil, err
		}
	}
	if n := fmt.Sprintf("-%s", baseData.name); n == helpArg || n == helpArgShort {
		return nil, fmt.Errorf("overwriting of the reserved flag %s not allowed", n)
	}
	addr := fld.Addr().Interface().(*T)
	f := &flagDataInstance[T]{
		baseFlagData: baseData,
		addr:         addr,
		defaultVal:   defaultVal,
	}
	return f, nil
}

func parseBaseFlagData(flagMetadata string) (flagData *baseFlagData, defaultVal string) {
	metadataParts := strings.Split(flagMetadata, "|")
	name := strings.TrimSpace(metadataParts[0])
	var (
		usage      string
		isRequired bool
	)
	if len(metadataParts) > 1 {
		usage = strings.TrimSpace(metadataParts[1])
	}
	if len(metadataParts) > 2 {
		defaultVal = strings.TrimSpace(metadataParts[2])
	}
	if len(metadataParts) > 3 {
		// here is space for extending the flag checking
		if metadataParts[3] == requiredValue {
			defaultVal = "" // if it is required, we ignore default value
			isRequired = true
		}
	}
	return &baseFlagData{
		name:       name,
		usage:      usage,
		isRequired: isRequired,
	}, defaultVal
}

// this currently cannot be a flagBuilder method due to the type parameters usage
func addRequired[T any](fb *flagBuilder, fd *flagDataInstance[T]) {
	if fd.isRequired {
		fb.required[fd.name] = fd.addr
	}
}
