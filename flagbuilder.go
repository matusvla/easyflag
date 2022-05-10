package cli

import (
	"errors"
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type flagBuilder struct {
	flags     []flagData
	values    []string
	mandatory map[string]interface{} // map[flag name]pointers to mandatory fields to be able to check if they have been filled after the initialization
	extFns    []func() error
}

func newFlagBuilder() *flagBuilder {
	return &flagBuilder{
		mandatory: make(map[string]interface{}),
	}
}

func (fb *flagBuilder) setUpFlags(cliParams interface{}) (err error) {
	cliV := reflect.ValueOf(cliParams)
	cliT := reflect.TypeOf(cliParams)
	if cliV.Kind() != reflect.Ptr {
		return errors.New("got non-pointer type")
	}
	cliV = cliV.Elem()
	cliT = cliT.Elem()
	if cliV.Kind() != reflect.Struct {
		return fmt.Errorf("got unexpected type kind %s (type %q), needs struct kind", cliV.Kind(), cliV.Type())
	}

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
			panic(fmt.Sprintf("unsupported flag type: %v", tpe))
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

func (fb *flagBuilder) parseFlags(args []string) (isHelpRequest bool, err error) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fb.attachFlags(fs)
	m := make(map[string]bool)
	for _, name := range fb.values {
		hasValue := strings.HasSuffix(name, "=")
		m["-"+strings.ReplaceAll(name, "=", "")] = hasValue
	}
	var filtered []string
	for i := 0; i < len(args); i++ {
		chunks := strings.Split(args[i], "=")
		arg := strings.ReplaceAll(chunks[0], "--", "-")
		hasValue, ex := m[arg]
		if !ex {
			if arg == helpArg || arg == helpArgShort {
				isHelpRequest = true
			} else {
				return false, fmt.Errorf("unexpected cli parameter %q", arg)
			}
		}
		if !hasValue {
			// -v
			filtered = append(filtered, arg)
			continue
		}
		// has value, either using = or just by next token
		if len(chunks) > 1 {
			// --conf=someValue
			filtered = append(filtered, args[i])
			continue
		}
		// -c
		if i+1 < len(args) {
			// someValue
			filtered = append(filtered, args[i:i+2]...)
			i++
		}
	}
	return isHelpRequest, fs.Parse(filtered)
}

func (fb *flagBuilder) validate() error {
	var missing []string
	for mandatoryKey, mandatoryValue := range fb.mandatory {
		fld := reflect.ValueOf(mandatoryValue).Elem()
		if fld.IsZero() {
			missing = append(missing, mandatoryKey)
		}
	}
	switch len(missing) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("missing mandatory flag %q or its value", strings.Join(missing, ", "))
	default:
		return fmt.Errorf("missing mandatory flags %q or their values", strings.Join(missing, ", "))
	}

}

type flagData interface{}

type basicFlagData struct {
	name      string
	withValue bool
	usage     string
	mandatory bool
}

type flagDataInstance[T any] struct {
	*basicFlagData
	addr       *T
	defaultVal T
}

func (fdi *basicFlagData) value() string {
	if !fdi.withValue {
		return fdi.name
	}
	return fdi.name + "="
}

func parseFlagData[T any](fld reflect.Value, flagMetadata string, tParser func(string) (T, error)) (*flagDataInstance[T], error) {
	basicData, flagDefault := parseBasicFlagData(flagMetadata)
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
		basicFlagData: basicData,
		addr:          addr,
		defaultVal:    defaultVal,
	}
	return f, nil
}

func parseBasicFlagData(flagMetadata string) (flagData *basicFlagData, flagDefault string) {
	fp := strings.Split(flagMetadata, "|")
	flagName := strings.TrimSpace(fp[0])
	var (
		flagUsage     string
		flagMandatory bool
	)
	if len(fp) > 1 {
		flagUsage = strings.TrimSpace(fp[1])
	}
	if len(fp) > 2 {
		flagDefault = strings.TrimSpace(fp[2])
	}
	if len(fp) > 3 {
		// here is space for extending the flag checking
		if fp[3] == mandatoryValueIdent {
			flagDefault = "" // if it is mandatory, we ignore default value
			flagMandatory = true
		}
	}
	return &basicFlagData{
		name:      strings.TrimSuffix(flagName, "="),
		withValue: strings.HasSuffix(flagName, "="),
		usage:     flagUsage,
		mandatory: flagMandatory,
	}, flagDefault
}

// this currently cannot be a flagBuilder method due to the type parameters usage
func addFlagData[T any](fb *flagBuilder, fd *flagDataInstance[T]) {
	fb.values = append(fb.values, fd.value())
	fb.flags = append(fb.flags, fd)
	if fd.mandatory {
		fb.mandatory[fd.name] = fd.addr
	}
}
