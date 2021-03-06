package easyflag

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type Params struct {
	Str           string        `flag:"str|Testing string||required"`
	Str2          string        `flag:"str2|Testing string2|Str2 default|"`
	Boo           bool          `flag:"boo|Testing boolean|true|"`
	Number        int           `flag:"num|Testing number|123|"`
	ExtNumber     int           `flag:"extnum|Extender testing number|"`
	Number64      int64         `flag:"num64|Testing number|1234|"`
	UNumber       uint          `flag:"unum|Testing number|12345|required"`
	UNumber64     uint64        `flag:"unum64|Testing number|123456|"`
	Float64       float64       `flag:"fnum64|Testing number|123.456|"`
	Dur           time.Duration `flag:"dur|Testing number|10m|"`
	NotAFlagField string
}

func (p *Params) Extend() error {
	p.ExtNumber = 9_999_999
	return nil
}

func TestParseFlags(t *testing.T) {
	type want struct {
		err    error
		params interface{}
	}
	tests := []struct {
		name      string
		cliParams []string
		arg       interface{}
		want      want
	}{
		{
			name:      "success",
			cliParams: []string{"--str=asdf", "-str2", "fdsa", "-boo", "-num=15", "--num64", "16", "-unum=17", "-unum64=18", "-dur=5m"},
			arg:       &Params{},
			want: want{
				params: &Params{
					Str:       "asdf",
					Str2:      "fdsa",
					Boo:       true,
					Number:    15,
					ExtNumber: 9999999,
					Number64:  16,
					UNumber:   17,
					UNumber64: 18,
					Float64:   123.456,
					Dur:       5 * time.Minute,
				},
			},
		},
		{
			name:      "success substructure",
			cliParams: []string{"-str=asdf", "-str2", "fdsa"},
			arg: &struct {
				Str       string `flag:"str|Testing string||required"`
				Substruct struct {
					Str2 string `flag:"str2|Testing string2|Str2 default|"`
				}
			}{},
			want: want{
				params: &struct {
					Str       string `flag:"str|Testing string||required"`
					Substruct struct {
						Str2 string `flag:"str2|Testing string2|Str2 default|"`
					}
				}{
					Str: "asdf",
					Substruct: struct {
						Str2 string `flag:"str2|Testing string2|Str2 default|"`
					}{
						Str2: "fdsa",
					},
				},
			},
		},
		{
			name:      "success - fields without flags",
			cliParams: []string{"-str=asdf"},
			arg: &struct {
				Str          string `flag:"str|Testing string||required"`
				AnotherField struct {
					SubstrStr string
				}
			}{},
			want: want{
				params: &struct {
					Str          string `flag:"str|Testing string||required"`
					AnotherField struct {
						SubstrStr string
					}
				}{
					Str: "asdf",
					AnotherField: struct {
						SubstrStr string
					}{},
				},
			},
		},
		{
			name:      "success boolean in allowed forms",
			cliParams: []string{"-boo", "-boo2=true", "-boo3=false"},
			arg: &struct {
				Boo  bool `flag:"boo"`
				Boo2 bool `flag:"boo2"`
				Boo3 bool `flag:"boo3"`
			}{},
			want: want{
				params: &struct {
					Boo  bool `flag:"boo"`
					Boo2 bool `flag:"boo2"`
					Boo3 bool `flag:"boo3"`
				}{
					Boo:  true,
					Boo2: true,
					Boo3: false,
				},
			},
		},
		{
			name:      "fail - invalid flags",
			cliParams: []string{"-str=asdf", "-str2", "fdsa", "-unum=10", "-random", "stuff"},
			arg:       &Params{},
			want: want{
				err:    errors.New("flag provided but not defined: -random"),
				params: &Params{},
			},
		},
		{
			name:      "fail - missing a required flag",
			cliParams: []string{"-str=asdf"},
			arg:       &Params{},
			want: want{
				err:    errors.New("missing required flag \"unum\" or its value"),
				params: &Params{},
			},
		},
		{
			name:      "fail - trying to overwrite the short help flag",
			cliParams: []string{""},
			arg: &struct {
				Boo bool `flag:"h"`
			}{},
			want: want{
				params: &struct {
					Boo bool `flag:"h"`
				}{},
				err: errors.New("reserved flag -h overwriting not allowed"),
			},
		},
		{
			name:      "fail - trying to overwrite the long help flag",
			cliParams: []string{""},
			arg: &struct {
				Boo bool `flag:"help"`
			}{},
			want: want{
				params: &struct {
					Boo bool `flag:"help"`
				}{},
				err: errors.New("reserved flag -help overwriting not allowed"),
			},
		},
		{
			name:      "success - nested params",
			cliParams: []string{"-str=asdf", "-str2", "fdsa", "-boo", "-num=15", "-num64", "16", "-unum=17", "-unum64=18", "-dur=5m"},
			arg:       &NestedParams{},
			want: want{
				params: &NestedParams{
					Params: Params{
						Str:       "asdf",
						Str2:      "fdsa",
						Boo:       true,
						Number:    15,
						ExtNumber: 9999999,
						Number64:  16,
						UNumber:   17,
						UNumber64: 18,
						Float64:   123.456,
						Dur:       5 * time.Minute,
					},
					AnotherThing: "extended",
				},
			},
		},
		{
			name:      "fail - params not a pointer",
			cliParams: []string{"-str=asdf", "-str2", "fdsa", "-boo", "-num=15", "-num64", "16", "-unum=17", "-unum64=18", "-dur=5m"},
			arg:       Params{},
			want: want{
				err: &InvalidParamsError{
					Type: reflect.TypeOf(Params{}),
				},
				params: Params{},
			},
		},
		{
			name:      "fail - nil params",
			cliParams: []string{"-str=asdf", "-str2", "fdsa", "-boo", "-num=15", "-num64", "16", "-unum=17", "-unum64=18", "-dur=5m"},
			arg:       nil,
			want: want{
				err: &InvalidParamsError{
					Type: reflect.TypeOf(nil),
				},
				params: nil,
			},
		},
		{
			name:      "fail - params pointer, but not a struct",
			cliParams: []string{"-str=asdf", "-str2", "fdsa", "-boo", "-num=15", "-num64", "16", "-unum=17", "-unum64=18", "-dur=5m"},
			arg:       strPointer,
			want: want{
				err: &InvalidParamsError{
					Type: reflect.TypeOf(strPointer),
				},
				params: strPointer,
			},
		},
		{
			name:      "fail in extension",
			cliParams: []string{},
			arg:       &FailingParams{},
			want: want{
				err:    fmt.Errorf("extension running failed: %w", failingParamsErr),
				params: &FailingParams{},
			},
		},
		{
			name:      "fail - nil",
			cliParams: nil,
			arg:       nil,
			want: want{
				err:    &InvalidParamsError{Type: nil},
				params: nil,
			},
		},
		{
			name:      "fail - fourth segment invalid",
			cliParams: []string{""},
			arg: &struct {
				Boo bool `flag:"str|Testing string||whatever"`
			}{},
			want: want{
				params: &struct {
					Boo bool `flag:"str|Testing string||whatever"`
				}{},
				err: errors.New("unsupported value \"whatever\" in the fourth metadata part"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = []string{"executable_name"}
			os.Args = append(os.Args, tt.cliParams...)
			err := ParseAndLoad(tt.arg)
			assert.Equal(t, tt.want.err, err)
			assert.Equal(t, tt.want.params, tt.arg)
		})
	}
}

type NestedParams struct {
	Params       Params
	AnotherThing string `flag:"AnotherThing|Testing string|"`
}

func (np *NestedParams) Extend() error {
	np.AnotherThing = "extended"
	return nil
}

var strPointer = func() *string {
	a := "wrong params"
	return &a
}()

var failingParamsErr = errors.New("mock error in extension")

type FailingParams struct {
	NotImportant string `flag:"ni|Testing string|"`
}

func (np *FailingParams) Extend() error {
	return failingParamsErr
}

func TestInvalidParamsError_Error(t *testing.T) {
	tests := []struct {
		name    string
		fldType reflect.Type
		want    string
	}{
		{
			name:    "non-pointer",
			fldType: reflect.TypeOf(5),
			want:    "flags parse: got non-pointer int",
		},
		{
			name: "not structure",
			fldType: reflect.TypeOf(func() *int {
				a := 5
				return &a
			}()),
			want: "flags parse: got *int",
		},
		{
			name:    "nil",
			fldType: nil,
			want:    "flags parse: got <nil>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &InvalidParamsError{
				Type: tt.fldType,
			}
			assert.Equalf(t, tt.want, e.Error(), "Error()")
		})
	}
}

