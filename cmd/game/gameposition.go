package main

import "github.com/gorilla/schema"

type Position struct {
	X int `schema:"x,required"`
	Y int `schema:"y,required"`
}

func ParsePosition(src map[string][]string) (Position, error) {
	positionDecoder := schema.NewDecoder()
	positionDecoder.IgnoreUnknownKeys(true)
	var pos Position
	err := positionDecoder.Decode(&pos, src)
	return pos, err
}
