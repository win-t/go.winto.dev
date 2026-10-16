package jdig

const strategicPatchKey = "$patch"

// Simulate kubernetes strategic array merge.
//
// only `replace`, `merge` and `delete` in `$patch` are supported.
func StrategicMerge(mergeKey string, src JArr) MergeHandler {
	return StrategicMergeFn(func(a, b JObj) bool {
		aValue, ok := AnyExists(a, mergeKey)
		if !ok {
			return false
		}
		bValue, ok := AnyExists(b, mergeKey)
		if !ok {
			return false
		}
		return aValue == bValue
	}, src)
}

// like [StrategicMerge] but instead of single merge key, pass the eq function
func StrategicMergeFn(eq func(JObj, JObj) bool, src JArr) MergeHandler {
	return MergeCallback(func(dstA any) any {
		dst, ok := dstA.(JArr)
		if !ok {
			return MergeWithoutResolve(dstA, src)
		}

		retval := make(JArr, 0, len(dst)+len(src))
		processed := make(map[int]struct{})
		for _, dstItem := range dst {
			dstItemM, ok := dstItem.(JObj)
			if !ok {
				retval = append(retval, dstItem)
				continue
			}

			skipDstItem := false
			for srcIdx, srcItem := range src {
				if _, ok := processed[srcIdx]; ok {
					continue
				}
				srcItemM, ok := srcItem.(JObj)
				if !ok {
					continue
				}
				if !eq(dstItemM, srcItemM) {
					continue
				}
				patchDirective, _ := srcItemM[strategicPatchKey].(string)
				delete(srcItemM, strategicPatchKey)
				switch patchDirective {
				case "replace":
					dstItem = srcItemM
				case "", "merge":
					dstItem = MergeWithoutResolve(dstItemM, srcItemM)
				case "delete":
					skipDstItem = true
				default:
					panic("jdig: patch directive \"" + patchDirective + "\" not supported")
				}
				processed[srcIdx] = struct{}{}
			}
			if !skipDstItem {
				retval = append(retval, dstItem)
			}
		}
		for srcIdx, srcItem := range src {
			if _, ok := processed[srcIdx]; ok {
				continue
			}
			srcItemM, ok := srcItem.(JObj)
			if !ok {
				retval = append(retval, srcItem)
				continue
			}
			delete(srcItemM, strategicPatchKey)
			retval = append(retval, srcItemM)
		}
		return retval
	})
}
