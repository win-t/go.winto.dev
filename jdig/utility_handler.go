package jdig

const strategicPatchKey = "$patch"

func objHaveKey(v any, keyName string) (string, JObj) {
	m, ok := v.(JObj)
	if !ok {
		return "", nil
	}
	keyValue, ok := m[keyName].(string)
	if !ok {
		return "", nil
	}
	return keyValue, m
}

// Simulate kubernetes strategic array merge.
//
// only `replace`, `merge` and `delete` in `$patch` are supported.
func StrategicMerge(mergeKey string, src JArr) MergeHandler {
	return MergeCallback(func(dstA any, defaultFn func(dst any, src any) any) any {
		dst, ok := dstA.(JArr)
		if !ok {
			return defaultFn(dstA, src)
		}

		keyedSrc := make(map[string]JObj)
		for _, item := range src {
			if keyValue, m := objHaveKey(item, mergeKey); keyValue != "" {
				keyedSrc[keyValue] = m
			}
		}

		processed := make(map[string]struct{})

		retval := make(JArr, 0, len(dst)+len(src))
		for _, item := range dst {
			keyValue, dstM := objHaveKey(item, mergeKey)
			if keyValue == "" {
				retval = append(retval, item)
				continue
			}
			srcM, ok := keyedSrc[keyValue]
			if !ok {
				retval = append(retval, item)
				continue
			}

			processed[keyValue] = struct{}{}
			patchDirective, _ := srcM[strategicPatchKey].(string)
			delete(srcM, strategicPatchKey)
			switch patchDirective {
			case "replace":
				retval = append(retval, srcM)
			case "", "merge":
				retval = append(retval, defaultFn(dstM, srcM))
			case "delete":
			default:
				panic("jdig: patch directive \"" + patchDirective + "\" not supported")
			}
		}
		for _, item := range src {
			if srcM, ok := item.(JObj); ok {
				delete(srcM, strategicPatchKey)
			}
			keyValue, srcM := objHaveKey(item, mergeKey)
			if keyValue == "" {
				retval = append(retval, item)
				continue
			}
			if _, processed := processed[keyValue]; !processed {
				retval = append(retval, srcM)
			}
		}

		return retval
	})
}

// NormalizeNilArrayAndMaps returns a MergeHandler that normalizes nil arrays and maps to empty ones.
func NormalizeNilArrayAndMaps(src any) MergeHandler {
	return MergeCallback(func(dst any, defaultFn func(dst any, src any) any) any {
		v := defaultFn(dst, src)
		return normalizeNilArrayAndMaps(v)
	})
}

func normalizeNilArrayAndMaps(v any) any {
	switch v := v.(type) {
	case JArr:
		if v == nil {
			return make(JArr, 0)
		}
		for i := range v {
			v[i] = normalizeNilArrayAndMaps(v[i])
		}
		return v
	case JObj:
		if v == nil {
			return make(JObj)
		}
		for k, vv := range v {
			v[k] = normalizeNilArrayAndMaps(vv)
		}
		return v
	default:
		return v
	}
}
