package mongodb

import "go.mongodb.org/mongo-driver/mongo/options"

type FindOption func(*findOptions)

var (
	Sort   = func(s interface{}) FindOption { return func(f *findOptions) { f.sort = s } }
	Offset = func(o int) FindOption { return func(f *findOptions) { f.skip = int64(o) } }
	Limit  = func(l int) FindOption {
		return func(f *findOptions) {
			f.limit = int64(l)
			if l == 0 {
				f.obeyZeroLimit = true
			}
		}
	}
	Projection = func(p interface{}) FindOption { return func(f *findOptions) { f.projection = p } }

	IgnoreZeroLimit = func() FindOption { return func(f *findOptions) { f.obeyZeroLimit = false } }
)

type findOptions struct {
	limit         int64
	skip          int64
	sort          interface{}
	projection    interface{}
	obeyZeroLimit bool
}

func newFindOptions(opts ...FindOption) *findOptions {
	f := &findOptions{}
	for _, o := range opts {
		o(f)
	}

	return f
}

func (fo findOptions) asDriverFindOption() *options.FindOptions {
	return options.Find().SetSort(fo.sort).SetSkip(fo.skip).SetLimit(fo.limit).SetProjection(fo.projection)
}

func (fo findOptions) asDriverFindOneOption() *options.FindOneOptions {
	return options.FindOne().SetSort(fo.sort).SetSkip(fo.skip).SetProjection(fo.projection)
}

func (fo findOptions) asDriverCountOption() *options.CountOptions {
	co := options.Count().SetSkip(fo.skip)
	if fo.limit > 0 {
		co.SetLimit(fo.limit)
	}

	return co
}
