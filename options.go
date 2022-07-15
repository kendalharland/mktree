package mktree

type Option interface {
	apply(*treeContext)
}

type option struct {
	applyFunc func(*treeContext)
}

func (o *option) apply(c *treeContext) {
	o.applyFunc(c)
}

func WithTemplateFunction(name string, f interface{}) Option {
	return &option{
		applyFunc: func(ctx *treeContext) {
			ctx.addTemplateFunc(name, f)
		},
	}
}
