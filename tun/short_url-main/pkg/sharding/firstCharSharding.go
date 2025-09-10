package sharding

import (
	"errors"
	"fmt"
)

func CustomShardingAlgorithm(source any) (string, error) {
	key, ok := source.(string)
	if !ok {
		return "", errors.New("invalid short_url")
	}
	firstChar := string(key[0])
	return fmt.Sprintf("_%s", firstChar), nil
}
