package main

//TO SPAWN A CLIENT THAT TALKS TO $name ON $port: tunnel $name $port 
	//SHOULD START A NEW INSTANCE OF THIS 'GRAM AND PRINT $random_port1
	//CONNECTING TO localhost:random_port1 SHOULD CONNECT TO SERVER
//ON THE SERVER SIDE, THERE SHOULD ONLY BE ONE INSTANCE RUNNING AT A TIME WITH MULTIPLE GORUTINES
	//WHEN IT DOWNLOADS A NEW CONNECTION IT SHOULD START A NEW SOCK GORUTINE WITH A RANDOM PORT
	//IT THEN USES THAT SOCK TO CONNECT TO $port

//GONNA NEED TO DO A sock.Close conn.Close AT SOME POINT
	//TIMEOUTS????
//UH OH HTTP HEADERS OFTEN HAVE A host: FIELD WHICH SAYS WHAT ADDRESS YOU CONNECTED TO
	//THIS WILL PROBABLY NO FRICK ANYTHING UP TOO BADLY, THE IP WOULD BE FINE BUT THE PORT WILL BE DIFFERENT
//MIGHT WANT TO LOWER THE DELAY TIME IN client WHEN ITS WAITING FOR A READ FROM conn
//GONNA NEED TO THINK ABOUT MULTIPLE CLIENTS CONNECTING TO ONE SERVER FROM DIFFERENT COMPUTERS TRYING TO USE THE SAME PORT AND STUFF

//HOLY MOLY HOW ARE YOU UnmarshalING THAT id HUH????

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

var EXIT = make(chan int)

type tunnel struct { //represents connection to waksmemes.x10host
	ToId string //field starts with a capital letter so that its "exported" (so it can be Marshal'd)
	ToPort string
	Id string
	lastMsgId int //unexported so it's not Marshal'd
}

type message struct { //represents a message to be posted to waksmemes
	Sender tunnel
	Type string
	Data string
	Part int
	TotalParts int
}


func client(to string, toPort string) { //actually a server, but pretends to be a client
	id := strconv.Itoa(rand.Intn(1000000000)) //unique id
	t := tunnel{to, toPort, id, 0}
	sock, _ := net.Listen("tcp", ":0")
	fmt.Println("Tunnel to", to, "open on", sock.Addr().String())
	for { //keep reusing same socket for every connection
		conn, _ := sock.Accept()
		t.upload([]byte{}, "open")
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
	serverGenerator = tunnel{Id: id}
	for { //keep checking for connection requests
		time.sleep(500 * time.Millisecond)
		newMsgs = serverGenerator.download()
		for _, msg := range newMsgs {
			if msg.Type == "open" {
				go serverConnection(msg) //start new server connection
			}
		}
	}
}

func serverConnection(serverId, openingMsg message) { //acts as a single open port on the server
	t := tunnel{Id: serverId, To: openingMsg.id}
	connRequest := t.download()
	fmt.Println(connRequest)
	//_, _ = net.Dial("tcp", "127.0.0.1:7878")
}

var MAX_DATA_PER_MSG = 5000 //playing it safe because b64 encoding and other parts of message make it longer
func (t tunnel) createMessages(data []byte, msgType string) []message { //waksmemes accepts <10000 byte messages
	numMessages := (len(data) / MAX_DATA_PER_MSG) + 1
	allMessages := make([]message, numMessages)
	for m := 0; m < numMessages; m++ {
		min := m * MAX_DATA_PER_MSG
		max := min + MAX_DATA_PER_MSG
		if max >= len(data) {
			max = len(data)
		}
		dataSlice := data[min: max]
		dataB64 := base64.StdEncoding.EncodeToString(dataSlice)
		allMessages[m] = message{t, msgType, dataB64, m, numMessages}
	}
	return allMessages
}

var UPLOAD_URL = "http://waksmemes.x10host.com/mess/?tunneling_tests"
func (t tunnel) upload(data []byte, msgType string) {
	allMsgs := t.createMessages(data, msgType)
	for _, msg := range allMsgs {
		encoded, _ := json.Marshal(msg)
		fmt.Println("Uploading:", string(encoded))
		//http.Post(UPLOAD_URL + "!post", nil, encoded)
	}
}

func (t tunnel) download() []message { //downloads all new messages intended for t
	fmt.Println(http.HandleFunc)
	filter := `{"id": {"min": ` + t.lastMsgId + `}`
	allJson := http.Post(UPLOAD_URL + "!get", "", filter)
	var allMsgs []messages
	allMsgs := json.Unmarshal(allJson, &allMsgs)
	for _, msg := range allMsgs { //inefficient, considering I'll have to loop over them agian later. whatever
		//uhhhhhh
	}
	t.lastMsgId = allMsgs[-1]//get the last id somehow
}

func main() {
	go client("test", "7878")
	fmt.Println(<-EXIT)
}
