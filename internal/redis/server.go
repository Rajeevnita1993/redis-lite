package redis

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Rajeevnita1993/redis-server/internal/resp"
)

type RedisServer struct {
	data       map[string]map[string]string // key ->[value, expiry]
	expiryTime map[string]time.Time
	dbFile     string // Database file path
}

func NewRedisServer(dbFile string) *RedisServer {

	server := &RedisServer{
		data:       make(map[string]map[string]string),
		expiryTime: make(map[string]time.Time),
		dbFile:     dbFile,
	}
	// Load database state from file
	server.loadFromDisk()
	return server

}

func (server *RedisServer) ListenAndServe(addr string) error {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	defer listener.Close()

	// Start a Goroutine to periodically check for expired keys
	go server.checkExpiry()

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go server.HandleConnection(conn)

	}

}

func (server *RedisServer) HandleConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		request := scanner.Text()
		response := server.processRequest(request)
		conn.Write([]byte(response))
	}
}

func (server *RedisServer) processRequest(request string) string {
	parts := strings.Fields(request)

	if len(parts) == 0 {
		return resp.SerializeError("invalid command")
	}

	command := strings.ToUpper(parts[0])
	switch command {
	case "PING":
		return resp.SerializeSimpleString("PONG")
	case "ECHO":
		if len(parts) < 2 {
			return resp.SerializeError("Wrong numbers of arguments for 'ECHO' command")
		}
		echoedString := strings.Join(parts[1:], "")
		return resp.SerializeBulkString(echoedString)
	case "SET":
		return server.setKey(parts)
		// if len(parts) < 3 {
		// 	return resp.SerializeError("wrong number of argument for 'SET' command")
		// }

		// key := parts[1]
		// value := strings.Join(parts[2:], " ")
		// server.data[key] = value
		// return resp.SerializeSimpleString("OK")
	case "GET":
		return server.getKey(parts)
		// if len(parts) < 2 {
		// 	return resp.SerializeError("wrong number of arguments for 'GET' command")

		// }
		// key := parts[1]
		// value, ok := server.data[key]
		// if !ok {
		// 	return resp.SerializeNullBulkString()
		// }
		// return resp.SerializeBulkString(value)
	case "EXISTS":
		return server.existsKey(parts)
	case "DEL":
		return server.delKey(parts)
	case "INCR":
		return server.incrKey(parts)
	case "DECR":
		return server.decrKey(parts)
	case "LPUSH":
		return server.lPushList(parts)
	case "RPUSH":
		return server.rPushList(parts)
	case "SAVE":
		return server.saveToDisk()
	default:
		return resp.SerializeError("unknown command '" + command + "'")
	}
}

func (server *RedisServer) setKey(parts []string) string {
	if len(parts) < 3 {
		return resp.SerializeError("Wrong number of arguments for 'SET' command")
	}
	key := parts[1]
	value := strings.Join(parts[2:], " ")

	// check for expiry options
	expiry := time.Time{}

	for i := 2; i < len(parts); i++ {
		if strings.ToUpper(parts[i]) == "EX" && i+1 < len(parts) {
			ex, err := strconv.Atoi(parts[i+1])
			if err == nil {
				expiry = time.Now().Add(time.Duration(ex) * time.Second)
				i++
			}
		} else if strings.ToUpper(parts[i]) == "PX" && i+1 < len(parts) {
			px, err := strconv.Atoi(parts[i+1])
			if err == nil {
				expiry = time.Now().Add(time.Duration(px) * time.Millisecond)
				i++
			}
		} else if strings.ToUpper(parts[i]) == "EXAT" && i+1 < len(parts) {
			exat, err := strconv.ParseInt(parts[i+1], 10, 64)
			if err == nil {
				expiry = time.Unix(exat, 0)
				i++
			}
		} else if strings.ToUpper(parts[i]) == "PXAT" && i+1 < len(parts) {
			pxat, err := strconv.ParseInt(parts[i+1], 10, 64)
			if err == nil {
				expiry = time.Unix(0, pxat*int64(time.Millisecond))
				i++
			}
		}
	}

	server.data[key] = map[string]string{"value": value, "expiry": expiry.Format(time.RFC3339)}
	if !expiry.IsZero() {
		server.expiryTime[key] = expiry
	}

	return resp.SerializeSimpleString("OK")

}

func (server *RedisServer) getKey(parts []string) string {
	if len(parts) < 2 {
		return resp.SerializeError("wrong number of arguments for 'GET' command")
	}
	key := parts[1]

	// Check if the key exists
	data, ok := server.data[key]
	if !ok {
		return resp.SerializeNullBulkString()
	}

	// Check if the key has expired
	expiry, ok := server.expiryTime[key]
	if ok && time.Now().After(expiry) {
		delete(server.data, key)
		delete(server.expiryTime, key)
		return resp.SerializeNullBulkString()

	}

	return resp.SerializeBulkString(data["value"])
}

