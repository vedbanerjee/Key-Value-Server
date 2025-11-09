package main

import (
	"bufio"
	"io"
    "fmt"
    "net/http"
	"encoding/json"
	"bytes"
	"time"
	"strconv"
)

type keyValue struct{
	Key int
	Value string
}

var n int = 15

func putKeyValue(key int, value string) {
	m := keyValue{Key : key, Value : value}
	jsonData, err := json.Marshal(m)
	if err != nil {
        panic(err)
    }
	url := "http://localhost:8080/put"
	contentType:="application/json"

	resp, err := http.Post(url, contentType, bytes.NewReader(jsonData))
	if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
	    panic(err)
	}
	fmt.Println("Server Response Body:", string(response))
}

func getValue(key int) {
	m:=keyValue{Key:key, Value : ""}
	jsonData, err:= json.Marshal(m)
	if err!=nil{
		panic(err)
	}

	url:="http://localhost:8080/get"
	contentType:="application/json"

	resp, err:= http.Post(url, contentType, bytes.NewReader(jsonData))
	if err!=nil{
		panic(err)
	}

	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err!=nil{
		panic(err)
	}

	fmt.Println("Server response: ", string(response))
}

func deleteKey(key int) {
	m := keyValue{Key:key, Value:""}
	jsonData, err := json.Marshal(m)
	if err!=nil{
		panic(err)
	}

	url := "http://localhost:8080/delete"
	contentType :="application/json"

	resp, err := http.Post(url, contentType, bytes.NewReader(jsonData))
	if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

	responseBody,err := io.ReadAll(resp.Body)
	if err!=nil{
		panic(err)
	}

	fmt.Println("Server response: ",string(responseBody))
}

func main() {
	
	
    resp, err := http.Get("http://localhost:8080/hello")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

	fmt.Printf("Response status: %d\n", resp.Status)

	//reading hello
	scanner :=bufio.NewScanner(resp.Body)
	scanner.Scan()
	fmt.Println(scanner.Text())
	
	start := time.Now()

	//put requests 
	for i := 0; i < n; i++ {
		key:=i
		startTime := time.Now()
		value := strconv.Itoa(key)
		putKeyValue(key, value)
		fmt.Println(time.Since(startTime))
	}

	//get requests
	for i := 0; i < n; i++ {
		key:=n-i-1
		startTime:=time.Now()
		getValue(key)
		fmt.Println(time.Since(startTime))
	}

	//delete requests
	for i := 0; i < n; i++ {
		key:=i
		startTime:=time.Now()
		deleteKey(key)
		fmt.Println(time.Since(startTime))
	}
	fmt.Println()

	fmt.Println(time.Since(start))
}
