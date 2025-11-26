package main

import (
    "encoding/json"
    "fmt"
    "math/rand"
    "net/http"
    "net/url"
    "os"
    "time"
)

type Gender string

const (
    Maskullin Gender = "Maskullin"
    Feminin 	Gender = "Feminin"
    Neutral Gender = "Neutral"
)

type Word struct {
    German    string   `json:"german"`
    English   string   `json:"english"`
	Gender Gender 	`json:"gender"`  
    Level     string   `json:"level"`
    Examples  []string `json:"examples"`
    Synonyms  []string `json:"synonyms"`
    Antonyms  []string `json:"antonyms"`
}

type WiseSentences struct {
	German string `json:"german"`
	English string `json:"english"`
}

func main() {
    botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
    chatID := os.Getenv("TELEGRAM_CHAT_ID")
    
    // 1. 단어 5개 랜덤 선택
    words := selectDailyWords(5)

	// 2. 오늘의 명언 선택
	wiseSentence := selectDailySentence()
    
    // 2. 메시지 포맷팅
    message := formatMessage(words, wiseSentence)
    
    // 3. 텔레그램 전송
    sendToTelegram(botToken, chatID, message)
}

func selectDailyWords(count int) []Word {
    file, err := os.ReadFile("vocabulary/words.json")
    if err != nil {
        panic(err)
    }

    var allWords []Word
    if err := json.Unmarshal(file, &allWords); err != nil {
        panic(err)
    }

    rand.Seed(time.Now().UnixNano())

    // count보다 짧을 경우 전체 반환
    if len(allWords) <= count {
        return allWords
    }

    // 랜덤 셔플 후 앞 count개 반환
    rand.Shuffle(len(allWords), func(i, j int) {
        allWords[i], allWords[j] = allWords[j], allWords[i]
    })

    return allWords[:count]
}

func selectDailySentence() WiseSentences {
    file, err := os.ReadFile("vocabulary/sentences.json")
    if err != nil {
        panic(err)
    }

    var sentences []WiseSentences
    if err := json.Unmarshal(file, &sentences); err != nil {
        panic(err)
    }

    rand.Seed(time.Now().UnixNano())
    return sentences[rand.Intn(len(sentences))]
}


func formatMessage(words []Word, sentence WiseSentences) string {
    msg := "🇩🇪 *Today's German Study* 🇩🇪\n\n"
    
    for i, word := range words {
        msg += fmt.Sprintf("*%d. %s* (%s)\n", i+1, word.German, word.Level)
        msg += fmt.Sprintf("📖 %s\n\n", word.English)
        
        // 예문 3개
        for _, example := range word.Examples {
            msg += fmt.Sprintf("💬 %s\n\n", example)
        }
        
        if len(word.Synonyms) > 0 {
            msg += fmt.Sprintf("🔄 Synonyms: %v\n\n", word.Synonyms)
        }
        if len(word.Antonyms) > 0 {
            msg += fmt.Sprintf("🔀 Antonyms: %v\n\n", word.Antonyms)
        }
        msg += "\n---\n\n"
    }

	msg += "💡 *Wise Sentence of the Day*\n\n"
	msg += fmt.Sprintf("🇩🇪 %s\n", sentence.German)
	msg += fmt.Sprintf("🇬🇧 %s\n", sentence.English)

    
    return msg
}

func sendToTelegram(botToken, chatID, message string) {
    apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
    
    data := url.Values{}
    data.Set("chat_id", chatID)
    data.Set("text", message)
    data.Set("parse_mode", "Markdown")
    
    http.PostForm(apiURL, data)
}
