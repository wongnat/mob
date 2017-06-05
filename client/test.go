package main

import "fmt"
import "net"

func main() {
    conn, _ := net.Dial("udp","8.8.8.8:80")
    //conn = nil
    conn = nil
    if conn != nil {
        fmt.Println("not nil")
    } else {
        fmt.Println("Can make structs nil")
    }
}
