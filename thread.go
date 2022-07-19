package mktree

type thread struct {
	templateFuncs map[string]interface{}
	sourceRoot    string
}

func (ctx *thread) addTemplateFunc(name string, f interface{}) {
	if ctx.templateFuncs == nil {
		ctx.templateFuncs = map[string]interface{}{}
	}
	ctx.templateFuncs[name] = f
}
