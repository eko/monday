package helper

func MergeMapString(map1 map[string]string, map2 map[string]string) map[string]string {
	var values = map[string]string{}

	if map1 != nil {
		values = map1
	}

	for key, value := range map2 {
		if _, ok := values[key]; !ok {
			values[key] = value
		}
	}

	return values
}
