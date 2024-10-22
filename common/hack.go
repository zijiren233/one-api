package common

import "unsafe"

// The change of bytes will cause the change of string synchronously
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// If string is readonly, modifying bytes will cause panic
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
