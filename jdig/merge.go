package jdig

type MergeHandler interface{ internal(MergeHandler) }

type mergeHandlerBase struct{}

func (mergeHandlerBase) MarshalJSON() ([]byte, error) {
	panic("jdig: MergeHandler can't be marshaled, you are most likely use MergeHandler without calling Merge")
}

func (mergeHandlerBase) internal(MergeHandler) {}

type discardType struct{ mergeHandlerBase }

// Return placeholder value used to delete the key when [jdig.Merge] called.
func DiscardKey() MergeHandler { return discardType{} }

// This function must return the merged value, the defaultFn function can be used as fallback to do the default merge.
//
// If the dst is nil, do not return another MergeHandler other than DiscardKey.
type MergeCallbackFn func(dst any, defaultFn func(dst any, src any) any) any

func (cb MergeCallbackFn) do(dst any, defaultFn func(dst any, src any) any) any {
	v := cb(dst, defaultFn)
	if dst == nil {
		if _, ok := v.(mergeHandler); ok {
			panic("jdig: dst is nil, handler must not return another MergeHandler other than DiscardKey")
		}
	}
	return v
}

type mergeHandler struct {
	mergeHandlerBase
	cb MergeCallbackFn
}

// Return a MergeHandler with the callback function called when [jdig.Merge] called.
// The merge happen from right to left, so it is possible for the callback to return another MergeHandler.
func MergeCallback(cb MergeCallbackFn) MergeHandler { return mergeHandler{cb: cb} }

// Merge multiple JSON values into one, in place.
// If any of the object contains [jdig.MergeHandler] value, the merge handler will be used.
func Merge(objs ...any) any {
	return multiObjsMerge(objs, true)
}

// like [jdig.Merge] but the remaining MergeHandler will not be resolved.
//
// this is useful when you want to construct values that later will eventually be passed to [jdig.Merge].
func MergeWithoutResolve(objs ...any) any {
	return multiObjsMerge(objs, false)
}

func multiObjsMerge(objs []any, resolve bool) any {
	if len(objs) == 0 {
		return nil
	}
	var dst any
	if len(objs) > 1 {
		for i := len(objs) - 1; i >= 1; i-- {
			objs[i-1] = merge(objs[i-1], objs[i])
		}
	}
	dst = merge(nil, objs[0])
	if resolve {
		dst = resolveRemainingHandler(dst)
	}
	return dst
}

func resolveRemainingHandler(v any) any {
	switch v := v.(type) {
	case JObj:
		for k := range v {
			if _, ok := v[k].(discardType); ok {
				delete(v, k)
			} else {
				v[k] = resolveRemainingHandler(v[k])
			}
		}
		return v
	case JArr:
		for i := range v {
			v[i] = resolveRemainingHandler(v[i])
		}
		return v
	case mergeHandler:
		return resolveRemainingHandler(v.cb.do(nil, merge))
	case discardType:
		return nil
	default:
		return v
	}
}

func merge(dst, src any) any {
	switch src := src.(type) {
	case JObj:
		return mergeObj(dst, src)
	case JArr:
		return mergeArr(dst, src)
	case mergeHandler:
		return src.cb.do(dst, merge)
	default:
		return src
	}
}

func mergeObj(dst any, src JObj) any {
	dstM, ok := dst.(JObj)
	if !ok {
		return src
	}
	for k := range src {
		if v, ok := dstM[k]; ok {
			dstM[k] = merge(v, src[k])
		} else {
			dstM[k] = src[k]
		}
	}
	return dstM
}

func mergeArr(dst any, src JArr) any {
	dstA, ok := dst.(JArr)
	if !ok {
		return src
	}
	l := min(len(dstA), len(src))
	for i := range src[:l] {
		dstA[i] = merge(dstA[i], src[i])
	}
	dstA = append(dstA, src[l:]...)
	return dstA
}

// Keep the value in dst
func Keep() MergeHandler {
	return MergeCallback(func(dst any, defaultFn func(dst any, src any) any) any {
		return dst
	})
}

// Replace value in dst
func Replace(v any) MergeHandler {
	return MergeCallback(func(dst any, defaultFn func(dst any, src any) any) any {
		return v
	})
}
