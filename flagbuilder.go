package cli

import (
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type flagBuilder struct {
	flags    []flagData
	values   []value
	required map[string]interface{} // map[flag name]pointers to the required fields to be able to check if they have been filled after the initialization
	extFns   []func() error
}

type value struct {
	name   string
	isBool bool
}

func newFlagBuilder() *flagBuilder {
	return &flagBuilder{
		required: make(map[string]interface{}),
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
			addFlagData(fb, fd)

		case bool:
			fd, err := parseFlagData(fld, flagMetadata, strconv.ParseBool)
			if err != nil {
				return err
			}
			addFlagData(fb, fd)

		case int:
			fd, err := parseFlagData(fld, flagMetadata, strconv.Atoi)
			if err != nil {
				return err
			}
			addFlagData(fb, fd)

		case int64:
			fd, err := parseFlagData(fld, flagMetadata, func(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 64)
			})
			if err != nil {
				return err
			}
			addFlagData(fb, fd)

		case uint:
			fd, err := parseFlagData(fld, flagMetadata, func(s string) (uint, error) {
				result, err := strconv.ParseUint(s, 10, 32)
				return uint(result), err
			})
			if err != nil {
				return err
			}
			addFlagData(fb, fd)

		case uint64:
			fd, err := parseFlagData(fld, flagMetadata, func(s string) (uint64, error) {
				return strconv.ParseUint(s, 10, 64)
			})
			if err != nil {
				return err
			}
			addFlagData(fb, fd)

		case float64:
			fd, err := parseFlagData(fld, flagMetadata, func(s string) (float64, error) {
				return strconv.ParseFloat(s, 64)
			})
			if err != nil {
				return err
			}
			addFlagData(fb, fd)

		case time.Duration:
			fd, err := parseFlagData(fld, flagMetadata, time.ParseDuration)
			if err != nil {
				return err
			}
			addFlagData(fb, fd)

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

func (fb *flagBuilder) attachFlags(fs *flag.FlagSet) {
	for _, flg := range fb.flags {
		switch f := flg.(type) {
		case *flagDataInstance[string]:
			fs.StringVar(f.addr, f.name, f.defaultVal, f.usage)
		case *flagDataInstance[bool]:
			fs.BoolVar(f.addr, f.name, f.defaultVal, f.usage)
		case *flagDataInstance[int]:
			fs.IntVar(f.addr, f.name, f.defaultVal, f.usage)
		case *flagDataInstance[int64]:
			fs.Int64Var(f.addr, f.name, f.defaultVal, f.usage)
		case *flagDataInstance[uint]:
			fs.UintVar(f.addr, f.name, f.defaultVal, f.usage)
		case *flagDataInstance[uint64]:
			fs.Uint64Var(f.addr, f.name, f.defaultVal, f.usage)
		case *flagDataInstance[float64]:
			fs.Float64Var(f.addr, f.name, f.defaultVal, f.usage)
		case *flagDataInstance[time.Duration]:
			fs.DurationVar(f.addr, f.name, f.defaultVal, f.usage)
		}
	}
}

func (fb *flagBuilder) parseFlags(args []string) error {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fb.attachFlags(fs)

	valueMap := make(map[string]bool, len(fb.values))
	for _, v := range fb.values {
		valueMap[fmt.Sprintf("-%s", v.name)] = v.isBool
	}
	for i := 0; i < len(args); i++ {
		chunks := strings.Split(args[i], "=")
		arg := chunks[0]
		if len(arg) > 2 && arg[:2] == "--" {
			arg = arg[1:]
		}
		hasValue, ex := valueMap[arg]
		if !ex {
			if arg == helpArg || arg == helpArgShort {
				continue
			} else {
				return fmt.Errorf("unexpected cli argument %q", arg)
			}
		}
		if !hasValue || len(chunks) > 1 {
			// -v
			continue
		}
		if i+1 < len(args) {
			i++
		}
	}

	return fs.Parse(args)
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

type flagData interface{}

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
func addFlagData[T any](fb *flagBuilder, fd *flagDataInstance[T]) {
	var a T
	_, isBool := any(a).(bool) // only booleans do not have to have a value
	fb.values = append(fb.values, value{
		name:   fd.value(),
		isBool: !isBool,
	})
	fb.flags = append(fb.flags, fd)
	if fd.isRequired {
		fb.required[fd.name] = fd.addr
	}
}
