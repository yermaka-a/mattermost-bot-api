package models

type User struct {
	UserId string           `msgpack:"user_id"`
	Votes  map[string]int64 `msgpack:"votes"`
}
