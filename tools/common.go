package tools

import (
	"os"
	"strconv"
	"strings"
)

func getOdpEnv() string {
	env := ""
	env1 := os.Getenv("ODP_ENV")
	env2 := os.Getenv("env")
	if env1 != "" {
		env = env1
	} else {
		env = env2
	}

	return env
}

func ParseStringToInterface(keyword string) []interface{} {
	var tmp []interface{}
	if keyword == "" {
		return tmp
	}
	s := strings.Split(keyword, ",")
	for _, v := range s {
		tmp = append(tmp, v)
	}
	return tmp
}

func InterfaceToString(it interface{}) string {
	var l interface{}

	switch it.(type) {
	case []interface{}:
		l = it.([]interface{})[0]
	default:
		l = it
	}

	switch l.(type) {
	case int:
		return strconv.Itoa(l.(int))
	case float64:
		return strconv.Itoa(int(l.(float64)))
	case string:
		return l.(string)
	}
	return ""
}

func InterfaceToInt(it interface{}) int {
	var l interface{}

	switch it.(type) {
	case []interface{}:
		l = it.([]interface{})[0]
	default:
		l = it
	}

	switch l.(type) {
	case int:
		return l.(int)
	case float64:
		return int(l.(float64))
	case string:
		tmp, err := strconv.Atoi(l.(string))
		if err != nil {
			return 0
		} else {
			return tmp
		}
	}
	return 0
}

func sortItems(items []map[string]interface{}, field string) {
	length := len(items)
	for i := 0; i < length-1; i++ {
		for j := i + 1; j < length; j++ {
			a := InterfaceToInt(items[i][field])
			b := InterfaceToInt(items[j][field])
			if a < b {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
