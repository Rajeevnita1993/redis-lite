package main

import (
	"github.com/Rajeevnita1993/redis-server/internal/redis"
)

func main() {

	server := redis.NewRedisServer("diskfile.txt")

	server.ListenAndServe(":6379")

}
