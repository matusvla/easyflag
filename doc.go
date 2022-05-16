/*
Package easyflag ...
TODO
Example of the input structure:

type Params struct {
	Str       string        `flag:"str|Testing string||required"`
	Str2      string        `flag:"str2|Testing string2|Str2 default|"`
	Boo       bool          `flag:"boo|Testing boolean|true|"`
	Number    int           `flag:"num|Testing number|123|"`
	ExtNumber int           `flag:"extnum|Extender testing number|"`
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
*/
package easyflag
