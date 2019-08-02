//Package logger does logging stuff
package logger

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var promAddr string

//SetPromAddr sets the prometheus address
func SetPromAddr(addr string) {
	promAddr = addr
}

//LogError will create a string containing all necessary error information
func LogError(fields map[string]string, level string) (s string) {

	result := "[" + level + "] " + time.Now().Format(time.RFC3339Nano) + " address=" + promAddr

	for k, v := range fields {
		if strings.Contains(k, " ") {
			k = `"` + k + `"`
		}
		if strings.Contains(v, " ") {
			v = `"` + v + `"`
		}
		result += " " + k + "=" + v
	}

	return "\n" + result

}

//PrintLog prints the log
func PrintLog(errors string, f *os.File) {
	fmt.Fprintf(f, errors)
	f.Close()
}
