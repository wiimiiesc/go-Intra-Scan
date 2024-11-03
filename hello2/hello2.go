package hello2

import (
	"fmt"
	"os/exec"
)

func main() {
	out, err := exec.Command("ping", "-w", "1000", "-n", "1", "192.168.0.1").Output()
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	fmt.Printf("out: %v\n", out)
}
