package main

import (
	"fmt"
	"strconv"
	"strings"
)

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func remove(s []int, i int) []int {
	if len(s) < i {
		return s
	}

	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func stripPort(s string) (int, error) {
	port, err := strconv.Atoi(s[strings.Index(s, ":")+1:])
	if err != nil {
		return -1, fmt.Errorf("Unable to strip server off to get port : %+v", err)
	}
	return port, nil
}
