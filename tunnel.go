package main

//UH OH HTTP HEADERS OFTEN HAVE A host: FIELD WHICH SAYS WHAT ADDRESS YOU CONNECTED TO
	//THIS WILL PROBABLY NO FRICK ANYTHING UP TOO BADLY, THE IP WOULD BE FINE BUT THE PORT WILL BE DIFFERENT
//GONNA NEED TO THINK ABOUT MULTIPLE CLIENTS CONNECTING TO ONE SERVER FROM DIFFERENT COMPUTERS TRYING TO USE THE SAME PORT AND STUFF (NOT using the same tunnel, just same port)

//SHOULD FIGURE OUT HOW LONG TIME.SLEEP ACTUALLY WAITS FOR STUFF
//SHOULD DO ONE OF THOSE EXPONENTIAL WAIT TIME THINGIES FOR CLIENT (AND MAYBE FOR SERVER BUT DON'T GO AS FAR WITH IT) JUST BECAUSE IT'D BE A BIT OF A MESS TO JUST BE CONSTANTLY DLING WHILE A TUNNEL IS OPEN
	//OH WAIT THAT DOESN'T MAKE ANY SENSE BECAUSE AN OPEN TUNNEL IS JUST WAITING FOR CONNECTION, ITS NOT CONNECTED TO ANYTHING? HOPEFULLY? WHATEVER, THINK ABOUT IT

//SOMETIMES wget HAS TO TRY TWICE OR SOMETHING? IDK
//CHANGE 100K LIMIT, ALSO SET MAX_MSGS = 1000 ON THE DOWNLOAD
//FIGURE OUT WHY YOU'RE BOOTED OFF SSH AFTER A WHILE
//ADD AN ERROR HANDLE FUNC THAT PRINTS ERRORS IF THEY'RE NON NIL
//DO ARG PARSING
//NEED TO KEEP TRACK OF ALL GORUTINES AND FIGURE OUT HOW TO CLOSE HANGING GORUTUNES
	//MAYBE JUST TIMEOUTS? 30 MINS WITH NO DOWNLOAD?

//OH DAMN THE LOOPBACK NETWORK THINGS SEEMS LIKE IT WOULD ACTUALLY WORK: EVERY SERVER YOU CONNECT TO GETS ASSIGNED A LOOPBACK IP WITH A NAME ON YOUR ROUTING TABLE SO YOU CAN JUST DO STUFF LIKE tunnel client crypto 22; ssh crypto@crypto
	//UHHHHHH THIS COULD FRICK WITH SOME STUFF BECAUSE REMEMBER THE host: FIELD PROBLEM?? THIS WOULD BE MORE LIKELY TO BE A PROBLEM WITH IPS THAN PORTS
	//TO FIX THE ISSUE YOU'D HAVE TO MAKE THE SERVER CONNECT TO THE SAME LOOPBACK ADDRESS WHICH WOULD BE KIND OF ANNOYING

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
	conn net.Conn
	open bool
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

func (t *tunnel) connReader() chan message {
	output := make(chan message)
	go func() {
		for t.open { //while the tunnel's open
			var err error
			data := []byte{}
			readAmt := 1000
			var totalBytesRead, bytesRead int
			for err == nil { //keep reading data until we timeout
				newData := make([]byte, readAmt)
				t.conn.SetReadDeadline(time.Now().Add(time.Millisecond)) //if we timeout it means theres nothing left to read
				bytesRead, err = t.conn.Read(newData)
				totalBytesRead += bytesRead
				data = append(data, newData...)
				readAmt *= 2 //double readAmt every round
			}
			if totalBytesRead > 0 { //ALWAYS OUTPUT DATA EVEN IF THERES AN ERROR BECAUSE IT CAN HAVE AN EOF RIGHT AT THE END OF ACTUAL DATA IF SOMEONE SENDS DATA AND THEN CLOSES RIGHT AWAY
					output <- message{Data: data[:totalBytesRead], Type: "data"}
			}
			netErr, ok := err.(net.Error) //have to cast to use .Timeout()
			if !ok || !netErr.Timeout() { //close the conn if we got a non timeout error
				data = []byte(err.Error())
				bytesRead = len(data)
				fmt.Println("Conn err (probably just closing):", err)
				output <- message{Data: data, Type: "close"}
				return //halt now that the conn is closed
			}
			time.Sleep(100 * time.Millisecond) //don't go too crazy
		}
		fmt.Println("readConn closing because tunnel was closed by download")
		output <- message{Data: []byte("tunnel was closed by download"), Type: "close"}
	}()
	return output
}

