// Browser-test host: uses the real production UI handler, without Garage.
package main

import (
	"khairul169/garage-webui/ui"
	"log"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()
	for _, base := range []string{"", "/console", "/tools/garage"} {
		os.Setenv("BASE_PATH", base)
		ui.ServeUI(mux)
	}
	log.Fatal(http.ListenAndServe("127.0.0.1:4173", mux))
}
