package models

import "github.com/tarantool/go-tarantool/v2"

type TarantoolStorage struct {
	Conn *tarantool.Connection
}
