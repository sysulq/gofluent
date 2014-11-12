package httpsqs

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"errors"
	"net/url"
	"net"
	"log"
	"strings"
	"strconv"
	"encoding/json"
	"bufio"
	"net/textproto"
	"io"
	"time"
)

type Client struct {
	charset			string
	auth			string
	host			string
	port			int32
	isDebug			bool
	pConn			map[string]*HttpsqsPersistenceConn
}


type HttpsqsPersistenceConn struct {
	pConnection	    *http.Client
	pConnResp		*http.Response
	pConn			net.Conn
}


type HttpsqsResponseString string



type HttpsqsResponse struct {
	Pos			int
	Data		HttpsqsResponseString
}


type HttpsqsQueueStatus struct {
	Name		string		`json:"name"`
	Maxqueue	int			`json:"maxqueue"`
	Putpos		int         `json:"putpos"`
	Putlap		int			`json:"putlap"`
	Getpos		int			`json:"getpos"`
	Getlap		int			`json:"getlap"`
	Unread		int			`json:"unread"`
}


func NewClient(host string, port int32,  auth string, isDebug bool) (*Client) {

	client := &Client{charset : "utf-8", auth: auth, host: host, port: port, isDebug: isDebug}
	client.pConn = make(map[string]*HttpsqsPersistenceConn, GLOBAL_HTTPSQS_PSOCKET)
	return client
}


func (client *Client) Get(key string) (result HttpsqsResponseString, err error) {

	err, data := client.http_get("get", key, nil)
	result = data.Data

	if "HTTPSQS_GET_END" == data.Data || "HTTPSQS_ERROR" == data.Data || err != nil {
		result = ""
		err = errors.New(string(data.Data))
	}
	return
}

func (client *Client) Gets(key string) (result HttpsqsResponse, err error) {

	err, result = client.http_get("get", key, nil)

	if "HTTPSQS_GET_END" == result.Data || "HTTPSQS_ERROR" == result.Data || err != nil {
		if err == nil {
			err = errors.New(string(result.Data))
		}
		result.Data = ""
	}
	return
}

func (client *Client) Puts(key string, value string) (result HttpsqsResponseString, err error) {

	err, rsdata := client.http_post("put", key, &value, nil)

	if "HTTPSQS_PUT_OK" != rsdata.Data  || err != nil {
		if err == nil {
			err = errors.New(string(rsdata.Data))
		}
		rsdata.Data = ""
	}
	return rsdata.Data, err
}


func (client *Client) Status(key string) (result HttpsqsResponseString, err error) {

	err, rsdata := client.http_get("status", key, nil)

	if "HTTPSQS_ERROR" == rsdata.Data || err != nil {
		if err == nil {
			err = errors.New(string(rsdata.Data))
		}
		rsdata.Data = ""
	}
	return rsdata.Data, err
}


func (client *Client) StatusJson(key string) (result HttpsqsQueueStatus, err error) {

	err, rsdata := client.http_get("status_json", key, nil)

	if "HTTPSQS_ERROR" == rsdata.Data || err != nil {
		if err == nil {
			err = errors.New(string(rsdata.Data))
		}
	} else {
		json.Unmarshal([]byte(rsdata.Data), &result)
	}
	return
}

func (client *Client) View(key string, pos int) (result HttpsqsResponseString, err error) {

	extra := make(map[string]string)
	extra["pos"] = strconv.Itoa(pos)
	err, rsdata := client.http_get("view", key, extra)

	if "HTTPSQS_ERROR" == rsdata.Data || err != nil {
		if err == nil {
			err = errors.New(string(rsdata.Data))
		}
		rsdata.Data = ""
	}
	return rsdata.Data, err
}

func (client *Client) Reset(key string) bool{

	err, rsdata := client.http_get("reset", key, nil)

	if "HTTPSQS_RESET_OK" != rsdata.Data || err != nil {
		return false

	}
	return true
}

func (client *Client) Synctime(num int) bool {

	extra := map[string]string{"num": strconv.Itoa(num)}
	err, rsdata := client.http_get("synctime", "httpsqs_synctime", extra)

	if "HTTPSQS_SYNCTIME_OK" != rsdata.Data || err != nil {

		log.Println(rsdata.Data)
		return false
	}
	return true
}

func (client *Client) Maxqueue(key string, num int) bool {

	extra := map[string]string{"num": strconv.Itoa(num)}

	err, rsdata := client.http_get("maxqueue", key, extra)

	if "HTTPSQS_MAXQUEUE_OK" != rsdata.Data || err != nil {

		log.Println(rsdata.Data)
		return false
	}
	return true
}


