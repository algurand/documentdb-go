package documentdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"io"
	"net/http"
	"strings"
)

type Clienter interface {
	Read(link string, ret interface{}) error
	Delete(link string) error
	Query(link string, query string, ret interface{}) error
	Create(link string, body, ret interface{}) error
	Replace(link string, body, ret interface{}) error
	Execute(link string, body, ret interface{}) error
}

type Client struct {
	Url    string
	Config Config
	http.Client
}

// Read resource by self link
func (c *Client) Read(link string, ret interface{}) error {
	return c.method("GET", link, http.StatusOK, ret, &bytes.Buffer{})
}

// Delete resource by self link
func (c *Client) Delete(link string) error {
	return c.method("DELETE", link, http.StatusNoContent, nil, &bytes.Buffer{})
}

// Query resource
func (c *Client) Query(link, query string, ret interface{}) error {
	buf := bytes.NewBufferString(querify(query))
	req, err := http.NewRequest("POST", path(c.Url, link), buf)
	if err != nil {
		return err
	}
	r := ResourceRequest(link, req, c.Config)
	if err = r.DefaultHeaders(c.Config.MasterKey); err != nil {
		return err
	}
	r.QueryHeaders(buf.Len())
	return c.do(r, http.StatusOK, ret)
}

// Create resource
func (c *Client) Create(link string, body, ret interface{}) error {
	data, err := stringify(body)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(data)
	return c.method("POST", link, http.StatusCreated, ret, buf)
}

// Replace resource
func (c *Client) Replace(link string, body, ret interface{}) error {
	data, err := stringify(body)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(data)
	return c.method("PUT", link, http.StatusOK, ret, buf)
}

// Replace resource
// TODO: DRY, move to methods instead of actions(POST, PUT, ...)
func (c *Client) Execute(link string, body, ret interface{}) error {
	data, err := stringify(body)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(data)
	return c.method("POST", link, http.StatusOK, ret, buf)
}

// Private generic method resource
func (c *Client) method(method, link string, status int, ret interface{}, body *bytes.Buffer) (err error) {
	log.Println("1")
	req, err := http.NewRequest(method, path(c.Url, link), body)
	log.Println("2")
	if err != nil {
		return err
	}
	r := ResourceRequest(link, req, c.Config)
	log.Println("3")
	if err = r.DefaultHeaders(c.Config.MasterKey); err != nil {
		return err
	}
	log.Println("4")
	return c.do(r, status, ret)
}

// Private Do function, DRY
func (c *Client) do(r *Request, status int, data interface{}) error {
	r.Header["x-ms-max-item-count"] = []string{"1000"}
	resp, err := c.Do(r.Request)
	log.Println("5")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != status {
		err1 := RequestError{}
		log.Println("7", resp.StatusCode)
		if err = readJson(resp.Body, &err1); err != nil{
			log.Println("error")
			log.Println(err)
		}
		err = &RequestError{
			Code:    err1.Code,
			Message: err1.Message,
			Header:  resp.Header,
		}
		return err
	}
	
	if data == nil {
		return nil
	}
	log.Println("6")
	return readJson(resp.Body, data)
}

// Generate link
func path(url string, args ...string) (link string) {
	args = append([]string{url}, args...)
	link = strings.Join(args, "/")
	return
}

// Read json response to given interface(struct, map, ..)
func readJson(reader io.Reader, data interface{}) error {
	return json.NewDecoder(reader).Decode(&data)
}

// Stringify query-string as documentdb expected
func querify(query string) string {
	return fmt.Sprintf(`{ "%s": "%s" }`, "query", query)
}

// Stringify body data
func stringify(body interface{}) (bt []byte, err error) {
	switch t := body.(type) {
	case string:
		bt = []byte(t)
	case []byte:
		bt = t
	default:
		bt, err = json.Marshal(t)
	}
	return
}
