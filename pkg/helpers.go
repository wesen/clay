package pkg

func CastList[To any, From any](list []From) ([]To, bool) {
	ret := []To{}

	for _, item := range list {
		var v interface{} = item
		casted, ok := v.(To)
		if !ok {
			return ret, false
		}

		ret = append(ret, casted)
	}

	return ret, true
}
