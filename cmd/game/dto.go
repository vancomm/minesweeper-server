package main

import "github.com/gorilla/schema"

type NewGameDTO struct {
	Width     int  `schema:"width,required"`
	Height    int  `schema:"height,required"`
	MineCount int  `schema:"mine_count,required"`
	Unique    bool `schema:"unique,required"`
}

func ParseNewGameDTO(src map[string][]string) (NewGameDTO, error) {
	dec := schema.NewDecoder()
	dec.IgnoreUnknownKeys(true)
	var dto NewGameDTO
	err := dec.Decode(&dto, src)
	return dto, err
}
