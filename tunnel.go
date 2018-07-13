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

//HOW AM I RECOMBINING THE MESSAGES? IT WOULD BE INEFFICIENT TO JUST LOOP OVER ALL THE MESSAGES TO RECOMBINE THEM AND THEN LOOP OVER THEM AGIAN TO SEND TO THE conn, BUT THEN IF I MAKE THIS LOOPING EFFICIENT I MIGHT WANT TO THINK ABOUT MAKING ALL THE OTHER LOOPING EFFICIENT WHICH MIGHT BE ROUGH
	//NEED TO THINK ABOUT IF THERE SHOULD EVER BE > 1 CONNECTION MADE AT ONCE TO A CLIENT OR A serverConnection 
//ALSO CLIENT AND serverConnection HAVE A LOT OF SIMILARITIES, YOU SHOULD TRY TO COMBINE THE FUNCTIONS A LITTLE BIT

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


func uniqueId() string {
	return strconv.Itoa(rand.Intn(1000000000))
}

func readConn(conn net.TCPConn) []byte {
	data := []byte{}
	for len(data) == 0 {
		time.Sleep(100 * time.Millisecond) //sleep time specified in nanoseconds
		data, _ = ioutil.ReadAll(conn) //ReadAll does all the buffer stuff for me
	}
	return data
}

func client(to string, toPort string) { //actually a server, but pretends to be a client
	t := tunnel{to, toPort, uniqueId(), 0}
	sock, _ := net.Listen("tcp", ":0")
	fmt.Println("Tunnel to", to, "open on", sock.Addr().String())
	for { //keep reusing same socket for every connection
		conn, _ := sock.Accept()
		t.upload([]byte{}, "open")
		for { //while connection is alive
			response := t.download()
			conn.Write(response)
			data := readConn(conn)
			t.upload(data, "data")
		}
	}
}

func server(id string) { //actually a set of clients but pretends to be a server
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

func serverConnection(openingMsg message) { //acts as a single open port on the server
	t := tunnel{Id: uniqueId(), To: openingMsg.id}
	conn, _ = net.Dial("tcp", "127.0.0.1:7878")
	for { //while conn is alive
		data := readConn(conn)
		t.upload(data, "data")
		response := t.download()
		conn.write(response)
	}
}

var UPLOAD_URL = "http://waksmemes.x10host.com/mess/?tunneling_tests"
var MAX_DATA_PER_MSG = 5000 //playing it safe because b64 encoding and other parts of message make it longer
func (t tunnel) upload(data []byte, msgType string) { //waksmemes accepts <10000 byte messages
	dataLen := len(data)
	numMessages := (dataLen / MAX_DATA_PER_MSG) + 1
	for start := 0; start < dataLen; start += MAX_DATA_PER_MSG { //break data into chunks, send each chunk
		end := start + MAX_DATA_PER_MSG
		if end >= dataLen {
			end = dataLen
		}
		dataSlice := data[start: end]
		dataB64 := base64.StdEncoding.EncodeToString(dataSlice)
		msg := message{t, msgType, dataB64, m, numMessages}
		encoded, _ := json.Marshal(msg)
		fmt.Println("Uploading:", string(encoded))
		http.Post(UPLOAD_URL + "!post", nil, encoded)
	}
}

func recombineMessages

var ID_REGEX = regexp.MustCompile(`"id": (\d+?)`)
func (t tunnel) download() []message { //downloads all new messages intended for t
	fmt.Println(http.HandleFunc)
	filter := `{"id": {"min": ` + t.lastMsgId + `}`
	allJson := http.Post(UPLOAD_URL + "!get", "", filter)
	var allMsgs []messages
	allMsgs := json.Unmarshal(allJson, &allMsgs)
	goodMsgs := make([]message, len(allMsgs)
	for _, msg := range allMsgs { //inefficient, considering I'll have to loop over them agian later. whatever
		if msg.Sender.ToId == t.Id {
			append(goodMsgs, msg)
		}
	}
	lastIdStr = string(ID_REGEX.find(allJson)
	t.lastMsgId = strconv.Atoi(lastIdStr)
	return goodMsgs
}

func main() {
	go client("test", "7878")
	fmt.Println(<-EXIT)
}
