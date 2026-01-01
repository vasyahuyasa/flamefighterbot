package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

var (
	telegramBotToken = ""
	openAIToken      = ""
	openAIChatModel  = ""
	openAIBaseURL    = ""
)

const defaultPrompt = `
Ты — SRE. Классифицируешь сообщения чата: нужен ли инцидент.

Отвечай ТОЛЬКО JSON:

{
  "create_incident": boolean,
  "severity": "P1" | "P2" | "P3" | "none",
  "summary": string | null,
  "rationale": string,
  "confidence": number
}

P1 — массовая/критическая недоступность сервиса или ключевых систем.  
P2 — частичная, региональная или нестабильная работа, жалобы.  
P3 — аномалии метрик, рассинхронизация данных, подозрение на сбой.
none - нет проблем.

Создавай инцидент если:
- «не работает», «подвисает», ошибки, жалобы;
- есть URL + проблема;
- подозрение на инфраструктуру/данные.

Не создавай если:
- инфо, метрики, статусы;
- бизнес-обсуждение, эмоции;
- внешняя проблема;
- сообщение о восстановлении.

Правила:
- локально (город/страница) → минимум P2;
- сомнение ignore vs P3 → P3;
- только JSON, без текста.`

func main() {
	telegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	openAIToken = os.Getenv("OPENAI_API_KEY")
	openAIChatModel = os.Getenv("OPENAI_CHAT_MODEL")
	openAIBaseURL = os.Getenv("OPENAI_BASE_URL")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New(telegramBotToken, opts...)
	if err != nil {
		log.Fatalf("cannot start telegram bot: %v", err)
	}

	b.Start(ctx)
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}

	msg := update.Message.Text
	triageResult, err := triageMsg(update.Message.Text)
	if err != nil {
		log.Printf("cannot triage message: %v", err)
		return
	}

	log.Printf("Message: %s\n%s", msg, triageResult.ToString())

	tgMsg := &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   triageResult.ToString(),
		ReplyParameters: &models.ReplyParameters{
			MessageID: update.Message.ID,
			ChatID:    update.Message.Chat.ID,
		},
	}

	_, err = b.SendMessage(ctx, tgMsg)
	if err != nil {
		log.Printf("cannot send message: %v", err)
	}
}

type Severity string

const (
	P1      Severity = "P1" // highest
	P2      Severity = "P2"
	P3      Severity = "P3"
	None    Severity = "none"
	Unknown Severity = "unknown"
)

type TriageResult struct {
	IsIncident bool
	Severity   Severity
	Summary    string
	Rationale  string
	Confidence float32
}

func (r *TriageResult) ToString() string {
	return fmt.Sprintf("IsIncident: %t\nSeverity: %v\nSummary: %v\nRationale: %s\nConfidence: %f\n", r.IsIncident, r.Severity, r.Summary, r.Rationale, r.Confidence)
}

func triageMsg(msg string) (TriageResult, error) {
	client := openai.NewClient(
		option.WithAPIKey(openAIToken),
		option.WithBaseURL(openAIBaseURL),
	)

	params := responses.ResponseNewParams{
		Model:           shared.ChatModel(openAIChatModel),
		Temperature:     openai.Float(0.3),
		MaxOutputTokens: openai.Int(500),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(msg),
		},
		Instructions: openai.String(defaultPrompt),
	}

	r, err := client.Responses.New(context.Background(), params)
	if err != nil {
		return TriageResult{}, fmt.Errorf("cannot do request to API: %w", err)
	}

	// if err != nil {
	// 	var e *responses.Error
	// 	if errors.As(err, &e) {
	// 		log.Printf("JSON: %v", e.RawJSON())
	// 		log.Printf("Response: %v", e.Response)
	// 		log.Fatalf("responses API error: %#v", e)
	// 	}

	resp := struct {
		CreateIncident bool    `json:"create_incident"`
		Severity       string  `json:"severity"`
		Summary        *string `json:"summary"`
		Rationale      string  `json:"rationale"`
		Confidence     float32 `json:"confidence"`
	}{}

	output := strings.TrimSuffix(strings.TrimPrefix(r.OutputText(), "```"), "```")

	err = json.Unmarshal([]byte(output), &resp)
	if err != nil {
		return TriageResult{}, fmt.Errorf("cannot unmarshal model response %q: %w", r.OutputText(), err)
	}

	var severity Severity

	switch resp.Severity {
	case "P1":
		severity = P1
	case "P2":
		severity = P2
	case "P3":
		severity = P3
	case "none":
		severity = None
	default:
		severity = Unknown
	}

	summary := ""
	if resp.Summary != nil {
		summary = *resp.Summary
	}

	return TriageResult{
		IsIncident: resp.CreateIncident,
		Severity:   severity,
		Summary:    summary,
		Rationale:  resp.Rationale,
		Confidence: resp.Confidence,
	}, nil
}