func b64encode(data []byte) []byte {
	dataB64 := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(dataB64, data) //no & because slices are mutable!
	return dataB64
}

func b64decode(dataB64 []byte) []byte {
	data := make([]byte, base64.StdEncoding.DecodedLen(len(dataB64)))
	amt, _ := base64.StdEncoding.Decode(data, dataB64)
	return data[:amt]
}

func client(to string, toPort string) { //actually a server, but pretends to be a client	
	sock, _ := net.Listen("tcp", ":0")
	fmt.Println("Tunnel to", to + ":" + toPort, "open on", sock.Addr().String())
	for { //keep reusing same socket for every connection
		conn, _ := sock.Accept() //you can have > 1 conn per port?? how did I just find out about this?
		fmt.Println("Recieved new connection")
		t := &tunnel{to, toPort, uniqueId(), 0, conn, true}
		go clientConnection(t)
	}
}

func clientConnection(t *tunnel) {
	serverTunnelId := uniqueId() //also make a unique id for server tunnel to use
	t.upload(message{Data: []byte(serverTunnelId), Type: "open"}) //connect the tunnel
	t.ToId = serverTunnelId //all following messages are sent to new server tunnel
	t.runConn() //then keep running untill the connection is closed
}

func server(id string) { //actually a set of clients but pretends to be a server
	fmt.Println("Starting tunnel server", id, "...")
	serverGenerator := &tunnel{Id: id, lastMsgId: 0}
	serverGenerator.newMessages() //call just to update lastMsgId
	fmt.Println("Tunnel ready, waiting for connections")
	for { //keep checking for connection requests
		time.Sleep(500 * time.Millisecond)
		newMsgs := serverGenerator.newMessages()
		for _, msg := range newMsgs { //order doesn't matter so we'll just loop forward
			if msg.Type == "open" && msg.Sender.ToId == id {
				go serverConnection(msg) //start new server connection
			}
		}
	}
}

func serverConnection(openingMsg message) { //acts as a single open port on the server
	tunnelId := string(b64decode(openingMsg.Data))
	fmt.Println("Opening new tunnel on port", openingMsg.Sender.ToPort, "(id ", b64decode(openingMsg.Data), ")")
	conn, err := net.Dial("tcp", "localhost:" + openingMsg.Sender.ToPort)
	if err != nil {
		fmt.Println("CONN ERR!!!!!!:", err)
	}
	t := &tunnel{Id: tunnelId, ToId: openingMsg.Sender.Id, conn: conn, open: true} //no lastMsgId 'cause geting time of open is hard. I don't need a lastMsgId because this tunnel has a unique id so I don't have any chance of picking up old messages by accident but it is kina gross that I'm looping through a hundred messages when I don't have to. Only alternative is to somehow unmarshal an id though and that is VERY hard
	t.runConn()
}

func (t *tunnel) runConn() {
	exit := make(chan string)
	go func() { //keep uploading info from conn
		readConn := t.connReader()
		for t.open {
			msg := <-readConn
			fmt.Println("\nRead Conn Data (to upload):\n", string(msg.Data))
			t.upload(msg)
			fmt.Println("GOT HERE")
			if msg.Type == "close" {
				t.open = false //set a flag instead of just breaking so that the download gorutine exits too
			}
		}
		t.conn.Close() //doesn't mirror the download func but it's the most convenient spot to .Close
		exit <- "conn had an error or was closed by client"
	}()
	go func() { //keep giving conn info
		download := t.downloader()
		for t.open {
			response := <-download
			if response.Type == "data" {
				fmt.Println("Writing", len(response.Data), "bytes to conn")
				_, err := t.conn.Write(response.Data)
				fmt.Println("POSSIBLE WRITING ERR (MAKE PROPER HANDLING LATER)??:", err)
			} else if response.Type == "close" {
				t.open = false
			}
		}
		exit <- "the other tunnel had an error or their conn was closed"
	}()
	fmt.Println(<-exit)
	fmt.Println(<-exit)
	fmt.Println("CONN DONE RUNNING")
}

