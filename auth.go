package gracenote

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

var apiURL *url.URL
var auth *Auth

var errResponseNotFound = errors.New("expected a RESPONSE element, but there were none")
var errResponseError = "response contained an error status: %s"
var errUnrecognisedStatus = "unrecognised status: %s"
var errUnhandleableStatus = fmt.Sprintf("not sure what to do with codes other than %d: got %s", http.StatusOK, "%s")
var errUnmarshalError = "error unmarshalling xml: %s\n raw xml: %s"

var register string = "REGISTER"

type Auth struct {
	XMLName  string `xml:"AUTH"`
	ClientID string `xml:"CLIENT"`
	UserID   string `xml:"USER"`
}

type Query struct {
	XMLName  string `xml:"QUERY"`
	CMD      string `xml:"CMD,attr"`
	ClientID string `xml:"CLIENT"`
}

type Queries struct {
	XMLName string `xml:"QUERIES"`
	Queries []Query
}

type Response struct {
	XMLName string `xml:"RESPONSE"`
	User    string `xml:"USER"`
	Status  string `xml:"STATUS,attr"`
}

type Responses struct {
	XMLName   string     `xml:"RESPONSES"`
	Message   string     `xml:"MESSAGE"`
	Responses []Response `xml:"RESPONSE"`
}

func generateAPIURL(clientID string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   strings.Join([]string{"c", clientID, ".web.cddbp.net"}, ""),
		Path:   filepath.Join("webapi", "xml", "1.0")}
}

// Clients must call Authenticate or Register before any API calls can be made

// Authenticate sets internal identification keys for your program
func Authenticate(clientID, clientIDTag, userID string) {
	apiURL = generateAPIURL(clientID)
	auth = &Auth{
		ClientID: strings.Join([]string{clientID, clientIDTag}, "-"),
		UserID:   userID}
}

// Register registers for a userID with gracenote
func Register(clientID, clientIDTag string) (string, error) {
	apiURL = generateAPIURL(clientID)
	query := Query{CMD: register, ClientID: clientID}
	resps, err := post(Queries{Queries: []Query{query}})
	if err != nil {
		return "", err
	}
	if len(resps.Responses) == 0 {
		return "", errResponseNotFound
	}
	switch resps.Responses[0].Status {
	case "ERROR":
		return "", fmt.Errorf(errResponseError, resps.Message)
	case "OK":
		Authenticate(clientID, clientIDTag, resps.Responses[0].User)
		return resps.Responses[0].User, nil
	default:
		return "", fmt.Errorf(errUnrecognisedStatus, resps.Responses[0].Status)
	}
}

var post = func(q Queries) (Responses, error) {
	var buf bytes.Buffer
	b, err := xml.Marshal(q)
	if err != nil {
		return Responses{}, err
	}
	_, err = buf.Write(b)
	if err != nil {
		return Responses{}, err
	}
	resp, err := http.Post(apiURL.String(), "application/xml", &buf)
	if err != nil {
		return Responses{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Responses{}, fmt.Errorf(errUnhandleableStatus, resp.Status)
	}
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return Responses{}, err
	}
	var r Responses
	fmt.Println(buf.String())
	err = xml.Unmarshal([]byte(buf.String()), &r)
	if err != nil {
		return Responses{}, fmt.Errorf(errUnmarshalError, err, buf.String())
	}
	return r, nil
}
