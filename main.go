package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const defaultPrompt = `
Ты — SRE. Классифицируешь сообщения чата: нужен ли инцидент.

Отвечай ТОЛЬКО JSON:

{
  "create_incident": boolean,
  "severity": "P1" | "P2" | "P3" | null,
  "summary": string | null,
  "rationale": string,
  "confidence": number
}

P1 — массовая/критическая недоступность сервиса или ключевых систем.  
P2 — частичная, региональная или нестабильная работа, жалобы.  
P3 — аномалии метрик, рассинхронизация данных, подозрение на сбой.

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
	token := os.Getenv("BOT_TOKEN")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		panic(err)
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
		log.Printf("cannot triage msg: %v", err)
		return
	}

	printTriageInfo(msg, triageResult)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}

type Severity string

const (
	P1 Severity = "P1" // highest
	P2 Severity = "P2"
	P3 Severity = "P3"
)

type TriageResult struct {
	IsIncident bool
	Severity   *Severity
	Summary    *string
	Rationale  string
	Confidence float32
}

func printTriageInfo(msg string, r TriageResult) {
	fmt.Printf("Message: %s\nIsIncident: %b\nSeverity: %v\nSummary: %v\nRationale: %s\n Confidence: %f", msg, r.IsIncident, r.Severity, r.Summary, r.Rationale, r.Confidence)
}

func triageMsg(msg string) (TriageResult, error) {

	return TriageResult{}, fmt.Errorf("unsupported")
}
