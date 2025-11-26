package main

import (
    "encoding/json"
    "fmt"
    "math/rand"
    "net/http"
   // "net/url"
    "os"
    "os/exec"
    "time"
)

type Word struct {
    German   string   `json:"german"`
    English  string   `json:"english"`
    Level    string   `json:"level"`
    Examples []string `json:"examples"`
    Synonyms []string `json:"synonyms"`
    Antonyms []string `json:"antonyms"`
}

type WiseSentences struct {
    German  string `json:"german"`
    English string `json:"english"`
}

const chatIDFile = "chat_ids.json"

func main() {
    botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

    // 1. /start 누른 사용자 새로 불러오기
    newIDs := fetchNewChatIDs(botToken)
    mergeChatIDs(newIDs)

    // 2. 단어/명언 선택
    words := selectDailyWords()
    sentence := selectDailySentence()
    message := formatMessage(words, sentence)

    // 3. 모든 사용자에게 전송
    chatIDs := loadChatIDs()
    for _, id := range chatIDs {
        sendToTelegram(botToken, id, message)
    }
}

// ---------------- getUpdates로 /start 감지 ----------------
func fetchNewChatIDs(botToken string) []string {
    apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", botToken)
    resp, err := http.Get(apiURL)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    var result struct {
        Result []struct {
            Message struct {
                Chat struct {
                    ID int64 `json:"id"`
                } `json:"chat"`
                Text string `json:"text"`
            } `json:"message"`
        } `json:"result"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    var newIDs []string
    for _, update := range result.Result {
        if update.Message.Text == "/start" {
            newIDs = append(newIDs, fmt.Sprintf("%d", update.Message.Chat.ID))
        }
    }
    return newIDs
}

// ---------------- chat_ids.json 관리 ----------------
func loadChatIDs() []string {
    if _, err := os.Stat(chatIDFile); os.IsNotExist(err) {
        return []string{}
    }
    data, _ := os.ReadFile(chatIDFile)
    var ids []string
    json.Unmarshal(data, &ids)
    return ids
}

func mergeChatIDs(newIDs []string) {
    ids := loadChatIDs()
    idMap := make(map[string]bool)
    for _, id := range ids {
        idMap[id] = true
    }
    for _, id := range newIDs {
        if !idMap[id] {
            ids = append(ids, id)
        }
    }

    data, _ := json.Marshal(ids)
    os.WriteFile(chatIDFile, data, 0644)

    // ------------------ Repo에 자동 커밋 & push ------------------
    
    cmd := exec.Command("git", "add", chatIDFile)
    if err := cmd.Run(); err != nil {
        fmt.Println("git add failed:", err)
    }

    cmd = exec.Command("git", "commit", "-m", "Update chat_ids.json [skip ci]")
    if err := cmd.Run(); err != nil {
        fmt.Println("git commit failed:", err)
    }

    cmd = exec.Command("git", "push")
    if err := cmd.Run(); err != nil {
        fmt.Println("git push failed:", err)
    }
}

// ---------------- 단어/명언 선택 ----------------
func selectDailyWords() []Word {
    a1File, _ := os.ReadFile("vocabulary/a1_words.json")
    a2File, _ := os.ReadFile("vocabulary/a2_words.json")
    b1File, _ := os.ReadFile("vocabulary/b1_words.json")

    var a1Words, a2Words, b1Words []Word
    json.Unmarshal(a1File, &a1Words)
    json.Unmarshal(a2File, &a2Words)
    json.Unmarshal(b1File, &b1Words)

    rand.Seed(time.Now().UnixNano())
    rand.Shuffle(len(a1Words), func(i, j int) { a1Words[i], a1Words[j] = a1Words[j], a1Words[i] })
    rand.Shuffle(len(a2Words), func(i, j int) { a2Words[i], a2Words[j] = a2Words[j], a2Words[i] })
    rand.Shuffle(len(b1Words), func(i, j int) { b1Words[i], b1Words[j] = b1Words[j], b1Words[i] })

    return append(append(a1Words[:3], a2Words[:3]...), b1Words[:4]...)
}

func selectDailySentence() WiseSentences {
    file, _ := os.ReadFile("vocabulary/sentences.json")
    var sentences []WiseSentences
    json.Unmarshal(file, &sentences)
    rand.Seed(time.Now().UnixNano())
    return sentences[rand.Intn(len(sentences))]
}

// ---------------- 메시지 포맷 ----------------
func formatMessage(words []Word, sentence WiseSentences) string {
    msg := "🇩🇪 *Today's German Study* 🇩🇪\n\n"
    for i, word := range words {
        msg += fmt.Sprintf("[%s] *%d. %s*\n", word.Level, i+1, word.German)
        msg += fmt.Sprintf("📖 %s\n\n", word.English)
        for _, ex := range word.Examples {
            msg += fmt.Sprintf("💬 %s\n\n", ex)
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

// ---------------- 텔레그램 전송 ----------------
func sendToTelegram(botToken, chatID, message string) {
    // apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
    // data := url.Values{}
    // data.Set("chat_id", chatID)
    // data.Set("text", message)
    // data.Set("parse_mode", "Markdown")
    // http.PostForm(apiURL, data)
}