func (server *RedisServer) existsKey(parts []string) string {
	if len(parts) != 2 {
		return resp.SerializeError("wrong number of arguments for 'EXISTS' command")
	}

	key := parts[1]
	if _, ok := server.data[key]; ok {
		return resp.SerializeInteger(1)
	}
	return resp.SerializeInteger(0)
}

func (server *RedisServer) delKey(parts []string) string {
	if len(parts) < 2 {
		return resp.SerializeError("wrong number of arguments for 'DEL' command")
	}
	count := 0
	for _, key := range parts[1:] {
		if _, ok := server.data[key]; ok {
			delete(server.data, key)
			delete(server.expiryTime, key)
			count++
		}
	}
	return resp.SerializeInteger(count)
}

func (server *RedisServer) incrKey(parts []string) string {
	if len(parts) != 2 {
		return resp.SerializeError("wrong number of arguments for 'INCR' command")
	}
	key := parts[1]
	if val, ok := server.data[key]; ok {
		num, err := strconv.Atoi(val["value"])
		if err != nil {
			return resp.SerializeError("value is not an integer")
		}
		// Increment the integer value
		num++
		// Update the value in the map
		server.data[key]["value"] = strconv.Itoa(num)
		return resp.SerializeInteger(num)
	}
	// If key does not exist, set it to 1
	server.data[key] = map[string]string{"value": "1"}
	return resp.SerializeInteger(1)
}

func (server *RedisServer) decrKey(parts []string) string {
	if len(parts) != 2 {
		return resp.SerializeError("wrong number of arguments for 'DECR' command")
	}
	key := parts[1]
	if val, ok := server.data[key]; ok {
		num, err := strconv.Atoi(val["value"])
		if err != nil {
			return resp.SerializeError("value is not an integer")
		}
		// Decrement the integer value
		num--
		// Update the value in the map
		server.data[key]["value"] = strconv.Itoa(num)
		return resp.SerializeInteger(num)
	}
	// If key does not exist, set it to 1
	server.data[key] = map[string]string{"value": "-1"}
	return resp.SerializeInteger(-1)
}

func (server *RedisServer) lPushList(parts []string) string {
	if len(parts) < 3 {
		return resp.SerializeError("wrong number of arguments for 'LPUSH' command")
	}
	key := parts[1]
	values := parts[2:]

	// Check if the key exists
	if _, ok := server.data[key]; !ok {
		// If the key doesn't exist, create a new list
		server.data[key] = make(map[string]string)
	}

	// Append values to the list
	for i := len(values) - 1; i >= 0; i-- {
		server.data[key][strconv.Itoa(len(server.data[key]))] = values[i]
	}

	// Return the length of the list after pushing the values
	return resp.SerializeInteger(len(server.data[key]))
}

func (server *RedisServer) rPushList(parts []string) string {
	if len(parts) < 3 {
		return resp.SerializeError("wrong number of arguments for 'RPUSH' command")
	}
	key := parts[1]
	values := parts[2:]

	// Check if the key exists
	if _, ok := server.data[key]; !ok {
		// If the key doesn't exist, create a new list
		server.data[key] = make(map[string]string)
	}

	// Append values to the list
	for _, value := range values {
		server.data[key][strconv.Itoa(len(server.data[key]))] = value
	}

	// Return the length of the list after pushing the values
	return resp.SerializeInteger(len(server.data[key]))
}

func (server *RedisServer) saveToDisk() string {
	// Serialize data to JSON
	dataBytes, err := json.Marshal(server.data)
	if err != nil {
		return resp.SerializeError("failed to serialize data")
	}

	// Write data to file
	file, err := os.Create(server.dbFile)
	if err != nil {
		return resp.SerializeError("failed to create database file")
	}
	defer file.Close()

	_, err = file.Write(dataBytes)
	if err != nil {
		return resp.SerializeError("failed to write data to file")
	}

	return resp.SerializeSimpleString("OK")
}

func (server *RedisServer) loadFromDisk() {
	// Open database file
	file, err := os.Open(server.dbFile)
	if err != nil {
		fmt.Println("No existing database file found.")
		return
	}
	defer file.Close()

	// Decode JSON data
	var data map[string]map[string]string
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		fmt.Println("Error decoding database file:", err)
		return
	}

	// Load data into server
	server.data = data
}

func (server *RedisServer) checkExpiry() {
	for {
		for key, expiry := range server.expiryTime {
			if time.Now().After(expiry) {
				delete(server.data, key)
				delete(server.expiryTime, key)
			}
		}
		time.Sleep(1 * time.Second)
	}
}
