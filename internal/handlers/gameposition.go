package handlers

import "github.com/gorilla/schema"

type Point struct {
	X int `schema:"x,required"`
	Y int `schema:"y,required"`
}

func ParsePoint(src map[string][]string) (Point, error) {
	pointDecoder := schema.NewDecoder()
	pointDecoder.IgnoreUnknownKeys(true)
	var p Point
	err := pointDecoder.Decode(&p, src)
	return p, err
}
