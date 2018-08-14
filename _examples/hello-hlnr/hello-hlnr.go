package main

import (
	"fmt"
	"net/http"
)

var myHTML = `
<html>
<center>
  <h1>Hello Heighliner!</h1>
</center>
</html>
`

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, myHTML)
}

func main() {
	http.HandleFunc("/", index)
	http.ListenAndServe(":8080", nil)
}
