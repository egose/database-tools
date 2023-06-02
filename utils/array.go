package utils

func CastToArray(value interface{}) []interface{} {
	// Check if the value is already an array or slice
	if arr, ok := value.([]interface{}); ok {
		return arr
	}

	// Create a new array and append the value
	return []interface{}{value}
}
