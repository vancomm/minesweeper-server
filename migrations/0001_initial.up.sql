CREATE TABLE IF NOT EXISTS player (
	player_id 		bigint 	GENERATED ALWAYS AS IDENTITY 
							PRIMARY KEY,
	username 		text 	UNIQUE NOT NULL,
	password_hash 	bytea 	NOT NULL,
	created_at 		timestamp with time zone
							DEFAULT now() 
							NOT NULL,
	updated_at 		timestamp with time zone 
							DEFAULT now() 
							NOT NULL
);

CREATE TABLE IF NOT EXISTS game_session (
	game_session_id	bigint 	GENERATED ALWAYS AS IDENTITY 
							PRIMARY KEY,
	player_id		bigint	REFERENCES player (player_id)
							NULL,
	width			integer	NOT NULL,
	height			integer	NOT NULL,
	mine_count		integer	NOT NULL,
	"unique"		boolean NOT NULL,
	dead			boolean NOT NULL,
	won				boolean NOT NULL,
	started_at		timestamp with time zone
							DEFAULT now()
							NOT NULL,
	ended_at		timestamp with time zone
							NULL,
	state			bytea	NOT NULL,
	created_at 		timestamp with time zone
							DEFAULT now() 
							NOT NULL,
	updated_at 		timestamp with time zone 
							DEFAULT now() 
							NOT NULL
);

CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = now();
	RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

CREATE OR REPLACE TRIGGER update_player_modtime
BEFORE UPDATE ON player
FOR EACH ROW EXECUTE FUNCTION update_modified_column();

CREATE OR REPLACE TRIGGER update_game_session_modtime
BEFORE UPDATE ON game_session
FOR EACH ROW EXECUTE FUNCTION update_modified_column();