---
title: "Language"
date: Tue, 12 Jul 2022 16:06:53 EDT
draft: false
---

{{<toc>}}

## Introduction

This page describes the design and behavior of the language.

## Preprocessing

### Variables

Variables are alphabetic literals surrounded by `%(` and `)`. They are not
interpreted. Instead they are replaced in the source using string substitution.
For more information about variables see the [Variables](/posts/reference/#variables)
section of the [reference](/posts/reference/).

## Language

mktree uses a basic [S-expression] ("sexpr") syntax to describe a tree of filesystem
entities to create. The types are:

- Strings
- Numbers (unsigned integers)
- Attributes
- Filesystem entities

### Strings

Strings are enclosed in double quotes. There are no supported escape sequences.

```
"This is a string."
```

### Numbers

Numbers are parsed as unsigned integers. The bit-width is determined by the context
when the number is evaluated. For example, when the number appears as `@perm` attribute
on a directory or file, it is evaluated as an unsigned 32-bit integer and used as a file
mode.

### Attributes

Attributes begin with `@` and are used to assign metadata to filesystem entities. Each
entity supports different attributes. For example, the `@contents` attribute is used to
set the contents of a `file`, but it's invalid to set `@contents` on `dir`.

### Filesystem entities

Filesystem entities such as directories and files are declared as sexprs. Each entity
requires a name and may accept several attributes unique to the entity type. For more
information about each kind of entity see the [Filesystem entities](/posts/reference/#file-system-entities) section of the [reference](/posts/reference/).

### Comments

Comments begin with ';' and span to the end of the current line.
Comments may appear anywhere the source, including in the middle of sexprs.

```
; This is a comment
```


## Grammar

```
tree      = stmt*
stmt      = comment
          | sexpr
comment   = ';' [^\n]*
sexpr     = '(' literal arg* ')'
arg       = sexpr
          | literal
literal   = KEYWORD
          | ATTRIBUTE
          | STRING
          | NUMBER
KEYWORD   = 'dir' | 'file' | 'link'
ATTRIBUTE = '@' [a-zA-Z0-9_-]+
STRING    = '"'[^"]*'"'
NUMBER    = [0-9]+
```


[S-expression]: https://en.wikipedia.org/wiki/S-expression