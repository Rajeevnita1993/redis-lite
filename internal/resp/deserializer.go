package resp

import (
	"errors"
	"strconv"
	"strings"
)

func DeserializeSimpleString(input string) (string, error) {
	if len(input) < 3 || input[0] != '+' {
		return "", errors.New("invalid simple string format")
	}
	return strings.TrimSpace(input[1 : len(input)-2]), nil
}

func DeserializeError(input string) (string, error) {
	if len(input) < 3 || input[0] != '-' {
		return "", errors.New("invalid error message format")
	}
	return strings.TrimSpace(input[1 : len(input)-2]), nil
}

func DeserializeInteger(input string) (int, error) {
	if len(input) < 3 || input[0] != ':' {
		return 0, errors.New("invalid integer format")
	}
	value, err := strconv.Atoi(input[1 : len(input)-2])
	if err != nil {
		return 0, err
	}
	return value, nil
}

func DeserializeBulkString(input string) (string, error) {
	if len(input) < 5 || input[0] != '$' {
		return "", errors.New("invalid bulk string format")
	}
	length, err := strconv.Atoi(input[1 : len(input)-2])
	if err != nil {
		return "", err
	}
	if length == -1 {
		return "", nil // Null bulk string
	}
	start := strings.Index(input, "\r\n") + 2
	return input[start : start+length], nil
}

func DeserializeArray(input string) ([]string, error) {
	if len(input) < 3 || input[0] != '*' {
		return nil, errors.New("invalid array format")
	}
	count, err := strconv.Atoi(input[1 : len(input)-2])
	if err != nil {
		return nil, err
	}
	if count == -1 {
		return nil, nil // Null array
	}
	var values []string
	start := 0
	for i := 0; i < count; i++ {
		bulkString, err := DeserializeBulkString(input[start:])
		if err != nil {
			return nil, err
		}
		values = append(values, bulkString)
		start += len(bulkString) + 5 // +5 for "$\r\n"
	}
	return values, nil
}
