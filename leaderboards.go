package main

type GameRecord struct {
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	MineCount int     `json:"mine_count"`
	Unique    bool    `json:"unique"`
	Playtime  float64 `json:"playtime"`
}

// NOTE: this is extremely inefficient
func compileGameRecords() ([]GameRecord, error) {
	var records []GameRecord
	keys, err := kvs.GetAllKeys()
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		var session GameSession
		if err = kvs.Get(key, &session); err != nil {
			return nil, err
		}
		if session.State.Won && !session.State.Dead {
			playtime := session.EndedAt.Sub(session.StartedAt).Seconds()
			record := GameRecord{
				Width:     session.State.Width,
				Height:    session.State.Height,
				MineCount: session.State.MineCount,
				Unique:    session.State.Unique,
				Playtime:  playtime,
			}
			records = append(records, record)
		}
	}
	return records, nil
}
