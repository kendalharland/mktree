---
title: "Reference"
date: Tue, 12 Jul 2022 16:06:53 EDT
draft: false
---

{{<toc>}}

## The command-line interface

```
%(content cli-usage)
```

## Concepts

### Variables

Variables may appear anywhere in the source file and must be surrounded by
`%(` and `)`.  Variables are inserted using simple string substitution by a
preprocessing step before code is interpreted. This means that variables are
not part of the mktree grammar.

Example:  Given a file `layout.tree` with the following contents:

```
(file "%(filename)")
```

This command will generate the file `example.txt`:

```
mktree layout.tree -vars=filename=example.txt
```

### Builtin Variables

#### root_dir

The variable `%(root_dir)` is always defined. It is the absolute path to the
directory where mktree will create all other files and directories. The caller
can set `root_dir` like any other variable using mktree's `-var` flag.

## Filesystem entities

### file

```
(file <filename> [attributes...])
```

Generates a regular file.

The filename must be a valid filename for the current platform. The filename is
evaluated relative to the root of its parent directory even if it starts with
one or more leading slashes. If the file is declared at the root level then the
filename is evaluated relative to the root directory.

__File attributes__

#### @contents

```
(@contents <value>)
```

Declares a string <value> to write to the the file. This is mutually exclusive
with `@template`; Attempting to set both `@contents` and `@template` on the
same file file results in an interpet error.

#### @perms

```
(@perms <mode>)
```

Declares the 32-bit Unix permissions to assign to the file. If this attribute 
is unset a default value of `0666` is used. This is typically given as a four
digit hexadecimal value such as `0755`.

#### @template

```
(@template <filename> [variables...])
```

The path to a Go template file that this program should execute to generate the
contents of the file.  The current user must have permission to read the template
file. See the [templates](#templates) section below for more information.

```
%(snippet template_example examples/examples.tree)
```

And the file `hello_world.txt.tmpl`:

```
Hello, %(first) %(last)!
```

This will generate the file `hello.txt` with the contents:

```
Hello, Example User!
```

__File examples__

```
%(snippet file_example examples/examples.tree)
```

### dir

```
(dir <dirname> [attributes... | children...])
```

Generates a directory.

The directory name must be a valid name for the current platform. The directory
path is evaluated relative to the root of its parent directory even if it starts
with one or more leading slashes. If the directory is declared at the root level
then the path is evaluated relative to the root directory. Directory attributes
and children may be given in any order, and directories may have any number of
children.

__Examples__

```
%(snippet dir_example examples/examples.tree)
```

#### @perms

```
(@perms <mode>)
```

## Template Files

Template files are executed using Go's [template](https://pkg.go.dev/text/template)
package.  mktree includes several built-in functions for use in templates, listed below.

### Builtin functions

#### FileExists

Returns true iff the file at the given path exists.

```
%(snippet file_exists_example examples/template_example.txt.tmpl)
```

#### FileContents

Returns the contents of the file at the given path as a string. If the
file does not exist an error is generated and template execution exits
early.

```
%(snippet file_contents_example examples/template_example.txt.tmpl)
```

#### Now

Returns the current time in the format described in [RFC-3339](https://datatracker.ietf.org/doc/html/rfc3339).

```
%(snippet now_example examples/template_example.txt.tmpl)
```

#### User

Returns the name of the current User, or the empty string if the current User's name
cannot be determined.

```
%(snippet user_example examples/template_example.txt.tmpl)
```
