package main

import "github.com/gorilla/schema"

type NewGame struct {
	Width     int  `schema:"width,required"`
	Height    int  `schema:"height,required"`
	MineCount int  `schema:"mine_count,required"`
	Unique    bool `schema:"unique,required"`
}

func ParseNewGame(src map[string][]string) (NewGame, error) {
	dec := schema.NewDecoder()
	dec.IgnoreUnknownKeys(true)
	var dto NewGame
	err := dec.Decode(&dto, src)
	return dto, err
}
