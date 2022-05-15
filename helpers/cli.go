package helpers

import (
	"bufio"
	"os"
)

func ScanInputToString() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}
