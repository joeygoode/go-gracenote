package gracenote

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_auth_marshal(t *testing.T) {
	a := Auth{
		ClientID: "client_id_string",
		UserID:   "user_id_string"}
	b, err := xml.MarshalIndent(&a, "", "  ")
	if assert.NoError(t, err) {
		assert.Equal(t, `<AUTH>
  <CLIENT>client_id_string</CLIENT>
  <USER>user_id_string</USER>
</AUTH>`, string(b))
	}
}

func Test_Register(t *testing.T) {
	expectedQuery := Queries{
		Queries: []Query{
			Query{
				CMD:      register,
				ClientID: "0"}}}
	sampleResps := []Responses{
		Responses{
			Message: "",
			Responses: []Response{
				Response{
					Status: "OK",
					User:   "123456"}}},
		Responses{
			Message: "Some error message",
			Responses: []Response{
				Response{
					Status: "ERROR",
					User:   ""}}},
		Responses{
			Message: "",
			Responses: []Response{
				Response{
					Status: "NO MATCH",
					User:   ""}}},
		Responses{
			Message:   "",
			Responses: []Response{}},
		Responses{}}
	sampleErrors := []error{
		nil,
		nil,
		nil,
		nil,
		assert.AnError}
	userIDs := []string{"123456", "", "", "", "", ""}
	errorFuncs := []func(assert.TestingT, error, ...interface{}) bool{
		assert.NoError,
		assert.Error,
		assert.Error,
		assert.Error,
		assert.Error}
	oldPost := post
	defer func() {
		post = oldPost
		apiURL = nil
		auth = nil
	}()
	for i := range sampleResps {
		post = func(query Queries) (Responses, error) {
			assert.Equal(t, query, expectedQuery)
			return sampleResps[i], sampleErrors[i]
		}
		userID, err := Register("0", "0")
		if errorFuncs[i](t, err) {
			assert.Equal(t, userID, userIDs[i])
		}
	}
}

func Test_generateAPIURL(t *testing.T) {
	url := generateAPIURL("123456")
	assert.Equal(t, "c123456.web.cddbp.net", url.Host)
}

func Test_post(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r.Body)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.Equal(t, `<QUERIES><QUERY CMD="REGISTER"><CLIENT>client_id_string</CLIENT></QUERY></QUERIES>`,
			buf.String())
		w.Write([]byte(`<RESPONSES><RESPONSE STATUS="OK"><USER>user_id_string</USER></RESPONSE></RESPONSES>`))
	}))
	defer ts.Close()
	var err error
	apiURL, err = url.Parse(ts.URL)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer func() {
		apiURL = nil
	}()
	q := Queries{Queries: []Query{Query{
		CMD:      register,
		ClientID: "client_id_string"}}}
	resps, err := post(q)
	if assert.NoError(t, err) {
		assert.Equal(t, Responses{Responses: []Response{Response{
			Status: "OK",
			User:   "user_id_string"}}}, resps)
	}
}
