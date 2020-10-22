package main

import (
	"encoding/json"
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

func isLink(text string) string {
	U, urlParseError := url.Parse(text)
	if urlParseError != nil {
		log.Println(urlParseError)
	}
	if U.Scheme == "" {
		return ""
	}

	if U.Host == "twitter.com" {
		return "Twitter"
	} else {
		return "Unknown"
	}

}

func parseTwitterURL(url string) string {
	sp := strings.Split(url, "/")
	id := strings.Split(sp[len(sp)-1], "?")[0]
	// log.Printf("Str id : %s", id)
	return id
}

func downloadTwitter(id int64) string {

	consumerKey := os.Getenv("CONSUMER_KEY")
	consumerSecret := os.Getenv("CONSUMER_SECRET")
	accessToken := os.Getenv("ACCESS_TOKEN")
	accessSecret := os.Getenv("ACCESS_SECRET")

	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)

	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	url := ""
	// log.Printf("%v\n", client)

	tweet, _, statusError := client.Statuses.Show(id, nil)
	if statusError != nil {
		log.Println(statusError)
		url = "Could not load twitter link"
	}

	log.Printf("Tweet: %v", tweet)
	entities := tweet.ExtendedEntities

	if entities == nil {
		log.Println("No entities found.")
		return "No entities found"
	}
	// log.Printf("Entities: %v", entities)
	media := entities.Media
	// log.Printf("Media: %v", media)

	if len(media) > 0 {
		videoVariants := media[0].VideoInfo.Variants
		if len(videoVariants) > 0 {
			// find largest bitrate
			log.Printf("Found %d video variants", len(videoVariants))

			bit := -1
			for _, vid := range videoVariants {
				if vid.Bitrate > bit {
					bit = vid.Bitrate
					url = vid.URL
				}
			}
			log.Printf("Largest bitrate : %d\n", bit)
		} else {
			log.Println("No videos")
			url = "No videos found"
		}
	} else {
		log.Println("No Media")
		url = "No media found"
	}

	return url
}

func update(w http.ResponseWriter, r *http.Request) {
	token := os.Getenv("TOKEN")
	URL := "https://api.telegram.org/bot" + token + "/"

	payload := url.Values{}

	jsonMap := make(map[string](interface{}))
	byteBody, bodyParseError := ioutil.ReadAll(r.Body)
	if bodyParseError != nil {
		log.Println(bodyParseError)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error in Parsing: " + bodyParseError.Error()))
		return
	}
	err := json.Unmarshal([]byte(byteBody), &jsonMap)
	if err != nil {
		log.Println("Error in json: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error in json: " + err.Error()))
		return
	}

	log.Printf("INFO: jsonMap, %s", jsonMap)

	messageMap := jsonMap["message"].(map[string]interface{})
	chatMap := messageMap["chat"].(map[string]interface{})

	chatID := chatMap["id"].(float64)
	chatIDInt := int64(chatID)
	log.Printf("chat_id: %d", chatIDInt)
	chatIDStr := strconv.FormatInt(chatIDInt, 10)
	log.Printf("chat_id_str : %s", chatIDStr)

	payload.Set("chat_id", chatIDStr)
	text := messageMap["text"].(string)
	log.Println(text)

	site := isLink(text)

	if site == "Twitter" {
		id, parseError := strconv.ParseInt(parseTwitterURL(text), 10, 64)
		if parseError != nil {
			log.Println(parseError)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Could not convert text to link; parser error : " + parseError.Error()))
		}
		log.Println(id)
		url := downloadTwitter(id)
		log.Println(url)
		payload.Set("text", url)
		resp, messageError := http.PostForm(URL+"sendMessage", payload)
		if messageError != nil {
			log.Println(messageError)
			w.WriteHeader(http.StatusExpectationFailed)
			w.Write([]byte("Could not send message to user: " + messageError.Error()))
		}
		log.Println(resp.StatusCode)
		// log.Printf("Output URL : %s", url)
	} else {
		w.Write([]byte("No link detected. Link must be a valid twitter link with embedded video."))
	}

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
