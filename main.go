package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/murakmii/gotermda/shell"

	"github.com/murakmii/gotermda/pty"
)

const (
	shellPath = "/bin/bash"
)

func main() {
	logger := log.New(os.Stdout, "[gotermda] ", log.LstdFlags)

	master, slave, err := pty.Open()
	if err != nil {
		logger.Fatalf("failed to open pty: %s", err)
	}

	logger.Printf("opened pty: %s", slave.Name())

	sh, err := shell.Start(shellPath, slave)
	if err != nil {
		logger.Fatalf("failed to start shell: %s", err)
	}

	logger.Printf("started shell(%s PID: %d)", shellPath, sh.Pid())

	go func() {
		reader := bufio.NewReader(master)
		for {
			r, _, err := reader.ReadRune()
			if err != nil {
				logger.Printf("failed to read rune from master: %s", err)
			}

			fmt.Printf(string(r))
		}
	}()

	sh.Wait()
}
