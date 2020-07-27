package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/joho/godotenv"
)

func is_link(text string) string {
	U, _ := url.Parse(text)
	if U.Scheme == "" {
		return ""
	} else {
		if U.Host == "twitter.com" {
			return "Twitter"
		} else if U.Host == "v.redd.it" {
			return "Reddit"
		} else {
			return "Unknown"
		}
	}
}

func parse_twitter_url(url string) string {
	sp := strings.Split(url, "/")
	id := strings.Split(sp[len(sp)-1], "?")[0]
	log.Printf("Str id : %s", id)
	return id
}

func download_twitter(id int64) string {

	consumerKey := os.Getenv("CONSUMER_KEY")
	consumerSecret := os.Getenv("CONSUMER_SECRET")
	accessToken := os.Getenv("ACCESS_TOKEN")
	accessSecret := os.Getenv("ACCESS_SECRET")

	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)

	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	tweet, _, _ := client.Statuses.Show(id, nil)

	// log.Println(tweet.ExtendedEntities.Media)
	media := tweet.ExtendedEntities.Media
	bit := -1
	url := ""

	if len(media) > 0 {
		videoVariants := media[0].VideoInfo.Variants
		if len(videoVariants) > 0 {
			// find largest bitrate

			for _, vid := range videoVariants {
				if vid.Bitrate > bit {
					bit = vid.Bitrate
					url = vid.URL
				}
			}
		} else {
			log.Println("No videos")
		}
	} else {
		log.Println("No Media")
	}

	return url
}

func update(w http.ResponseWriter, r *http.Request) {
	token := os.Getenv("TOKEN")
	URL := "https://api.telegram.org/bot" + token + "/"

	payload := url.Values{}

	jsonMap := make(map[string](interface{}))
	byteBody, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal([]byte(byteBody), &jsonMap)
	if err != nil {
		log.Println("Error in json")
	}

	log.Printf("INFO: jsonMap, %s", jsonMap)

	messageMap := jsonMap["message"].(map[string]interface{})
	chatMap := messageMap["chat"].(map[string]interface{})
	chatID := chatMap["id"].(float64)
	log.Printf("chat_id: %f", chatID)
	payload.Set("chat_id", strconv.FormatFloat(chatID, 'E', -1, 64))
	text := messageMap["text"].(string)

	site := is_link(text)

	if site == "Twitter" {
		id, _ := strconv.ParseInt(parse_twitter_url(text), 10, 64)
		url := download_twitter(id)
		log.Printf("Output URL : %s", url)
		payload.Set("text", url)
		http.PostForm(URL+"sendMessage", payload)
	} else if site == "Reddit" {
		// do something
	}

	io.WriteString(w, "hello world")

}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	PORT := os.Getenv("PORT")

	if PORT == "" {
		PORT = "5000"
	}

	http.HandleFunc("/update", update)
	http.ListenAndServe(":"+PORT, nil)

}
