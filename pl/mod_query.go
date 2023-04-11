package pl

import (
	"fmt"
)

// ---------------------------------------------------------------------------
// 1) Anchor operations
func qFirst(
	info *IntrinsicInfo,
	_ *Evaluator,
	_ string,
	args []Val,
) (Val, error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}

	a := args[0]
	if a.IsList() {
		if a.List().Length() == 0 {
			return NewValNull(), fmt.Errorf("list is empty")
		}
		return a.List().At(0), nil
	} else {
		must(a.IsPair(), "must be pair")
		return a.Pair().First, nil
	}
}

func qLast(
	info *IntrinsicInfo,
	_ *Evaluator,
	_ string,
	args []Val,
) (Val, error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}
	a := args[0]
	if a.IsList() {
		if a.List().Length() == 0 {
			return NewValNull(), fmt.Errorf("list is empty")
		}
		return a.List().At(a.List().Length() - 1), nil
	} else {
		must(a.IsPair(), "must be pair")
		return a.Pair().Second, nil
	}
	return NewValNull(), fmt.Errorf("type %s does not support q::last", a.Id())
}

func qRest(
	info *IntrinsicInfo,
	_ *Evaluator,
	_ string,
	args []Val,
) (Val, error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}
	a := args[0]
	if a.IsList() {
		if a.List().Length() == 0 {
			return NewValList(), nil
		}
		o := NewValList()
		for _, v := range a.List().Data[1:] {
			o.AddList(v)
		}
		return o, nil
	} else {
		must(a.IsPair(), "must be pair")
		x := NewValList()
		x.AddList(a.Pair().Second)
		return x, nil
	}

	return NewValNull(), fmt.Errorf("type %s does not support q::last", a.Id())
}

// ---------------------------------------------------------------------------
// 2) Projection
func qSelect(
	info *IntrinsicInfo,
	_ *Evaluator,
	_ string,
	args []Val,
) (Val, error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}
	a0 := args[0]
	if a0.IsList() {
		l := a0.List()
		o := NewValList()
		for _, v := range args[1:] {
			idx := int(v.Int())
			if idx >= 0 && idx <= l.Length() {
				o.AddList(l.Data[idx])
			}
		}
		return o, nil
	} else {
		must(a0.IsMap(), "must be map")
		m := a0.Map()
		o := NewValMap()
		for _, v := range args[1:] {
			key := v.String()
			if val, ok := m.Get(key); ok {
				o.AddMap(key, val)
			}
		}
		return o, nil
	}
}

func qSlice(
	info *IntrinsicInfo,
	_ *Evaluator,
	_ string,
	args []Val,
) (Val, error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}
	size := len(args)
	a0 := args[0]
	list := a0.List()
	start := 0
	end := list.Length()
	step := 1

	switch size {
	case 2:
		start = int(args[1].Int())
		break

	case 3:
		start = int(args[1].Int())
		end = int(args[2].Int())
		break
	default:
		start = int(args[1].Int())
		end = int(args[2].Int())
		step = int(args[3].Int())
		break
	}

	if end >= list.Length() {
		end = list.Length()
	}
	if start > end {
		start = end
	}

	o := NewValList()
	for ; start < end; start += step {
		o.AddList(list.At(start))
	}
	return o, nil
}

// helper function to allow map to work with map/reduce style coding
func addMapResult(
	m *Map,
	key string,
	val Val,
) {
	if m.Has(key) {
		v, ok := m.Get(key)
		must(ok, "value must be existed")
		must(v.IsList(), "must be list")
		v.AddList(val)
	} else {
		l := NewValList()
		l.AddList(val)
		m.Set(key, l)
	}
}

