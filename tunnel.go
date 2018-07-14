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
//GONNA NEED TO THINK ABOUT MULTIPLE CLIENTS CONNECTING TO ONE SERVER FROM DIFFERENT COMPUTERS TRYING TO USE THE SAME PORT AND STUFF (NOT using the same tunnel, just same port)

//CLIENT AND serverConnection HAVE A LOT OF SIMILARITIES, YOU SHOULD TRY TO COMBINE THE FUNCTIONS A LITTLE BIT
//UGH NOW server NEEDS TO DOWNLOAD ALL ITS MESSAGES EVEN THOUGH EVERYTHING ELSE JUST NEEDS ONE MESSAGE, GONNA HAVE TO MAKE A NEW FUNCTION TO DO THAT
	//REMEMBER THAT THIS NEW FUNCTION IS FOR openERS ONLY SO IT DOESN'T NEED TO CHECK FOR MULTIPART MESSAGES
	//YOU STILL NEED MESSAGE TYPES THOUGH!!! BECAUSE OF CLOSING MESSAGES THAT YOU HAVEN'T IMPLEMENTED YET

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


type tunnel struct { //represents connection to waksmemes.x10host
	ToId string //field starts with a capital letter so that its "exported" (so it can be Marshal'd)
	ToPort string
	Id string
	lastMsgId int //unexported so it's not Marshal'd
}

type message struct { //represents a message to be posted to waksmemes
	Sender tunnel
	Type string
	Data []byte
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
			t.ToId = response.Sender.Id
			conn.Write(response.Data)
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
			if msg.Type == "open" && msg.Id == id {
				go serverConnection(msg) //start new server connection
			}
		}
	}
}

func serverConnection(openingMsg message) { //acts as a single open port on the server
	t := tunnel{Id: uniqueId(), To: openingMsg.id}
	conn, _ = net.Dial("tcp", "localhost:" + openingMsg.Sender.ToPort)
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
		part := (start / MAX_DATA_PER_MSG) + 1
		msg := message{t, msgType, dataB64, part, numMessages}
		encoded, _ := json.Marshal(msg)
		fmt.Println("Uploading:", string(encoded))
		http.Post(UPLOAD_URL + "!post", nil, encoded)
	}
}

var ID_REGEX = regexp.MustCompile(`"id": (\d+?)`)
func (t tunnel) download() message { //downloads the latest message intended for t
	var fullData []byte
	lastMsgChunk := message{Part: -1, TotalParts: 1}
	for lastMsgChunk.Part < lastMsgChunk.TotalParts { //loop until whole message recieved
		filter := `{"id": {"min": ` + t.lastMsgId + `}`
		allJson, _ := http.Post(UPLOAD_URL + "!get", "", filter)
		if allJson != "[]" { //if we recieved any data
			var allMsgs []messages
			allMsgs := json.Unmarshal(allJson, &allMsgs)
			for _, msg = range allMsgs {
				if msg.Sender.ToId == t.Id { //get all the data we can
					fullData = append(fullData, msg.Data...)
					lastMsgChunk = msg
				}
			}
			lastIdStr = string(ID_REGEX.find(allJson)
			t.lastMsgId = strconv.Atoi(lastIdStr) //don't want to get the same data twice!
		}
	}
	lastMsgChunk.Data = fullData //Last part of the message should have all important data except for fullData
	return lastMsgChunk
}

func (t tunnel) newMessages() []message {
	
}


var EXIT = make(chan int)
func main() {
	go client("test", "7878")
	fmt.Println(<-EXIT)
}
