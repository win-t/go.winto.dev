package sqlbuilder

type Query struct {
	query  []byte
	scans  []any
	values []any
}

func Build(args ...any) *Query {
	q := &Query{}
	for _, a := range args {
		switch a := a.(type) {
		case string:
			q.query = append(q.query, a...)
		case ScanArg:
			q.query = append(q.query, a.expr...)
			q.scans = append(q.scans, a.target)
		case ValueArg:
			q.query = append(q.query, a.placeHolder...)
			q.values = append(q.values, a.value)
		default:
			panic("sqlbuilder: invalid value")
		}
	}
	return q
}

func (q *Query) Query() string {
	return string(q.query)
}

func (q *Query) Scans() []any {
	return q.scans
}

func (q *Query) Values() []any {
	return q.values
}

type ScanArg struct {
	expr   string
	target any
}

func Scan(expr string, target any) ScanArg {
	return ScanArg{expr, target}
}

type ValueArg struct {
	placeHolder string
	value       any
}

func Value(placeHolder string, value any) ValueArg {
	return ValueArg{placeHolder, value}
}
