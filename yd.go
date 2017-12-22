package main

import (
  "log"
  "fmt"
  YDisk "github.com/slytomcat/YD.go/YDisk.go"
  "os"
)

/* Initialize default logger */
var Logger *log.Logger = log.New(os.Stderr, "", log.Lshortfile|log.Lmicroseconds) // | log.Lmicroseconds)


func main() {
  if len(os.Args) < 3 {
    Logger.Fatal("Error: Path to yandex-disc config-file and path to synchronized folder",
             "must be provided via first and second command line arguments")
  }

  // stdin <- Commads
  // Status updates -> stdout
  // Log messages -> stderr
  // External program/operator have to decide what to do with daemon and pass command.
  // Wrapper itself doesn't auto-start or stop daemo on its start/exit

  YD := NewYDisk(os.Args[1], os.Args[2], func(s string) {fmt.Println(s)})
  //YD := NewYDisk("/home/stc/.config/yandex-disk/config.cfg", "/home/stc/Yandex.Disk")

  // stdin reader cycle
  var inp string = ""
  for inp != "exit" {
    //fmt.Println("Commands: start, stop, output, exit")
    inp = ""
    fmt.Scanln(&inp)
    YD.Commands <- inp
  }
  Logger.Println("Exit requested.")
  YD.Close()
}
