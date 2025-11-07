package main

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "decsproject/cache" 
)

type keyValue struct {
    Key   int    `json:"key"`
    Value string `json:"value"`
}

var capacityCache int = 10
var db *sql.DB
var lruCache *cache.LRUCache

func hello(w http.ResponseWriter, req *http.Request) {
    fmt.Fprintf(w, "hello")
}

func put(w http.ResponseWriter, req *http.Request) {
    data, err := io.ReadAll(req.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusInternalServerError)
        return
    }

    var receivedData keyValue
    err = json.Unmarshal(data, &receivedData)
    if err != nil {
        http.Error(w, "Invalid JSON format", http.StatusBadRequest)
        return
    }

    sqlQuery := `
        INSERT INTO KeyValue (id, value)
        VALUES (?, ?)
        ON DUPLICATE KEY UPDATE value = ?`

    _, err = db.Exec(sqlQuery, receivedData.Key, receivedData.Value, receivedData.Value)
    if err != nil {
        log.Printf("Database EXEC error (put/upsert) for key %d: %v", receivedData.Key, err)
        http.Error(w, "Failed to execute upsert query", http.StatusInternalServerError)
        return
    }

    keyStr := fmt.Sprintf("%d", receivedData.Key)
    lruCache.Put(keyStr, receivedData.Value)

    fmt.Fprintf(w, "Key %d value %s created/updated", receivedData.Key, receivedData.Value)
}

func get(w http.ResponseWriter, req *http.Request) {
    data, err := io.ReadAll(req.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusInternalServerError)
        return
    }

    var toSend keyValue
    err = json.Unmarshal(data, &toSend)
    if err != nil {
        http.Error(w, "Invalid JSON format", http.StatusBadRequest)
        return
    }

    keyStr := fmt.Sprintf("%d", toSend.Key)

    if value, found := lruCache.Get(keyStr); found {
        fmt.Fprintf(w, "The value for key %d is %s (from cache)", toSend.Key, value)
        return
    }

    var value string
    sqlQuery := "SELECT value FROM KeyValue WHERE id = ?"
    err = db.QueryRow(sqlQuery, toSend.Key).Scan(&value)
    if err == sql.ErrNoRows {
        w.WriteHeader(http.StatusNotFound)
        fmt.Fprintf(w, "Key %d is not present", toSend.Key)
        return
    }
    if err != nil {
        log.Printf("Database QueryRow error (get) for key %d: %v", toSend.Key, err)
        http.Error(w, "Failed to execute query", http.StatusInternalServerError)
        return
    }

    lruCache.Put(keyStr, value)

    fmt.Fprintf(w, "The value for key %d is %s (from DB)", toSend.Key, value)
}

func del(w http.ResponseWriter, req *http.Request) {
    resp, err := io.ReadAll(req.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusInternalServerError)
        return
    }

    var toDelete keyValue
    err = json.Unmarshal(resp, &toDelete)
    if err != nil {
        http.Error(w, "Invalid JSON format", http.StatusBadRequest)
        return
    }

    sqlQuery := "DELETE FROM KeyValue WHERE id = ?"
    result, err := db.Exec(sqlQuery, toDelete.Key)
    if err != nil {
        http.Error(w, "Failed to execute delete query", http.StatusInternalServerError)
        return
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected > 0 {
        keyStr := fmt.Sprintf("%d", toDelete.Key)
        lruCache.DeleteKey(keyStr)

        fmt.Fprintf(w, "Key-Value pair for key %d has been deleted", toDelete.Key)
    } else {
        w.WriteHeader(http.StatusNotFound)
        fmt.Fprintf(w, "Key %d is not present", toDelete.Key)
    }
}

func main() {
    var err error
    db, err = sql.Open("mysql", "root:vedmumbai2003@tcp(127.0.0.1:3306)/decsdb")
    if err != nil {
        log.Fatalf("Failed to connect to the database: %v", err)
    }
    defer db.Close()

    if err := db.Ping(); err != nil {
        log.Fatalf("Database connection error: %v", err)
    }

    lruCache = cache.NewLRUCache(capacityCache)

    http.HandleFunc("/hello", hello)
    http.HandleFunc("/put", put)
    http.HandleFunc("/get", get)
    http.HandleFunc("/delete", del)

    log.Println("Server running on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
