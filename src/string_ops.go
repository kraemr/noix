package main

import "strings"

func removeLeading(str string, cut string) string {
	return strings.TrimLeft(str, cut)
}

func removeTrailing(str string, cut string) string {
	return strings.TrimRight(str, cut)
}

func removeDuplicates(str string, dupe string, replace string) string {
	return strings.ReplaceAll(str, dupe, replace)
}