func qMap(info *IntrinsicInfo, eval *Evaluator, _ string, args []Val) (Val,
	error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}
	a0 := args[0]
	cb := args[1]
	closure := cb.Closure()

	if a0.IsList() {
		output := NewValMap()
		m := output.Map()

		l := a0.List()
		for k, v := range l.Data {
			cbArgs := []Val{
				NewValInt(k),
				v,
			}
			v, err := closure.Call(eval, cbArgs)
			if err != nil {
				return NewValNull(), err
			}
			if !v.IsPair() {
				return NewValNull(),
					fmt.Errorf("q::map's filter function must returns a pair")
			}

			if !v.Pair().First.IsString() {
				return NewValNull(),
					fmt.Errorf("q::map's filter function must returns " +
						"a pair with first to be string")
			}

			key := v.Pair().First.String()
			addMapResult(m, key, v.Pair().Second)
		}

		return output, nil
	}

	must(a0.IsMap(), "must be map")
	output := NewValMap()
	m := output.Map()
	a0m := a0.Map()
	var err error
	rErr := &err

	a0m.Foreach(
		func(key string, value Val) bool {
			cbArgs := []Val{
				NewValStr(key),
				value,
			}

			v, err := closure.Call(eval, cbArgs)
			if err != nil {
				*rErr = err
				return false
			}
			if !v.IsPair() {
				*rErr = fmt.Errorf(
					"q::map's filter function must returns a " +
						"pair")
				return false
			}
			if !v.Pair().First.IsString() {
				*rErr = fmt.Errorf(
					"q::map's filter function must returns " +
						"a pair with first to be string")
				return false
			}

			addMapResult(m, v.Pair().First.String(),
				v.Pair().Second)
			return true
		},
	)

	if err != nil {
		return NewValNull(), err
	} else {
		return output, nil
	}
}

// ---------------------------------------------------------------------------
// 3) Filter operations
func filterImpl(
	name string,
	info *IntrinsicInfo,
	eval *Evaluator,
	_ string,
	args []Val,
	should func(bool) bool,
) (Val, error) {
	a0 := args[0]
	fn := args[1].Closure()

	if a0.IsList() {
		o := NewValList()
		for k, v := range a0.List().Data {
			vv, err := fn.Call(
				eval,
				[]Val{
					NewValInt(k),
					v,
				},
			)
			if err != nil {
				return NewValNull(), err
			}
			if !vv.IsBool() {
				return NewValNull(),
					fmt.Errorf("%s callback function must return bool", name)
			}
			if should(vv.Bool()) {
				o.AddList(v)
			}
		}

		return o, nil
	}

	{
		must(a0.IsMap(), "must be map")

		o := NewValMap()
		var err error
		rErr := &err
		a0.Map().Foreach(
			func(k string, v Val) bool {
				vv, err := fn.Call(
					eval,
					[]Val{NewValStr(k), v},
				)
				if err != nil {
					*rErr = err
					return false
				}
				if !vv.IsBool() {
					*rErr =
						fmt.Errorf("%s callback function must return bool",
							name)
					return false
				}
				if should(vv.Bool()) {
					o.AddMap(k, v)
				}
				return true
			},
		)

		if err != nil {
			return NewValNull(), err
		} else {
			return o, nil
		}
	}
}

func qFilter(
	info *IntrinsicInfo,
	eval *Evaluator,
	name string,
	args []Val,
) (Val, error) {
	return filterImpl(
		"q::filter",
		info,
		eval,
		name,
		args,
		func(p bool) bool {
			return p
		},
	)
}

func qFilterNot(
	info *IntrinsicInfo,
	eval *Evaluator,
	name string,
	args []Val,
) (Val, error) {
	return filterImpl(
		"q::filter_not",
		info,
		eval,
		name,
		args,
		func(p bool) bool {
			return !p
		},
	)
}

// ---------------------------------------------------------------------------
// 4) aggregation
//
// All the aggregation function will try to use the first element inside of the
// array as the default type

const (
	isint  = 0
	isreal = 1
	isnone = 2
)

func firstNum(l *List) (int64, float64, int, int) {
	for idx, v := range l.Data {
		if v.IsInt() {
			return v.Int(), 0, isint, idx
		}
		if v.IsReal() {
			return 0, v.Real(), isreal, idx
		}
	}

	return 0, 0.0, isnone, -1
}

func qagg(
	l *List,
	args []Val,
	joiner func(int64, int64, float64, float64, int) (int64, float64, int),
) (int64, float64, int, error) {
	ival, rval, t, idx := firstNum(l)

	if t == isint {
		current := ival
		for _, v := range l.Data[idx+1:] {
			if v.IsInt() {
				vv := v.Int()
				a, _, c := joiner(current, vv, 0.0, 0.0, isint)
				must(c == isint, "must be int")
				current = a
			}
		}
		return current, 0.0, isint, nil
	}

	if t == isreal {
		current := rval
		for _, v := range l.Data[idx+1:] {
			if v.IsReal() {
				vv := v.Real()
				_, b, c := joiner(0, 0, current, vv, isreal)
				must(c == isreal, "must be real")
				current = b
			}
		}
		return 0, current, isreal, nil
	}

	return 0, 0.0, isnone, nil
}

