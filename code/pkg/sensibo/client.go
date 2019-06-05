package sensibo

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/jinzhu/copier"
)

// Client is a Sensibo Sky's client data
type Client struct {
	BaseURL    *url.URL
	State      Pod
	origState  Pod
	httpClient *http.Client
	apiKey     string
}

// NewClient returns an authenticated Sensibo Client.
// State is not yet loaded, call LoadState() to complete initialization
func NewClient(httpClient *http.Client) *Client {
	c := &Client{}

	if httpClient == nil {
		c.httpClient = &http.Client{Timeout: time.Second * 10}
	}

	var err error
	c.BaseURL, err = url.Parse("https://home.sensibo.com/api/v2/")

	if err != nil {
		log.Fatal(err)
	}

	c.loadApiKey()
	return c
}

// loadApiKey will fetch the API key from the environment or AWS's SSM
// Panics on failure to do so
func (c *Client) loadApiKey() {

	// try to load it from the environment
	var ok bool
	c.apiKey, ok = os.LookupEnv("SENSIBO_API_KEY")
	if !ok {
		log.Fatal("Need SENSIBO_API_KEY set")
	}
	if strings.HasPrefix(c.apiKey, "ssm:") {
		path := strings.TrimPrefix(c.apiKey, "ssm:")
		sess, err := session.NewSession(aws.NewConfig())
		if err != nil {
			log.Fatal(err)
		}

		withDecryption := true
		svc := ssm.New(sess)
		req := ssm.GetParameterInput{Name: &path, WithDecryption: &withDecryption}
		resp, err := svc.GetParameter(&req)
		if err != nil {
			log.Fatal(err)
		}

		c.apiKey = *resp.Parameter.Value
	}
}

// LoadState fetches remote state from the Sensibo service
// It currently only supports a single Pod on an account
func (c *Client) LoadState() error {
	// loads the current state and stores it in the client
	req, err := c.newRequest("GET", "users/me/pods", "fields=acState,measurements,smartMode,id", nil)
	if err != nil {
		return err
	}
	podList := &PodList{}
	_, err = c.do(req, podList)
	if err == nil {
		// TODO make this work with more than one device
		// log.Printf("Setting state:\n%v\n", podList.Pods[0])
		// Make 2 copies of the original state.
		// This means we can 'diff' them to see if changes need to be synced
		copier.Copy(&c.State, &podList.Pods[0])
		copier.Copy(&c.origState, &podList.Pods[0])

	}
	return err
}

// floatEq checks if floats are (approximately) equivalent
func floatEq(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

// Equivalent checks if 2 SmartMode structs are (approximately) equivalent
func (sm *SmartMode) Equivalent(other *SmartMode) bool {
	if !floatEq(sm.LowTemperatureThreshold, other.LowTemperatureThreshold) {
		log.Println("Low Threshold differed")
		return false
	}
	if !floatEq(sm.HighTemperatureThreshold, other.HighTemperatureThreshold) {
		log.Println("High Threshold differed")
		return false
	}
	if sm.HighTemperatureState != other.HighTemperatureState {
		log.Println("High Temperature States differed")
		return false
	}
	if sm.LowTemperatureState != other.LowTemperatureState {
		log.Println("Low Temperature States differed")
		return false
	}
	return true

}

// PushState submits the state to the Sensibo API only if it's changed since being fetched
func (c *Client) PushState() error {

	if !c.State.SmartMode.Equivalent(&c.origState.SmartMode) {
		log.Println("SmartMode changed, pushing it")
		err := c.updateSmartMode()
		if err != nil {
			return err
		}
	} else {
		log.Println("SmartMode config unchanged, not pushing")
	}

	if c.State.AcState != c.origState.AcState {
		log.Println("AC State changed, pushing it")
		err := c.updateAcState()
		if err != nil {
			return err
		}
	} else {
		log.Println("AC State unchanged, not pushing")
	}

	return nil
}

// updateAcState formats and pushes the acstate resource
func (c *Client) updateAcState() error {
	marshaled, err := json.Marshal(c.State.AcState)
	if err != nil {
		return err
	}
	// Sensibo wants a strange format for this specific POST body
	sensiboed := fmt.Sprintf("{\"acState\": %s}\n", string(marshaled))
	req, err := c.newRequest("POST", fmt.Sprintf("pods/%s/acStates", c.State.ID), "", strings.NewReader(sensiboed))
	if err != nil {
		return err
	}
	_, err = c.do(req, nil)
	return err
}

// updateSmartMode formats and pushes the SmartMode resource
func (c *Client) updateSmartMode() error {
	marshaled, err := json.MarshalIndent(c.State.SmartMode, "", "  ")
	if err != nil {
		return err
	}

	req, err := c.newRequest("POST", fmt.Sprintf("pods/%s/smartmode", c.State.ID), "", strings.NewReader(string(marshaled)))

	if err != nil {
		return err
	}
	_, err = c.do(req, nil)
	return err
}

// newRequest creates a new request with a given path, body and query parameters
// Adds authentication information
func (c *Client) newRequest(method, path, query string, body io.Reader) (*http.Request, error) {

	// add Base URL and API key to path
	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)

	q, _ := url.ParseQuery(query)
	q.Add("apiKey", c.apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// do calls an HTTP request and deserializes the response
// it logs full request/response in case of error and returns error codes
func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	dump, _ := httputil.DumpRequestOut(req, true)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseData, err := ioutil.ReadAll(resp.Body)
	responseString := string(responseData)

	if err != nil {

		log.Printf("Request:\n%s\n", string(dump))
		log.Printf("Response:\n%s\n", responseString)
		return resp, err
	}

	if v != nil {
		err = json.Unmarshal(responseData, v)
		if err != nil {
			log.Printf("got parse error: %v\n", err)
			log.Printf("Request:\n%s\n", string(dump))
			log.Printf("Response:\n%s\n", responseString)
		}
	}
	return resp, err
}
