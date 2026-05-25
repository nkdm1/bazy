package misc

import (
	"fmt"
	"os"
	"runtime"
	"crypto/sha512"
)

func HashPassword(password string) string {
	sha := sha512.New()
	sha.Write([]byte(password))
	return string(sha.Sum(nil))
}

func Panicf(format string, v ...any) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = -1
	}
	userMsg := fmt.Sprintf(format, v...)
	fmt.Fprintf(os.Stderr, "panic: %s\n", userMsg)
	fmt.Fprintf(os.Stderr, "\t%s:%d\n", file, line)
	os.Exit(2)
}
