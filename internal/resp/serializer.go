package resp

import (
	"fmt"
	"strings"
)

func SerializeSimpleString(value string) string {
	return fmt.Sprintf("+%s\r\n", value)
}

func SerializeError(errorMessage string) string {
	return fmt.Sprintf("-%s\r\n", errorMessage)
}

func SerializeInteger(value int) string {
	return fmt.Sprintf(":%d\r\n", value)
}

func SerializeBulkString(value string) string {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return "$0\r\n\r\n"
	}
	return fmt.Sprintf("$%d\r\n%s\r\n", len(trimmedValue), trimmedValue)
}

func SerializeArray(values []string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("*%d\r\n", len(values)))
	for _, v := range values {
		builder.WriteString(SerializeBulkString(v))
	}
	return builder.String()
}

func SerializeNullBulkString() string {
	return "$-1\r\n"
}
