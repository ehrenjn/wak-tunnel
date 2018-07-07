package main

//TO SPAWN A CLIENT THAT TALKS TO $name ON $port: tunnel $name $port 
	//SHOULD START A NEW INSTANCE OF THIS 'GRAM AND PRINT $random_port1
	//CONNECTING TO localhost:random_port1 SHOULD CONNECT TO SERVER
//ON THE SERVER SIDE, THERE SHOULD ONLY BE ONE INSTANCE RUNNING AT A TIME WITH MULTIPLE GORUTINES
	//WHEN IT DOWNLOADS A NEW CONNECTION IT SHOULD START A NEW SOCK GORUTINE WITH A RANDOM PORT
	//IT THEN USES THAT SOCK TO CONNECT TO $port

//GONNA WANT A tunnel STRUCT TO KEEP TRACK OF to AND id AND MAKE upload AND download INTO METHODS
//ALSO WANT TO STOP IT FROM CONSTANTLY READING FROM conn, DOES JUST CHECKING IF data IS EMPTY WORK?

import (
	"fmt"
	"net"
	"net/http"
	"io/ioutil"
	"math/rand"
	"strconv"
)

var exit = make(chan int)


func client(to string) { //actually a server, but pretends to be a client
	id := strconv.Itoa(rand.Intn(1000000000)) //unique id
	sock, _ := net.Listen("tcp", ":0")
	fmt.Println("Tunnel to", to, "open on", sock.Addr().String())
	for { //keep reusing same socket for every connection
		conn, _ := sock.Accept()
		upload(to, id, []byte{}, "init")
		for { //while connection is alive
			response := download()
			conn.Write(response)
			data, _ := ioutil.ReadAll(conn) //calls conn.Read until it gets all its info
			upload(data)
			fmt.Println(string(data))
		}
	}
}

func testClient(addr string) {
	net.Dial("tcp", addr)
}

func server() { //actually a client but pretends to be a server
	_, _ = net.Dial("tcp", "127.0.0.1:7878")
}

func upload(to string, from string, data []byte, msgType string) {
	fmt.Println(msgType, data)
	//post data to waksmemes.x10host.com/mess/?tunneling_tests!post
}

func download() []byte {
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
	go client("test")
	fmt.Println(<-exit)
}