var UPLOAD_URL = "http://waksmemes.x10host.com/mess/?tunneling_tests2"
var MAX_DATA_PER_MSG = 5000 //playing it safe because b64 encoding and other parts of message make it longer
func (t *tunnel) upload(fullMsg message) {
	dataLen := len(fullMsg.Data)
	offset := 1
	if dataLen % MAX_DATA_PER_MSG == 0 { //only offset by 1 if datalen is not a multiple of MAX_DATA_PER_MSG
		offset = 0
	}
	numMessages := (dataLen / MAX_DATA_PER_MSG) + offset //offset is kinda gross but its all I could think of
	for start := 0; start < dataLen; start += MAX_DATA_PER_MSG { //break data into chunks, send each chunk
		end := start + MAX_DATA_PER_MSG
		if end >= dataLen {
			end = dataLen
		}
		dataSlice := fullMsg.Data[start: end]
		dataB64 := b64encode(dataSlice)
		part := (start / MAX_DATA_PER_MSG) + 1
		msg := message{t, fullMsg.Type, dataB64, part, numMessages}
		encoded, _ := json.Marshal(msg) //Marshal magically figures out pointers which is pretty nice
		fmt.Println("Uploading", len(encoded), "bytes")
		post(UPLOAD_URL + "!post", encoded)
	}
}

func (t *tunnel) downloader() chan message { //downloads the latest message intended for t, DOESN'T FOLLOW THE SAME PATTERN AS upload BECAUSE IT WOULD BE VERY DIFFICULT TO MAKE ONE CALL TO THIS FUNCTION RETURN ONE message (MAINLY BECAUSE I CAN'T Unmarshal MESSAGE idS)
	var currentMsg message
	output := make(chan message)
	go func() {
		for t.open { //just keep downloading until the tunnel closes down
			allMsgs := t.newMessages()
			for m := len(allMsgs) - 1; m >= 0; m-- { //LOOPING BACKWARDS BECAUSE NEED OLDEST FIRST
				msg := allMsgs[m]
				if msg.Sender.ToId == t.Id { //only look at data for this tunnel
					fmt.Println("Downloaded msg part", msg.Part, "of", msg.TotalParts)
					if msg.Part == 1 {
						currentMsg = msg
						currentMsg.Data = []byte{} //reset Data
					}
					msg.Data = b64decode(msg.Data)
					currentMsg.Data = append(currentMsg.Data, msg.Data...)
					if msg.Part == msg.TotalParts {
						fmt.Println("\nDownloaded:\n", string(currentMsg.Data))
						output <- currentMsg
						if currentMsg.Type == "close" { //only way out of downloading
							fmt.Println("download closing because it's recieved a close message")
							return
						}
					}
				}
			}
			time.Sleep(100 * time.Millisecond) //don't go too crazy
		}
		fmt.Println("downloader closing because readConn closed the tunnel's conn")
		output <- message{Data: []byte("readConn closed the tunnel's conn"), Type: "close"}
	}()
	return output
}

var ID_REGEX = regexp.MustCompile(`"id":(\d+?),`)
func (t *tunnel) newMessages() []message { //downloads all the messages this tunnel hasn't encountered yet
	filter := fmt.Sprintf(`{"id": {"min": %d}, "MAX_MSGS": 1000}`, t.lastMsgId + 1)
	allJson := post(UPLOAD_URL + "!get", []byte(filter))
	var allMsgs []message
	json.Unmarshal(allJson, &allMsgs)
	if len(allMsgs) > 0 { //don't want to get the same data twice!
		lastMsgIdBytes := ID_REGEX.FindSubmatch(allJson)[1]
		t.lastMsgId, _ = strconv.Atoi(string(lastMsgIdBytes))
	}
	return allMsgs //returns NEWEST first NOT OLDEST
}


func main() {
	rand.Seed(time.Now().UnixNano()) //have to seed the rng
	if os.Args[1] == "client" {
		client(os.Args[2], os.Args[3])
	} else if os.Args[1] == "server" {
		server(os.Args[2])
	}
}
