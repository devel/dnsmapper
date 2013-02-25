package main

import (
	"github.com/vmihailenco/redis"
)

var Redis *redis.Client

func redisConnect() {
	password := "" // no password set
	db := -1       // use default DB
	Redis = redis.NewTCPClient("localhost:6379", password, int64(db))
	// defer client.Close()
}
