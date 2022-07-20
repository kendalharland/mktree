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

### Variables

Variables are given as command-line arguments and may appear anywhere in the source
file. They are surrounded by `%(` and `)`.  Variables are given on the command line using
the `-vars` flag. Any leading or trailing whitespace around the variable's name is stripped
from the input source.

Variables are replaced in the source using string substitution during a preprocessing step. This happens before the input source is interpreted, meaning variables are not part of the mktree grammar.

__Example__

Given a file `layout.tree` with the following contents:

```
(file "%( filename   )")
```

This command will generate the file `example.txt`:

```
mktree layout.tree -vars=filename=example.txt
```

### Builtin Variables

#### root_dir

The variable `%(root_dir)` is always defined. It is the absolute path to the
directory where mktree will create all other files and directories. The caller
can set `root_dir` using the CLI's `-root` flag.  It is an error to attempt to
set the root dir by passing `-vars=root_dir=...`

## API Reference

### file

```
(file <filename> [attributes...])
```

Generates a regular file.

The filename must be a valid filename for the current platform. It is
evaluated relative to the root of its parent directory even if
it contains one or more leading slashes. If the file is declared at the
root level then the filename is evaluated relative to the `root_dir`.


__File attributes__


#### @contents

```
(@contents <value>)
```

Declares a string <value> to use as the file contents. Attempting to set both `@contents` and 
`@template` on the same file results in an error.

#### @perms

```
(@perms <mode>)
```

Declares the 32-bit Unix permissions to assign to the file. If this attribute 
is unset a default value of `0777` is used. This must be given as 4 octal digits
such as `0755`.

#### @template

```
(@template <filename> [variables...])
```

The path to a Go template file that this program should execute to generate the
contents of the file. The filename must be relative to the parent directory of the
input source file and the current user must have permission to read it.  Attempting
to set both `@contents` and `@template` on the same file results in an error. 

See the [templates](#template-files) section below for more information about templates.


### dir

```
(dir <dirname> [attributes... | children...])
```

Generates a directory.

The directory name must be a valid name for the current platform. It is
evaluated relative to the root of its parent directory even if
it contains one or more leading slashes. If the directory is declared at the
root level then the directory name is evaluated relative to the `root_dir`.
Directory attributes and children may be given in any order and directories may
have any number of children.

#### @perms

```
(@perms <mode>)
```

### link

```
(link <target> <link-name> [attributes...])
```

Creates a link to a file or directory.

The target is an absolute or relative path to another directory or file.
The link name is evaluated relative to the root of its parent directory even if
it contains one or more leading slashes. If the link is declared at the
root level then the link name is evaluated relative to the `root_dir`. By default
links are hard links. To make a symbolic link, use the `[@symbolic](#@symbolic)` attribute.

#### @perms

```
(@perms <mode>)
```

#### @symbolic

```
(@symbolic)
```

This attribute causes mktree to create a symbolic link instead of a hard one.


## Template Files

Template files are executed using Go's [template](https://pkg.go.dev/text/template)
package. mktree includes several built-in functions for use in templates, listed below.

```
%(snippet template_example examples/docs/examples.tree)
```

And the file `hello_world.txt.tmpl`:

```
Hello, %(first) %(last)!
```

This will generate the file `hello.txt` with the contents:

```
Hello, Example User!
```

### Builtin functions

#### FileExists

Returns true iff the file at the given path exists.

```
%(snippet file_exists_example examples/docs/template_example.txt.tmpl)
```

#### FileContents

Returns the contents of the file at the given path as a string. If the
file does not exist an error is generated and template execution exits
early.

```
%(snippet file_contents_example examples/docs/template_example.txt.tmpl)
```

#### Now

Returns the current time in the format described in [RFC-3339](https://datatracker.ietf.org/doc/html/rfc3339).

```
%(snippet now_example examples/docs/template_example.txt.tmpl)
```

#### User

Returns the name of the current User, or the empty string if the current User's name
cannot be determined.

```
%(snippet user_example examples/docs/template_example.txt.tmpl)
```

#### Var

Returns the value of any builtin or command-line [variable](#variables).

```
%(snippet var_example examples/docs/template_example.txt.tmpl)
```

#### Year

Returns the current year.

```
%(snippet year_example examples/docs/template_example.txt.tmpl)
```