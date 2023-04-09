package internal

import (
	"fmt"
	"os"
	"strings"
)

func DebugBlock(title any, value any) (n int, err error) {
	delim := strings.Repeat("-", 12)
	return fmt.Fprintf(os.Stdout, "%s %s %s\n%s\n", delim, title, delim, value)
}
