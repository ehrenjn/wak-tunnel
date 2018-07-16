package main

//TO SPAWN A CLIENT THAT TALKS TO $name ON $port: tunnel $name $port 
	//SHOULD START A NEW INSTANCE OF THIS 'GRAM AND PRINT $random_port1
	//CONNECTING TO localhost:random_port1 SHOULD CONNECT TO SERVER
//ON THE SERVER SIDE, THERE SHOULD ONLY BE ONE INSTANCE RUNNING AT A TIME WITH MULTIPLE GORUTINES
	//WHEN IT DOWNLOADS A NEW CONNECTION IT SHOULD START A NEW SOCK GORUTINE WITH A RANDOM PORT
	//IT THEN USES THAT SOCK TO CONNECT TO $port

//GONNA NEED TO DO A sock.Close conn.Close AT SOME POINT
	//TIMEOUTS????
	//TIMEOUT ON THAT conn.Read 
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

//SHOULD FIGURE OUT HOW LONG TIME.SLEEP ACTUALLY WAITS FOR STUFF
//SHOULD DO ONE OF THOSE EXPONENTIAL WAIT TIME THINGIES FOR CLIENT (AND MAYBE FOR SERVER BUT DON'T GO AS FAR WITH IT) JUST BECAUSE IT'D BE A BIT OF A MESS TO JUST BE CONSTANTLY DLING WHILE A TUNNEL IS OPEN
	//OH WAIT THAT DOESN'T MAKE ANY SENSE BECAUSE AN OPEN TUNNEL IS JUST WAITING FOR CONNECTION, ITS NOT CONNECTED TO ANYTHING? HOPEFULLY? WHATEVER, THINK ABOUT IT

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

func post(url string, payload []byte) []byte { //does a post request
	resp, _ := http.Post(url, "", bytes.NewReader(payload)) //.Post takes a Reader for some reason
	defer resp.Body.Close()
	respBytes, _ := ioutil.ReadAll(resp.Body) //ReadAll does all the buffer stuff for me
	return respBytes
}

var CONN_READ_BUFFER_SIZE = 100000 //don't expect any message to ever be bigger than this
func readConn(conn net.Conn) []byte {
	data := make([]byte, CONN_READ_BUFFER_SIZE)
	bytesRead, err := conn.Read(data)
	if err != nil {
		fmt.Println("HALTING BECAUSE:", err)
		<-make(chan int)
	}
	return data[:bytesRead]
}

func client(to string, toPort string) { //actually a server, but pretends to be a client
	t := &tunnel{to, toPort, uniqueId(), 0}
	sock, _ := net.Listen("tcp", ":0")
	fmt.Println("Tunnel to", to + ":" + toPort, "open on", sock.Addr().String())
	for { //keep reusing same socket for every connection
		conn, _ := sock.Accept()
		fmt.Println("Recieved new connection")
		serverTunnelId := uniqueId() //a unique id for server tunnel to use
		t.upload([]byte(serverTunnelId), "open")
		t.ToId = serverTunnelId //all following messages are sent to new server tunnel
		exit := t.runConn(conn)
		fmt.Println(<-exit)
	}
}

func server(id string) { //actually a set of clients but pretends to be a server
	fmt.Println("Starting tunnel server", id, "...")
	serverGenerator := &tunnel{Id: id, lastMsgId: 0}
	serverGenerator.newMessages() //call just to update lastMsgId
	fmt.Println("Tunnel ready, waiting for connections")
	for { //keep checking for connection requests
		time.Sleep(500 * time.Millisecond)
		newMsgs := serverGenerator.newMessages()
		for _, msg := range newMsgs {
			if msg.Type == "open" && msg.Sender.ToId == id {
				go serverConnection(msg) //start new server connection
			}
		}
	}
}

func serverConnection(openingMsg message) { //acts as a single open port on the server
	fmt.Println("Opening new tunnel on port", openingMsg.Sender.ToPort)
	t := &tunnel{Id: string(openingMsg.Data), ToId: openingMsg.Sender.Id} //no lastMsgId 'cause geting time of open is hard. I don't need a lastMsgId because this tunnel has a unique id so I don't have any chance of picking up old messages by accident but it is kina gross that I'm looping through a hundred messages when I don't have to. Only alternative is to somehow unmarshal an id though and that is VERY hard
	conn, _ := net.Dial("tcp", "localhost:" + openingMsg.Sender.ToPort)
	exit := t.runConn(conn)
	fmt.Println(<-exit)
}

func (t *tunnel) runConn(conn net.Conn) chan string {
	exit := make(chan string)
	go func() { //keep uploading info from conn
		for {
			data := readConn(conn)
			fmt.Println("GOT DATA:", data)
			t.upload(data, "data")
		}
	}()
	go func() { //keep giving conn info
		download := t.downloader()
		for {
			response := <-download
			conn.Write(response.Data)
		}
	}()
	return exit
}

var UPLOAD_URL = "http://waksmemes.x10host.com/mess/?tunneling_tests2"
var MAX_DATA_PER_MSG = 5000 //playing it safe because b64 encoding and other parts of message make it longer
func (t *tunnel) upload(data []byte, msgType string) { //WON'T WORK FOR 0 BYTE MESSAGES
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

func (t *tunnel) downloader() chan message { //downloads the latest message intended for t, DOESN'T FOLLOW THE SAME PATTERN AS upload BECAUSE IT WOULD BE VERY DIFFICULT TO MAKE ONE CALL TO THIS FUNCTION RETURN ONE message (MAINLY BECAUSE I CAN'T Unmarshal MESSAGE idS)
	var currentMsg message
	output := make(chan message)
	go func() {
		for { //just keep downloading
			allMsgs := t.newMessages()
			for _, msg := range allMsgs {
				if msg.Sender.ToId == t.Id { //only look at data for this tunnel
					fmt.Println("Downloaded msg part", msg.Part, "of", msg.TotalParts)
					if msg.Part == 1 {
						currentMsg = msg
						currentMsg.Data = []byte{} //reset Data
					}
					msgChunkData := make([]byte, base64.StdEncoding.DecodedLen(len(msg.Data)))
					base64.StdEncoding.Decode(msgChunkData, msg.Data)
					currentMsg.Data = append(currentMsg.Data, msg.Data...)
					if msg.Part == msg.TotalParts {
						fmt.Println("Downloaded:", currentMsg.Data)
						output <- currentMsg
					}
				}
			}
			time.Sleep(100 * time.Millisecond) //don't go too crazy
		}
	}
	return output
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
		client("test", os.Args[2])
	} else if os.Args[1] == "server" {
		server("test")
	}
}
