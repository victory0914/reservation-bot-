package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var cookies string
var localStorage string
var userAgent string

var bookingConfirmUrl string
var calendarUrl string

func allFilesFreshAndHaveContent(files []string, maxAgeSeconds int64) bool {
	now := time.Now()
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			return false
		}
		// Check file is not empty
		if info.Size() == 0 {
			return false
		}
		// Check file is fresh (modified within maxAgeSeconds)
		modAge := now.Sub(info.ModTime()).Seconds()
		if modAge > float64(maxAgeSeconds) {
			return false
		}
	}
	return true
}

func LoadData() {
	cookies = ""
	localStorage = ""
	userAgent = ""
	log.Println("Loading data...")
	log.Println("Reading cookies.json")
	cookieFile, err := os.Open("../files/cookies.json")
	if err !=nil {
		log.Fatal(err)
	}
	defer cookieFile.Close()

	scanner := bufio.NewScanner(cookieFile)
	for scanner.Scan() {
		cookies += scanner.Text()
	}
	log.Println("Cookies loaded successfully")
    
	log.Println("Reading localStorage.json")

	localFile, err := os.Open("../files/localStorage.json")
	if err !=nil {
		log.Fatal(err)
	}
	defer localFile.Close()

	scanner = bufio.NewScanner(localFile)
	for scanner.Scan() {
		localStorage += scanner.Text()
	}
	log.Println("localStorage loaded successfully")

	log.Println("Reading user_agent.txt")
	uaFile, err := os.ReadFile("../files/user_agent.txt")
	if err != nil {
		log.Fatal(err)
	}
	userAgent = string(uaFile)

	log.Println("User Agent loaded successfully")

	log.Println("Data loading phase completed")
}

func main() {
	
	bookingConfirmUrl ="https://yoyaku.cityheaven.net/Confirm/ConfirmList/niigata/A1501/A150101/arabiannight"	
	calendarUrl ="https://www.cityheaven.net/niigata/A1501/A150101/arabiannight/A6ShopReservation/?girl_id=52809022"
	files := []string{"../files/cookies.json", "../files/localStorage.json", "../files/user_agent.txt"}
	maxAgeSeconds := int64(50) // 2 minutes freshness window
	log.Println("Polling for files to be freshly populated (within last 2 minutes)...")
	for {
		if allFilesFreshAndHaveContent(files, maxAgeSeconds) {
			log.Println("All files are fresh and have content. Proceeding to load data.")
			break
		}
		log.Println("Waiting for files to be freshly populated...")
		time.Sleep(2 * time.Second)
	}
	LoadData()
	log.Println("Data loaded successfully. Cookies length:", len(cookies), "localStorage length:", len(localStorage), "User Agent length:", len(userAgent))

	res, err := http.Post(calendarUrl, "application/json", nil)
	if err != nil {
		log.Fatal("Error making POST request:", err)

	}
	defer res.Body.Close()
	log.Println("POST request to booking confirm URL completed with status:", res.Status)
	bodyBytes, err  := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Error reading response body:", err)
	}
	log.Println("Response body: ", string(bodyBytes))

}