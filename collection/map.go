package collection

// MergeStringMaps merges k/v pairs from the src map into the dst.
func MergeStringMaps(src, dst map[string]string) map[string]string {
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
