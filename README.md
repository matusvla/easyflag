# Easy flag

The **easyflag** package simplifies working with the native go [flag](https://pkg.go.dev/flag) package by simplifying
the flag definition and parsing process. The flags are defined as the struct field tags instead:

```go
type Params struct {
    Str           string        `flag:"str|Very important string||required"`
    Number        int           `flag:"num|Int with a default value|123|"`
    Boo           bool          `flag:"boo`
    NotAFlagField string
}
```

Moreover, the package supports [nested structures](#nested-structures) and [user defined extensions](#user-defined-extensions) executed immediately after the flag parsing.

- The currently supported field types are: `string`, `bool`, `int`, `int64`, `uint`, `uint64`, `float64`
and`time.Duration`.
- The package does not distinguish between the flag form with one and two leading hyphens (e.g. `-help` and `--help` are both valid, and they mean the same)
- The allowed form of a boolean flag is either `-boo` without any value or `-boo=true` for an explicit value setup. This corresponds to the behavior of the native go [flag](https://pkg.go.dev/flag) package. 
- For any field type other than boolean both forms `-str val` and `str=val` are allowed
- There are two reserved flags `-h` and `-help`. If a user provides one of these, only the information about
  the available flags is printed ant the program exits.

## Flag definition

Flags are defined as fields in a structure. The type of the flag corresponds to the type of the
field and the additional flag details are described using the `flag` field tag.

The value of the `flag` field tag consists of four parts separated by the `|` character. Only the first value is
mandatory.

- The first value is the **name** of the matching CLI flag. Bool flags can be passed without an explicit value which
  sets the underlying field to true.
- The second value is the **flag's usage description**.
- The third value is the **default value** of this flag.
- The fourth value is used to specify that a flag is **required**. This overrides the default value of the flag.

The fields without the `flag` field tag are ignored.

## Nested structures

The nested structures are supported as well. This reduces boilerplate code as it allows for the reuse of predefined params blocks.

```go
type BuildVersionFlag struct {
    HasVersionPrintout bool `flag:"v|prints out version (in semver aka v1.4.78 format)"`
}

type Params struct {
  Str           string        `flag:"str|Very important string||required"`
  Version BuildVersionFlag
}
```

## User defined extensions

The parameter validation or modification can be done by implementing the `Extender` interface. 
The method `Extend() error` is then automatically called after the command line parameters parsing.

If any of the substructures implement the  `Extender` interface, its `Extend` method is called as well.

Example of the validation:

```go
type Params struct {
  DayInMonth           int        `flag:"d|Day in month|"`
}

func (p *Params) Extend() error {
	if d := p.DayInMonth; d < 0 || d > 31 {
		return fmt.Errorf("invalid day in month %d", d)
    }
	return nil
}
```