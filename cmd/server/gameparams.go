package main

import "github.com/gorilla/schema"

type GameParams struct {
	Width     int  `schema:"width,required"`
	Height    int  `schema:"height,required"`
	MineCount int  `schema:"mine_count,required"`
	Unique    bool `schema:"unique,required"`
}

func decodeGameParams(src map[string][]string) (GameParams, error) {
	dec := schema.NewDecoder()
	dec.IgnoreUnknownKeys(true)
	var dto GameParams
	err := dec.Decode(&dto, src)
	return dto, err
}
