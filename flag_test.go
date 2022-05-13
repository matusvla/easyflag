package cli

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type Params struct {
	Str           string        `flag:"str|Testing string||mandatory"`
	Str2          string        `flag:"str2|Testing string2|Str2 default|"`
	Boo           bool          `flag:"boo|Testing boolean|true|"`
	Number        int           `flag:"num|Testing number|123|"`
	ExtNumber     int           `flag:"extnum|Extension testing number|"`
	Number64      int64         `flag:"num64|Testing number|1234|"`
	UNumber       uint          `flag:"unum|Testing number|12345|mandatory"`
	UNumber64     uint64        `flag:"unum64|Testing number|123456|"`
	Float64       float64       `flag:"fnum64|Testing number|123.456|"`
	Dur           time.Duration `flag:"dur|Testing number|10m|"`
	NotAFlagField string
}

func (p *Params) Extend() error {
	p.ExtNumber = 9999999
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
				Str       string `flag:"str|Testing string||mandatory"`
				Substruct struct {
					Str2 string `flag:"str2|Testing string2|Str2 default|"`
				}
			}{},
			want: want{
				params: &struct {
					Str       string `flag:"str|Testing string||mandatory"`
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
			cliParams: []string{"-str=asdf", "-str2", "fdsa", "-unum=10", "random", "stuff"},
			arg:       &Params{},
			want: want{
				err:    errors.New("unexpected cli argument \"random\""),
				params: &Params{},
			},
		},
		{
			name:      "fail - validation flags",
			cliParams: []string{"-str=asdf"},
			arg:       &Params{},
			want: want{
				err:    errors.New("missing mandatory flag \"unum\" or its value"),
				params: &Params{},
			},
		},
		{
			name:      "success- nested params",
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
			name:      "failure - params not a pointer",
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
			name:      "failure - nil params",
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
			name:      "failure - params pointer, but not a struct",
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
			name:      "failure in extension",
			cliParams: []string{},
			arg:       &FailingParams{},
			want: want{
				err:    fmt.Errorf("extension running failed: %w", failingParamsErr),
				params: &FailingParams{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = []string{"executable_name"}
			os.Args = append(os.Args, tt.cliParams...)
			err := ParseAndLoadFlags(tt.arg)
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
