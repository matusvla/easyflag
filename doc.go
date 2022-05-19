/*
The easyflag package simplifies working with the native go flag package (https://pkg.go.dev/flag) by simplifying
the flag definition and parsing process. The flags are defined as structure field tags:

	type params struct {
		Str           string `flag:"str|Very important string||required"`
		Number        int    `flag:"num|Int with a default value|123|"`
		Boo           bool   `flag:"boo"`
		NotAFlagField string
	}

and an instance of this structure is filled simply by using the following code snippet in the main function:

	var p params
	if err := easyflag.ParseAndLoad(&p); err != nil {
		[...]
	}

Moreover, the package supports nested structures and user defined extensions executed immediately after the flag parsing.

Flag definition

Flags are defined as fields in a structure. The type of the flag corresponds to the type of the
field and the additional flag details are described using the `flag` field tag.
The currently supported field types are: string, bool, int, int64, uint, uint64, float64 and time.Duration.

The value of the flag field tag consists of four parts separated by the '|' character. Only the first value is
mandatory.

	The first value is the name of the matching CLI flag.
	The second value is the flag's usage description.
	The third value is the default value of this flag.
	The fourth value is used to specify that a flag is required. This overrides the default value of the flag.

The fields without the flag field tag are ignored.

Nested structures

There is a support for nested structures as well. This reduces boilerplate code as it allows for the reuse of predefined
blocks of CLI parameters.

User defined extensions

The passed structure can implement the Extender interface if there is a need for validation or modification
of the flag values passed by the user.
The structure's Extend method is then automatically called after the CLI flag values are loaded.

If any of the nested substructures implements the Extender interface, its Extend method is called as well.

Usage notes

- The package does not distinguish between the flag form with one and two leading hyphens (e.g. -help and --help are
both valid, and they mean the same).

- The allowed form of a boolean flag is either -boo without any value or -boo=true for an explicit value setup.
This corresponds to the behavior of the native go flag package.

- For any field type other than boolean both forms -str val and str=val are allowed.

- There are two reserved flags -h and -help. If a user provides one of these, only the information about
the available flags is printed ant the program exits.
*/
package easyflag