func qaggret(
	i int64,
	r float64,
	t int,
	err error,
) (Val, error) {
	if err != nil {
		return NewValNull(), err
	}
	switch t {
	case isnone:
		return NewValNull(), nil
	case isint:
		return NewValInt64(i), nil
	default:
		return NewValReal(r), nil
	}
}

func qMax(
	info *IntrinsicInfo,
	eval *Evaluator,
	_ string,
	args []Val,
) (Val, error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}
	l := args[0].List()

	ival, rval, t, err := qagg(
		l,
		args,
		func(iprev int64, icur int64, rprev float64, rcur float64, t int) (int64, float64, int) {
			if t == isint {
				if iprev < icur {
					return icur, 0, isint
				} else {
					return iprev, 0, isint
				}
			} else {
				if rprev < rcur {
					return 0, rcur, isreal
				} else {
					return 0, rprev, isreal
				}
			}
		},
	)

	return qaggret(ival, rval, t, err)
}

func qMin(
	info *IntrinsicInfo,
	eval *Evaluator,
	_ string,
	args []Val,
) (Val, error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}
	l := args[0].List()

	ival, rval, t, err := qagg(
		l,
		args,
		func(iprev int64, icur int64, rprev float64, rcur float64, t int) (int64, float64, int) {
			if t == isint {
				if iprev > icur {
					return icur, 0, isint
				} else {
					return iprev, 0, isint
				}
			} else {
				if rprev > rcur {
					return 0, rcur, isreal
				} else {
					return 0, rprev, isreal
				}
			}
		},
	)

	return qaggret(ival, rval, t, err)
}

func qSum(
	info *IntrinsicInfo,
	eval *Evaluator,
	_ string,
	args []Val,
) (Val, error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}

	l := args[0].List()

	ival, rval, t, err := qagg(
		l, args,
		func(iprev int64, icur int64, rprev float64, rcur float64, t int) (int64, float64, int) {
			if t == isint {
				return iprev + icur, 0, isint
			} else {
				return 0, rprev + rcur, isreal
			}
		})

	return qaggret(ival, rval, t, err)
}

type qavginfo struct {
	count int
	rtt   float64
	itt   int64
}

func qAvg(
	info *IntrinsicInfo,
	eval *Evaluator,
	_ string,
	args []Val,
) (Val, error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}

	l := args[0].List()
	avg := &qavginfo{}

	_, _, t, err := qagg(
		l,
		args,
		func(_ int64, icur int64, _ float64, rcur float64, t int) (int64, float64, int) {
			avg.count++
			if t == isint {
				avg.itt += icur
				return 0, 0, isint
			} else {
				avg.rtt += rcur
				return 0, 0, isreal
			}
		},
	)

	if err != nil {
		return NewValNull(), err
	}

	switch t {
	case isnone:
		return NewValNull(), nil
	case isint:
		return NewValReal(float64(avg.itt) / float64(avg.count)), nil
	default:
		return NewValReal(avg.rtt / float64(avg.count)), nil
	}
}

func qCount(
	info *IntrinsicInfo,
	eval *Evaluator,
	_ string,
	args []Val,
) (Val, error) {
	if _, err := info.Check(args); err != nil {
		return NewValNull(), err
	}

	count := 1
	refCount := &count
	l := args[0].List()

	_, _, t, err := qagg(l, args,
		func(_ int64, _ int64, _ float64, _ float64, t int) (int64, float64, int) {
			*refCount++
			return 0, 0.0, t
		},
	)

	if err != nil {
		return NewValNull(), err
	} else {
		if t == isnone {
			return NewValInt(0), nil
		} else {
			return NewValInt(count), nil
		}
	}
}

func init() {
	addMF("q", "first", "", "{%l}{%p}", qFirst)
	addMF("q", "last", "", "{%l}{%p}", qLast)
	addMF("q", "rest", "", "{%l}{%p}", qRest)
	addMF("q", "select", "", "{%l}{%l%d*}{%m}{%m%s*}", qSelect)
	addMF("q", "slice", "", "{%l%d}{%l%d%d}{%l%d%d%d}", qSlice)
	addMF("q", "map", "", "{%l%c}{%m%c}", qMap)
	addMF("q", "filter", "", "{%l%c}{%m%c}", qFilter)
	addMF("q", "filter_not", "", "{%l%c}{%m%c}", qFilterNot)

	// aggregation
	addMF("q", "min", "", "{%l}", qMin)
	addMF("q", "max", "", "{%l}", qMax)
	addMF("q", "sum", "", "{%l}", qSum)
	addMF("q", "count", "", "{%l}", qCount)
	addMF("q", "avg", "", "{%l}", qAvg)
}
