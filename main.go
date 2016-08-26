package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	PAGE_TOKEN = ""
	AUTH_TOKEN = ""
)

func main() {
	http.HandleFunc("/webhook", MessengerVerify)

	fmt.Println("Starting server on :9090")
	log.Fatal(http.ListenAndServe(":9091", nil))
}

func MessengerVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" { // {{{
		challenge := r.URL.Query().Get("hub.challenge")
		verify_token := r.URL.Query().Get("hub.verify_token")

		if len(verify_token) > 0 && len(challenge) > 0 && verify_token == "developers-are-gods" {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, challenge)
			return
		} // }}}
	} else if r.Method == "POST" {
		defer r.Body.Close()

		input := new(MessengerInput)
		if err := json.NewDecoder(r.Body).Decode(input); err == nil {

			//lets swap sender and recipient
			reply := input.Entry[0].Messaging[0]
			reply.Sender, reply.Recipient = reply.Recipient, reply.Sender

			if resp, err := getApiAiResponse(*input); err == nil {
				reply.Message.Text = resp
				reply.Message.Seq = 0 //these fields are not used so remove them with omit empty
				reply.Message.Mid = ""

				b, _ := json.Marshal(reply)
				url := fmt.Sprintf("https://graph.facebook.com/v2.6/me/messages?access_token=%s", PAGE_TOKEN)
				http.Post(url,
					"application/json",
					bytes.NewReader(b))
			}
			return
		}
	}
	// {{{
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(400)
	fmt.Fprintf(w, "Bad Request") // }}}
}

type MessengerInput struct {
	Entry []struct {
		Time      uint64 `json:"time,omitempty"`
		Messaging []struct {
			Sender struct {
				Id string `json:"id"`
			} `json:"sender,omitempty"`
			Recipient struct {
				Id string `json:"id"`
			} `json:"recipient,omitempty"`
			Timestamp uint64 `json:"timestamp,omitempty"`
			Message   *struct {
				Mid  string `json:"mid,omitempty"`
				Seq  uint64 `json:"seq,omitempty"`
				Text string `json:"text"`
			} `json:"message,omitempty"`
		} `json:"messaging"`
	}
}

type ApiAiInput struct {
	Status struct {
		Code      int
		ErrorType string
	}
	Result struct {
		Action           *string
		ActionIncomplete bool
		Speech           string
	} `json:"result"`
}

func getApiAiResponse(m MessengerInput) (resp string, err error) {
	params := url.Values{}
	params.Add("query", m.Entry[0].Messaging[0].Message.Text)
	params.Set("sessionId", m.Entry[0].Messaging[0].Sender.Id)

	url := fmt.Sprintf("https://api.api.ai/v1/query?V=20160518&lang=En&%s", params.Encode())
	ai, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	ai.Header.Set("Authorization", "Bearer "+AUTH_TOKEN)

	if resp, err := http.DefaultClient.Do(ai); err != nil {
		return "", err
	} else {
		defer resp.Body.Close()

		var input ApiAiInput
		datastring, _ := ioutil.ReadAll(resp.Body)
		err := json.NewDecoder(strings.NewReader(string(datastring))).Decode(&input)
		if err != nil {
			return "", err
		}

		return input.Result.Speech, nil
	}
}
