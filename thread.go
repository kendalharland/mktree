package mktree

import "path/filepath"

type thread struct {
	templateFuncs map[string]interface{}
	sourceRoot    string
}

func newThread(filename string, opts ...Option) *thread {
	sourceRoot := filepath.Dir(filename)
	if sourceRoot == "" {
		sourceRoot = "."
	}

	t := &thread{sourceRoot: sourceRoot}
	for _, o := range opts {
		o.apply(t)
	}
	return t
}

func (thr *thread) addTemplateFunc(name string, f interface{}) {
	if thr.templateFuncs == nil {
		thr.templateFuncs = map[string]interface{}{}
	}
	thr.templateFuncs[name] = f
}
