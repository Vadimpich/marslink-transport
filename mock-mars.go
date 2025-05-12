package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", handleRequest2)

	port := "3010"
	log.Println("Mock Mars Level started on port :" + port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))

}

func handleRequest2(w http.ResponseWriter, r *http.Request) {
	data, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	log.Println(string(data))
}
