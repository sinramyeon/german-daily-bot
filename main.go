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
    
    // 1. 단어 랜덤 선택
    words := selectDailyWords()

	// 2. 오늘의 명언 선택
	wiseSentence := selectDailySentence()
    
    // 2. 메시지 포맷팅
    message := formatMessage(words, wiseSentence)
    
    // 3. 텔레그램 전송
    sendToTelegram(botToken, chatID, message)
}

func selectDailyWords() []Word {
    a1File, err := os.ReadFile("vocabulary/a1_words.json")
    if err != nil {
        panic(err)
    }

    a2File, err := os.ReadFile("vocabulary/a2_words.json")
    if err != nil {
        panic(err)
    }

    b1File, err := os.ReadFile("vocabulary/b1_words.json")
    if err != nil {
        panic(err)
    }

    var allWords []Word
    var a1Words, a2Words, b1Words []Word

    if err := json.Unmarshal(a1File, &a1Words); err != nil {
        panic(err)
    }
    if err := json.Unmarshal(a2File, &a2Words); err != nil {
        panic(err)
    }
    if err := json.Unmarshal(b1File, &b1Words); err != nil {
        panic(err)
    }
 
    rand.Seed(time.Now().UnixNano())

    // a1에서 3개, a2에서 3개, b1에서 4개 선택
    rand.Shuffle(len(a1Words), func(i, j int) { a1Words[i], a1Words[j] = a1Words[j], a1Words[i] })
    rand.Shuffle(len(a2Words), func(i, j int) { a2Words[i], a2Words[j] = a2Words[j], a2Words[i] })
    rand.Shuffle(len(b1Words), func(i, j int) { b1Words[i], b1Words[j] = b1Words[j], b1Words[i] })
    allWords = append(allWords, a1Words[:3]...)
    allWords = append(allWords, a2Words[:3]...)
    allWords = append(allWords, b1Words[:4]...)

    return allWords
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
        msg += fmt.Sprintf(" [%s] *%d. %s*\n", word.Level, i+1, word.German)
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
