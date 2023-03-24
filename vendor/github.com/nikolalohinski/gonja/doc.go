// A jinja like template-engine
//
// Make sure to read README.md in the repository as well.
//
// A tiny example with template strings:
//
//     // Compile the template first (i. e. creating the AST)
//     tpl, err := gonja.FromString("Hello {{ name | capfirst }}!")
//     if err != nil {
//         panic(err)
//     }
//     // Now you can render the template with the given
//     // gonja.Context how often you want to.
//     out, err := tpl.Execute(gonja.Context{"name": "fred"})
//     if err != nil {
//         panic(err)
//     }
//     fmt.Println(out) // Output: Hello Fred!
//
package gonja

import (
	_ "github.com/nikolalohinski/gonja/docs"
)