func (client *Client) http_get(opt string, name string, extra map[string]string) (err error, result HttpsqsResponse) {

	// build URL
	v := url.Values{}
	v.Set("auth", client.auth)
	v.Set("charset", client.charset)
	v.Set("name", name)
	v.Set("opt", opt)
	for item, val := range extra {
		v.Set(item, val)
	}
	url := fmt.Sprintf("http://%s:%d/?%s", client.host, client.port, v.Encode())

	// Send Http Request
	hc := new(http.Client)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Connection", "close")
	resp, err := hc.Do(req)
	if nil != err {
		log.Println("error:", err.Error())
		return
	}
	defer resp.Body.Close()
	defer func() {
		if (client.isDebug) {
			log.Println(url)
			log.Println(resp.Header)
			log.Printf("%+v", result.Data)
		}
		if err := recover() ; err != nil {
			log.Println("Error:", err)
		}
	}()

	// Read Response
	bt, err := ioutil.ReadAll(resp.Body)
	if pos, ok := resp.Header["Pos"]; true == ok && len(pos) > 0{
		result.Pos, _ = strconv.Atoi(pos[0])
	}
	result.Data = HttpsqsResponseString(bt)
	return
}

func (client *Client) http_post(opt string, name string, body *string, extra map[string]string) (err error, result HttpsqsResponse) {

	// build URL
	v := url.Values{}
	v.Set("auth", client.auth)
	v.Set("charset", client.charset)
	v.Set("name", name)
	v.Set("opt", opt)
	for item, val := range extra {
		v.Set(item, val)
	}
	url := fmt.Sprintf("http://%s:%d/?%s", client.host, client.port, v.Encode())

	// Send Http Request
	hc := new(http.Client)
	read := strings.NewReader(*body)
	req, _ := http.NewRequest("POST", url, read)
	req.Header.Set("Connection", "close")
	resp, err := hc.Do(req)
	if nil != err {
		log.Println("error:", err.Error())
		return
	}
	defer resp.Body.Close()
	defer func() {
		if err := recover() ; err != nil {
			log.Println("Error:", err)
		}
	}()

	// Read Response
	bt, err := ioutil.ReadAll(resp.Body)
	if pos, ok := resp.Header["Pos"]; true == ok && len(pos) > 0{
		result.Pos, _ = strconv.Atoi(pos[0])
	}
	result.Data = HttpsqsResponseString(bt)
	return
}


//=========================================================================================
//
//
//					Http persistenct connection Keep-Alive
//
//
//=========================================================================================


func (client *Client) PGet(key string) (result HttpsqsResponseString, err error) {

	err, data := client.http_pget("get", key, nil)
	result = data.Data

	if "HTTPSQS_GET_END" == data.Data || "HTTPSQS_ERROR" == data.Data || err != nil {
		result = ""
		err = errors.New(string(data.Data))
	}
	return
}

func (client *Client) PGets(key string) (result HttpsqsResponse, err error) {

	err, result = client.http_pget("get", key, nil)

	if "HTTPSQS_GET_END" == result.Data || "HTTPSQS_ERROR" == result.Data || err != nil {
		if err == nil {
			err = errors.New(string(result.Data))
		}
		result.Data = ""
	}
	return
}

func (client *Client) PPuts(key string, value string) (result HttpsqsResponseString, err error) {

	err, rsdata := client.http_ppost("put", key, &value, nil)

	if "HTTPSQS_PUT_OK" != rsdata.Data  || err != nil {

		if err == nil {
			err = errors.New(string(rsdata.Data))
		}
	}
	return rsdata.Data, err
}



