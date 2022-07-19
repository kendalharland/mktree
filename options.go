package mktree

type Option interface {
	apply(*thread)
}

type option struct {
	applyFunc func(*thread)
}

func (o *option) apply(c *thread) {
	o.applyFunc(c)
}

func WithTemplateFunction(name string, f interface{}) Option {
	return &option{
		applyFunc: func(ctx *thread) {
			ctx.addTemplateFunc(name, f)
		},
	}
}
