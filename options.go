package mktree

// Option affects the way filesystem enties are created.
type Option interface {
	apply(*thread)
}

// WithTemplateFunction registers the template function f under the give name.
//
// The function will be made available my templates.
func WithTemplateFunction(name string, f interface{}) Option {
	return &option{
		applyFunc: func(t *thread) {
			t.addTemplateFunc(name, f)
		},
	}
}

type option struct {
	applyFunc func(*thread)
}

func (o *option) apply(c *thread) {
	o.applyFunc(c)
}
