---
title: "Home"
date: Tue, 12 Jul 2022 16:06:53 EDT
---

**mktree** is an s-expression based configuration language for generating boilerplate files and directories.

With mktree, writing a code generator is as easy as declaring the filesystem layout:

```
; file: layout.tree
(dir "users/%(username)"
  (file "README.md"
      (@contents "This directory belongs to %(username)")
      (@perms 0666)))
```

And running `mktree`:

```
mktree -vars=username=kendal layout.tree 
```


* To get started, check out some of the [examples](https://github.com/kendalharland/mktree/tree/main/examples)

* For more details, check out the [reference](posts/reference/).