func BenchmarkParseAndLoadFlags(b *testing.B) {
	os.Args = []string{"executable_name", "--str=asdf", "-str2", "fdsa", "-boo", "-num=15", "--num64", "16", "-unum=17", "-unum64=18", "-dur=5m"}
	for i := 0; i < b.N; i++ {
		var p Params
		err := ParseAndLoad(&p)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkOrdinaryFlags(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var p Params
		fs := flag.NewFlagSet("", flag.PanicOnError)
		fs.StringVar(&p.Str, "str", "", "Testing string")
		fs.StringVar(&p.Str2, "str2", "Str2 default", "Testing string2")
		fs.BoolVar(&p.Boo, "boo", true, "Testing boolean")
		fs.IntVar(&p.Number, "num", 123, "Testing number")
		fs.IntVar(&p.ExtNumber, "extnum", 0, "Extender testing number")
		fs.Int64Var(&p.Number64, "num64", 1234, "Testing number")
		fs.UintVar(&p.UNumber, "unum", 12345, "Testing number")
		fs.Uint64Var(&p.UNumber64, "unum64", 123456, "Testing number")
		fs.Float64Var(&p.Float64, "fnum64", 123.456, "Testing number")
		fs.DurationVar(&p.Dur, "dur", 10*time.Minute, "Testing number")
		if err := fs.Parse([]string{"--str=asdf", "-str2", "fdsa", "-boo", "-num=15", "--num64", "16", "-unum=17", "-unum64=18", "-dur=5m"}); err != nil {
			panic(err)
		}
		p.ExtNumber = 9_999_999
	}
}

// Example_basic demonstrates the basic usage of the package.
func Example_basic() {
	var p struct {
		InputPath string `flag:"in|Path to the input file||required"`
		OutputLen int64  `flag:"n|Maximum number of characters to read (-1 for all)|-1"`
	}

	if err := ParseAndLoad(&p); err != nil {
		log.Fatalf("error while parsing the cli parameters: %s", err.Error())
	}
}

// Example_basic demonstrates the usage of nested structures.
func Example_nested() {
	type userAuth struct {
		Username string `flag:"user|Username||required"`
		Password string `flag:"pass|Passsword"`
	}

	type serverInfo struct {
		Host string `flag:"a|Server host address|127.0.0.1"`
		Port int    `flag:"p|Server port|80"`
	}

	var p struct {
		UserAuth   userAuth
		ServerInfo serverInfo
	}

	if err := ParseAndLoad(&p); err != nil {
		log.Fatalf("error while parsing the cli parameters: %s", err.Error())
	}
}
