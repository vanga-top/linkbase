package utils

import (
	"encoding/json"
	"fmt"
)

// JSONToMap parse the jsonic index parameters to map
func JSONToMap(mStr string) (map[string]string, error) {
	buffer := make(map[string]any)
	err := json.Unmarshal([]byte(mStr), &buffer)
	if err != nil {
		return nil, fmt.Errorf("unmarshal params failed, %w", err)
	}
	ret := make(map[string]string)
	for key, value := range buffer {
		valueStr := fmt.Sprintf("%v", value)
		ret[key] = valueStr
	}
	return ret, nil
}

func MapToJSON(m map[string]string) []byte {
	// error won't happen here.
	bs, _ := json.Marshal(m)
	return bs
}
