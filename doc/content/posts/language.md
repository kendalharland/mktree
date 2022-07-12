---
title: "Language"
date: Tue, 12 Jul 2022 16:06:53 EDT
draft: false
---

## Grammar

```
config    = sexpr*
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
