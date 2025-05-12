package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", handleRequest)

	port := "3000"
	log.Println("Mock Earth Level started on port :" + port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))

}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	data, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	log.Println(string(data))
}
