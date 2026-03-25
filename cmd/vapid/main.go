// Command vapid generates a VAPID key pair.
// With -update-env, appends the keys to .env.
package main

import (
	"flag"
	"fmt"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	updateEnv := flag.Bool("update-env", false, "append generated keys to .env")
	flag.Parse()

	privKey, pubKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatalf("generate VAPID keys: %v", err)
	}

	if *updateEnv {
		writeToEnv(pubKey, privKey)
		return
	}

	fmt.Printf("VAPID_PUBLIC_KEY=%s\n", pubKey)
	fmt.Printf("VAPID_PRIVATE_KEY=%s\n", privKey)
}