//
// Post the http request with persistence coneection by POST method
// Which server is run as keep-alive, and client with command-line mode, persistence connect will be the fast choosen
//
func (client *Client) http_pget(opt string, name string, extra map[string]string) (err error, result HttpsqsResponse) {

	var out, pConnectionKey, status string
	var ContentLength, Pos int

	// build URL
	v := url.Values{}
	v.Set("auth", client.auth)
	v.Set("charset", client.charset)
	v.Set("name", name)
	v.Set("opt", opt)
	for item, val := range extra {
		v.Set(item, val)
	}

	pConnectionKey = fmt.Sprintf("%s:%d", client.host, client.port)

	defer func() {
		if (client.isDebug) {
			for _, n := range strings.Split(out, "\r\n") {
				log.Println("==> " + n )
			}
			log.Println("=== " + status)
			log.Println("<== " + result.Data)
		}

		if err != nil {
			client.pConn[pConnectionKey].pConn.Close()
			delete(client.pConn, pConnectionKey)
		}

		if err := recover(); err != nil {
			return
		}
	}()

	hc, ok := client.pConn[pConnectionKey]
	if !ok {
		conn, connOk := net.Dial("tcp", pConnectionKey)
		if connOk != nil {
			delete(client.pConn, pConnectionKey)
			err = errors.New("Dail connect Error!!!")
			return
		}
		client.pConn[pConnectionKey] = &HttpsqsPersistenceConn{pConn : conn}
		hc = client.pConn[pConnectionKey]
	}

	out = fmt.Sprintf("GET /?%s HTTP/1.1\r\n", v.Encode());
	out += fmt.Sprintf("Host: %s\r\n", client.host)
	out += fmt.Sprintf("Connection: Keep-Alive\r\n\r\n")
	hc.pConn.Write([]byte(out))

	reader := bufio.NewReader(hc.pConn)
	tp := textproto.NewReader(reader)

	line, err := tp.ReadLine()
	f := strings.SplitN(line, " ", 3)
	if len(f) < 2{
		err = errors.New("malformed HTTP response")
		return
	}

	for {
		line, err := tp.ReadLine()
		if err != nil || line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			f := strings.SplitN(line, ": ", 2)
			ContentLength, _ = strconv.Atoi(f[1])
		}

		if strings.HasPrefix(line, "Pos: ") {
			f := strings.SplitN(line, ": ", 2)
			Pos, _ = strconv.Atoi(f[1])
		}
	}

	rd := io.LimitReader(reader, int64(ContentLength))

	// Read Response
	bt, err := ioutil.ReadAll(rd)
	result.Pos = Pos
	result.Data = HttpsqsResponseString(bt)
	return
}

//
// Post the http request with persistence coneection by POST method
// Which server is run as keep-alive, and client with command-line mode, persistence connect will be the fast choosen
//
func (client *Client) http_ppost(opt string, name string, body *string, extra map[string]string) (err error, result HttpsqsResponse) {

	// build URL
	var out, pConnectionKey, status string
	var ContentLength, Pos int

	v := url.Values{}
	v.Set("auth", client.auth)
	v.Set("charset", client.charset)
	v.Set("name", name)
	v.Set("opt", opt)
	for item, val := range extra {
		v.Set(item, val)
	}

	pConnectionKey = fmt.Sprintf("%s:%d", client.host, client.port)

	defer func() {
		if (client.isDebug) {
			for _, n := range strings.Split(out, "\r\n") {
				log.Println("==> " + n )
			}
			log.Println(status)
			log.Println("<== " + result.Data)
		}
		if err != nil {
			//client.pConn[pConnectionKey].pConn.Close()
			//delete(client.pConn, pConnectionKey)
		}

		if err := recover(); err != nil {
			return
		}
	}()


	hc, ok := client.pConn[pConnectionKey]
	if !ok {
		conn, connOk := net.DialTimeout("tcp", pConnectionKey, time.Second * time.Duration(GLOBAL_HTTPSQS_TIMEOUT))
		if connOk != nil {
			err = connOk
			return
		}
		client.pConn[pConnectionKey] = &HttpsqsPersistenceConn{pConn : conn}
		hc = client.pConn[pConnectionKey]
	}

	out = fmt.Sprintf("POST /?%s HTTP/1.1\r\n", v.Encode());
	out += fmt.Sprintf("Host: %s\r\n", client.host)
	out += fmt.Sprintf("Content-Length: %d\r\n", len(*body))
	out += fmt.Sprintf("Connection: Keep-Alive\r\n\r\n")
	outbyte := out + *body
	_, err = hc.pConn.Write([]byte(outbyte));
	if  err != nil {
		return
	}

	reader := bufio.NewReader(hc.pConn)
	tp := textproto.NewReader(reader)

	status, err = tp.ReadLine()
	f := strings.SplitN(status, " ", 3)
	if len(f) < 2 {
		err = errors.New("malformed HTTP response!!!")
		return
	}

	for {
		line, err := tp.ReadLine()

		if err != nil || line == "" {
			break
		}

		if strings.HasPrefix(line, "Content-Length: ") {
			f := strings.SplitN(line, ": ", 2)
			ContentLength, _ = strconv.Atoi(f[1])
		}

		if strings.HasPrefix(line, "Pos: ") {
			f := strings.SplitN(line, ": ", 2)
			Pos, _ = strconv.Atoi(f[1])
		}
	}

	rd := io.LimitReader(reader, int64(ContentLength))

	// Read Response
	bt, err := ioutil.ReadAll(rd)
	result.Pos = Pos
	result.Data = HttpsqsResponseString(bt)
	return
}




func (rs HttpsqsResponse)String()(string) {
	return fmt.Sprintf("%s", rs.Data);
}

