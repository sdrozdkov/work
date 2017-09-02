package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	sentinel "github.com/FZambia/go-sentinel"
	"github.com/garyburd/redigo/redis"
	"github.com/gocraft/work/webui"
)

var (
	redisHostPort    = flag.String("redis", ":6379", "redis hostport")
	sentinelHostPort = flag.String("redis-sentinel", ":26379", "redis-sentinel hostport")
	redisDatabase    = flag.String("database", "0", "redis database")
	redisNamespace   = flag.String("ns", "work", "redis namespace")
	webHostPort      = flag.String("listen", ":5040", "hostport to listen for HTTP JSON API")
)

func main() {
	flag.Parse()

	fmt.Println("Starting workwebui:")
	fmt.Println("redis = ", *redisHostPort)
	fmt.Println("sentinel = ", *sentinelHostPort)
	fmt.Println("database = ", *redisDatabase)
	fmt.Println("namespace = ", *redisNamespace)
	fmt.Println("listen = ", *webHostPort)

	database, err := strconv.Atoi(*redisDatabase)
	if err != nil {
		fmt.Printf("Error: %v is not a valid database value", *redisDatabase)
		return
	}

	redisMaster := newSentinelPool(*sentinelHostPort, database)
	pool := newPool(*redisMaster, database)

	server := webui.NewServer(*redisNamespace, pool, *webHostPort)
	server.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	<-c

	server.Stop()

	fmt.Println("\nQuitting...")
}

func newPool(addr string, database int) *redis.Pool {
	return &redis.Pool{
		MaxActive:   3,
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(addr, redis.DialDatabase(database))
		},
		Wait: true,
	}
}

func newSentinelPool(addr string, database int) *string {
	sntnl := &sentinel.Sentinel{
		Addrs:      []string{addr},
		MasterName: "mymaster",
		Dial: func(addr string) (redis.Conn, error) {
			timeout := 500 * time.Millisecond
			c, err := redis.DialTimeout("tcp", addr, timeout, timeout, timeout)
			if err != nil {
				return nil, err
			}
			return c, nil
		},
	}
	master, _ := sntnl.MasterAddr()
	return &master

}
