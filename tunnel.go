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

//UGGGH NOW I REALIZE THAT IN HTTP THE CLIENT IS THE ONE THAT SENDS DATA FIRST, THERES A FEW WAYS OF SOLVING THIS ONE
	//YOU HAVE TO KNOW WHETHER THIS IS TRUE IN ALL TCP BASED PROTOS. IT PROBABLY ISN'T
	//IF IT IS TRUE YOU COULD JUST HAVE THE SERVER SEND AN ack BACK WHEN A CONNECTION IS MADE
	//IF IT IS TRUE A BETTER BUT PRETTY ANNOYING SOLUTION WOULD BE SENDING DATA WITH THE open MESSAGE
	//IF ITS NOT TRUE YOU NEED TO FIGURE OUT A CRAZY MODEL THAT PROBABLY INVOLVES LISTENING FOR MORE DATA AT THE SAME TIME AS SENDING IT


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
	"regexp"
	"bytes"
	"os"
)


type tunnel struct { //represents connection to waksmemes.x10host
	ToId string //field starts with a capital letter so that its "exported" (so it can be Marshal'd)
	ToPort string
	Id string
	lastMsgId int //unexported so it's not Marshal'd
}

type message struct { //represents a message to be posted to waksmemes
	Sender *tunnel
	Type string
	Data []byte
	Part int
	TotalParts int
}


func uniqueId() string {
	return strconv.Itoa(rand.Int())
}

func readConn(conn net.Conn) []byte { //CAN'T RECIEVE 0 LENGTH MESSAGES, but I don't think I need to?
	data := []byte{}
	for len(data) == 0 {
		time.Sleep(100 * time.Millisecond) //sleep time specified in nanoseconds
		data, _ = ioutil.ReadAll(conn) //ReadAll does all the buffer stuff for me
	}
	return data
}

func post(url string, payload []byte) []byte { //does a post request
	fmt.Println("Posting:", string(payload))
	resp, _ := http.Post(url, "", bytes.NewReader(payload)) //.Post takes a Reader for some reason
	defer resp.Body.Close()
	respBytes, _ := ioutil.ReadAll(resp.Body)
	return respBytes
}

func client(to string, toPort string) { //actually a server, but pretends to be a client
	t := &tunnel{to, toPort, uniqueId(), 0}
	sock, _ := net.Listen("tcp", ":0")
	fmt.Println("Tunnel to", to, "open on", sock.Addr().String())
	for { //keep reusing same socket for every connection
		conn, _ := sock.Accept()
		fmt.Println("Recieved new connection")
		t.upload([]byte{0}, "open") //just send a single character
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
	serverGenerator := &tunnel{Id: id, lastMsgId: 0}
	serverGenerator.newMessages() //call just to update lastMsgId
	for { //keep checking for connection requests
		time.Sleep(500 * time.Millisecond)
		newMsgs := serverGenerator.newMessages()
		for _, msg := range newMsgs {
			if msg.Type == "open" && msg.Sender.ToId == id {
				go serverConnection(msg, serverGenerator.lastMsgId) //start new server connection
			}
		}
	}
}

func serverConnection(openingMsg message, lastMsgId int) { //acts as a single open port on the server
	fmt.Println("Opening new tunnel on port", openingMsg.Sender.ToPort)
	t := &tunnel{Id: uniqueId(), ToId: openingMsg.Sender.Id, lastMsgId: lastMsgId}
	conn, _ := net.Dial("tcp", "localhost:" + openingMsg.Sender.ToPort)
	for { //while conn is alive
		data := readConn(conn)
		t.upload(data, "data")
		response := t.download()
		conn.Write(response.Data)
	}
}

var UPLOAD_URL = "http://waksmemes.x10host.com/mess/?tunneling_tests2"
var MAX_DATA_PER_MSG = 5000 //playing it safe because b64 encoding and other parts of message make it longer
//upload WON'T WORK FOR 0 BYTE MESSAGES, but that shoudln't matter because readConn can't recieve them anyway
func (t *tunnel) upload(data []byte, msgType string) {
	dataLen := len(data)
	numMessages := (dataLen / MAX_DATA_PER_MSG) + 1
	for start := 0; start < dataLen; start += MAX_DATA_PER_MSG { //break data into chunks, send each chunk
		end := start + MAX_DATA_PER_MSG
		if end >= dataLen {
			end = dataLen
		}
		dataSlice := data[start: end]
		dataB64 := make([]byte, base64.StdEncoding.EncodedLen(len(dataSlice)))
		base64.StdEncoding.Encode(dataB64, dataSlice) //no & because slices are mutable!
		part := (start / MAX_DATA_PER_MSG) + 1
		msg := message{t, msgType, dataB64, part, numMessages}
		encoded, _ := json.Marshal(msg) //Marshal magically figures out pointers which is pretty nice
		fmt.Println("Uploading:", string(encoded))
		post(UPLOAD_URL + "!post", encoded)
	}
}

func (t *tunnel) download() message { //downloads the latest message intended for t
	var fullData []byte
	lastMsgChunk := message{Part: -1, TotalParts: 1}
	for lastMsgChunk.Part < lastMsgChunk.TotalParts { //loop until whole message recieved
		allMsgs := t.newMessages()
		for _, msg := range allMsgs {
			if msg.Sender.ToId == t.Id { //only look at data for this tunnel
				msgChunkData := make([]byte, base64.StdEncoding.DecodedLen(len(msg.Data)))
				base64.StdEncoding.Decode(msgChunkData, msg.Data)
				fullData = append(fullData, msg.Data...)
				lastMsgChunk = msg
			}
		}
	}
	lastMsgChunk.Data = fullData //Last part of the message should have all important data except for fullData
	return lastMsgChunk
}

var ID_REGEX = regexp.MustCompile(`"id":(\d+?),`)
func (t *tunnel) newMessages() []message { //downloads all the messages this tunnel hasn't encountered yet
	filter := fmt.Sprintf(`{"id": {"min": %d}}`, t.lastMsgId + 1)
	allJson := post(UPLOAD_URL + "!get", []byte(filter))
	var allMsgs []message
	json.Unmarshal(allJson, &allMsgs)
	if len(allMsgs) > 0 { //don't want to get the same data twice!
		lastMsgIdBytes := ID_REGEX.FindSubmatch(allJson)[1]
		t.lastMsgId, _ = strconv.Atoi(string(lastMsgIdBytes))
	}
	return allMsgs
}


func main() {
	rand.Seed(time.Now().UnixNano()) //have to seed the rng
	if os.Args[1] == "client" {
		client("test", "7788")
	} else if os.Args[1] == "server" {
		server("test")
	}
}