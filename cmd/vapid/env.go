package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

const envFile = ".env"

func writeToEnv(pubKey, privKey string) {
	if hasVAPIDKeys(envFile) {
		log.Fatal("VAPID keys already present in .env — remove them first to regenerate")
	}

	f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("open %s: %v", envFile, err)
	}
	defer f.Close()

	if _, err := f.WriteString("\nVAPID_PUBLIC_KEY=" + pubKey + "\nVAPID_PRIVATE_KEY=" + privKey + "\n"); err != nil {
		log.Fatalf("write %s: %v", envFile, err)
	}

	log.Printf("VAPID keys written to %s", envFile)
}

func hasVAPIDKeys(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VAPID_PUBLIC_KEY=") || strings.HasPrefix(line, "VAPID_PRIVATE_KEY=") {
			return true
		}
	}
	return false
}
