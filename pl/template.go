package pl

import (
	"bytes"
	"fmt"

	// go template
	"text/template"

	// pongo
	"github.com/flosch/pongo2"

	// markdown
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

type Template interface {
	Compile(name, input string, opt Val) error
	Execute(context Val) (string, error)
}

type TemplateFactory interface {
	Create() Template
}

type goTemplate struct {
	goT *template.Template
}

func (t *goTemplate) Compile(name, input string, _ Val) error {
	tp, err := template.New(name).Parse(input)
	if err != nil {
		return err
	}
	t.goT = tp
	return nil
}

// convert a context value into a template context to be accessed by the go
// template engine
func toctx(ctx Val) (interface{}, error) {
	if ctx.IsNull() {
		return nil, nil
	}

	switch ctx.Type {
	case ValNull:
		return nil, nil
	case ValInt:
		return ctx.Int(), nil
	case ValReal:
		return ctx.Real(), nil
	case ValStr:
		return ctx.String(), nil
	case ValBool:
		return ctx.Bool(), nil

	case ValPair:
		k, e1 := toctx(ctx.Pair().First)
		v, e2 := toctx(ctx.Pair().Second)
		if e1 != nil {
			return nil, e1
		}
		if e2 != nil {
			return nil, e2
		}
		return map[string]interface{}{
			"first":  k,
			"second": v,
		}, nil

	case ValList:
		o := []interface{}{}
		for _, d := range ctx.List().Data {
			v, err := toctx(d)
			if err != nil {
				return nil, err
			}
			o = append(o, v)
		}
		return o, nil

	case ValMap:
		out := map[string]interface{}{}

		var err error
		rErr := &err

		ctx.Map().Foreach(
			func(key string, value Val) bool {
				v, err := toctx(value)
				if err != nil {
					*rErr = err
					return false
				}
				out[key] = v
				return true
			},
		)

		if err != nil {
			return nil, err
		} else {
			return out, nil
		}

	default:
		return nil, fmt.Errorf("invalid context type")
	}
}

func (t *goTemplate) Execute(ctx Val) (string, error) {
	x := new(bytes.Buffer)
	if cctx, err := toctx(ctx); err != nil {
		return "", err
	} else if err := t.goT.Execute(x, cctx); err != nil {
		return "", err
	}
	return x.String(), nil
}

// for now markdown is static at all, ie no runtime rendering what's so ever
type mdTemplate struct {
	md string
}

func (t *mdTemplate) Compile(_, input string, _ Val) error {
	r := html.NewRenderer(
		html.RendererOptions{Flags: html.CommonFlags})

	txt := markdown.ToHTML([]byte(input), nil, r)
	t.md = string(txt)
	return nil
}

func (t *mdTemplate) Execute(ctx Val) (string, error) {
	return t.md, nil
}

type pongoTemplate struct {
	tpl *pongo2.Template
}

func (t *pongoTemplate) Compile(_, input string, _ Val) error {
	r, err := pongo2.FromString(input)
	if err != nil {
		return err
	}
	t.tpl = r
	return nil
}

func (t *pongoTemplate) tocontext(v Val) (pongo2.Context, error) {
	switch v.Type {
	case ValPair:

		k, e1 := toctx(v.Pair().First)
		if e1 != nil {
			return pongo2.Context{}, e1
		}

		v, e2 := toctx(v.Pair().Second)
		if e2 != nil {
			return pongo2.Context{}, e2
		}

		return pongo2.Context{
			"first":  k,
			"second": v,
		}, nil

	case ValMap:
		p := make(pongo2.Context)
		var err error
		rErr := &err

		v.Map().Foreach(
			func(k string, v Val) bool {
				vv, err := toctx(v)
				if err != nil {
					*rErr = err
					return false
				}
				p[k] = vv
				return true
			},
		)
		if err != nil {
			return nil, err
		} else {
			return p, nil
		}

	default:
		return make(pongo2.Context), nil
	}
}

func (t *pongoTemplate) Execute(ctx Val) (string, error) {
	cctx, err := t.tocontext(ctx)
	if err != nil {
		return "", err
	}
	return t.tpl.Execute(cctx)
}

type gotempfac struct{}

func (f *gotempfac) Create() Template {
	return &goTemplate{}
}

type mdtempfac struct{}

func (f *mdtempfac) Create() Template {
	return &mdTemplate{}
}

type pongotempfac struct{}

func (f *pongotempfac) Create() Template {
	return &pongoTemplate{}
}

// Public interface to allow user to register multiple different template engine
// into PL language environment for customization
var templatefacmap = make(map[string]TemplateFactory)

func AddTemplateFactory(name string, f TemplateFactory) {
	templatefacmap[name] = f
}

func init() {
	AddTemplateFactory("go", &gotempfac{})
	AddTemplateFactory("md", &mdtempfac{})
	AddTemplateFactory("pongo", &pongotempfac{})
}

func newTemplate(t string) Template {
	f, ok := templatefacmap[t]
	if ok {
		return f.Create()
	} else {
		return nil
	}
}
