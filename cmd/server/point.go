package main

import "github.com/gorilla/schema"

type point struct {
	X int `schema:"x,required"`
	Y int `schema:"y,required"`
}

func decodePoint(src map[string][]string) (point, error) {
	pointDecoder := schema.NewDecoder()
	pointDecoder.IgnoreUnknownKeys(true)
	var p point
	err := pointDecoder.Decode(&p, src)
	return p, err
}
