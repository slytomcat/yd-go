package main

import (
  "log"
  "fmt"
  YDisk "github.com/slytomcat/YD.go/YDisk"
  "os"
  "encoding/json"
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

  YD := YDisk.NewYDisk(os.Args[1], os.Args[2])
  //YD := NewYDisk("/home/stc/.config/yandex-disk/config.cfg", "/home/stc/Yandex.Disk")

  //
  go func() {
    Logger.Println("Staus updates formater started")
    var msj []byte
    for {
      yds, ok := <- YD.Updates
      if ok {
        msj, _ = json.Marshal(yds)
        fmt.Println(string(msj))
      } else {
        Logger.Println("Staus updates formater exited.")
        return
      }

    }
  }()

  // stdin reader cycle
  var inp string = ""
  var msj []byte
  for inp != "exit" {
    //fmt.Println("Commands: start, stop, output, exit")
    inp = ""
    fmt.Scanln(&inp)
    switch inp {
      case "start":
        YD.Start()
      case "stop":
        YD.Stop()
      case "output":
        msj, _ = json.Marshal(YD.Output())
        fmt.Println("{\"Output\": " + string(msj) + "}")
    }
  }
  Logger.Println("Exit requested.")
  YD.Close()
}
