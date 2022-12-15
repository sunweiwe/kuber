package maps

func LabelChanged(origin, target map[string]string) bool {
	if len(origin) == 0 {
		return true
	}

	for k, v := range target {
		t, exist := origin[k]
		if !exist {
			return true
		}
		if t != v {
			return true
		}
	}

	return false
}

func LabelDelete(origin, labels map[string]string) map[string]string {
	if len(origin) == 0 {
		return origin
	}
	for k := range labels {
		delete(origin, k)
	}

	return origin
}
