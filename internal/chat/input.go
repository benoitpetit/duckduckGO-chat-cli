package chat

import (
	"bufio"
	"os"
	"strings"

	"github.com/fatih/color"
)

func readSearchInput() string {
	color.Blue("\n🔍 Enter text to search: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
