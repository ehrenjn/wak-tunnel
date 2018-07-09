package main

//TO SPAWN A CLIENT THAT TALKS TO $name ON $port: tunnel $name $port 
	//SHOULD START A NEW INSTANCE OF THIS 'GRAM AND PRINT $random_port1
	//CONNECTING TO localhost:random_port1 SHOULD CONNECT TO SERVER
//ON THE SERVER SIDE, THERE SHOULD ONLY BE ONE INSTANCE RUNNING AT A TIME WITH MULTIPLE GORUTINES
	//WHEN IT DOWNLOADS A NEW CONNECTION IT SHOULD START A NEW SOCK GORUTINE WITH A RANDOM PORT
	//IT THEN USES THAT SOCK TO CONNECT TO $port

//GONNA NEED TO DO A sock.Close conn.Close AT SOME POINT
//UH OH HTTP HEADERS OFTEN HAVE A host: FIELD WHICH SAYS WHAT ADDRESS YOU CONNECTED TO
	//THIS WILL PROBABLY NO FRICK ANYTHING UP TOO BADLY, THE IP WOULD BE FINE BUT THE PORT WILL BE DIFFERENT
//NOW FIGURING OUT JSON POSTING AND DECODING

import (
	"fmt"
	"net"
	"net/http"
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"
	"encoding/json"
	"encoding/base64"
)

var exit = make(chan int)

type tunnel struct {
	ToId string //field starts with a capital letter so that its "exported" (so it can be Marshal'd)
	ToPort string
	Id string
}

type message struct {
	Sender tunnel
	Type string
	Data []byte
}


func client(to string, toPort string) { //actually a server, but pretends to be a client
	id := strconv.Itoa(rand.Intn(1000000000)) //unique id
	t := tunnel{to, toPort, id}
	sock, _ := net.Listen("tcp", ":0")
	fmt.Println("Tunnel to", to, "open on", sock.Addr().String())
	for { //keep reusing same socket for every connection
		conn, _ := sock.Accept()
		t.upload([]byte{}, "init")
		for { //while connection is alive
			response := t.download()
			conn.Write(response)
			data := []byte{}
			for len(data) == 0 {
				time.Sleep(100 * time.Millisecond) //sleep time specified in nanoseconds
				data, _ = ioutil.ReadAll(conn) //ReadAll does all the buffer stuff for me
			}
			t.upload(data, "data")
		}
	}
}

func server(id string) { //actually a client but pretends to be a server
	t := tunnel{Id: id}
	connRequest := t.download()
	fmt.Println(connRequest)
	//_, _ = net.Dial("tcp", "127.0.0.1:7878")
}

func (t tunnel) upload(data []byte, msgType string) {
	msg := message{Sender: t, Type: msgType, Data: data}
	encoded, _ := json.Marshal(msg)
	fmt.Println("Uploading:", string(encoded))
	//post data to waksmemes.x10host.com/mess/?tunneling_tests!post
}

func (t tunnel) download() []byte {
	fmt.Println(http.HandleFunc)
	//check waksmemes for new content intended for this comp
	//nab content, send it to the intended port
	//GONNA NEED SOME NOTION OF A SESSION SO I DON'T OPEN A NEW SOCKET EVERY TIME
	//ALSO GONNA WANT TO HAVE MULTIPLE CONNECTIONS MADE TO ONE SERVER AT ONCE
		//(SO YOU'LL NEED MULTIPLE SERVERS OPEN AT ONCE, EACH WITH 1 SOCKET)
	//IF YOU WANT MULTIPLE CLIENTS ON ONE COMP TOO THINGS COULD GET MESSY, THINK ABOUT IT
	return []byte("XD")
}

func main() {
	go client("test", "7878")
	fmt.Println(<-exit)
}
