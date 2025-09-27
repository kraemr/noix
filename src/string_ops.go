package main

import "strings"

func removeLeading(str string, cut string) string {
	return strings.TrimLeft(str, cut)
}

func removeTrailing(str string, cut string) string {
	if strings.HasSuffix(str, cut) {
		return str[:len(str)-len(cut)]
	}
	return str
}

func removeDuplicates(str string, dupe string, replace string) string {
	return strings.ReplaceAll(str, dupe, replace)
}
