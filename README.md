# flamefighterbot

Simple SRE triage bot. Filter chat message with AI and trigger only for important ones.

## How to use

1. Get OpenAI API token
2. Register bot with botfather and remember it token
3. Get telegram chat-id for monitoring flame discarding 
4. Define params in bot environment and run it

**Environment variables**

| Name               | Description                       |
|--------------------|-----------------------------------|
| TELEGRAM_BOT_TOKEN | Telegram bot token from botfather |
| OPENAI_API_KEY     | OpenAI API token                  |
| OPENAI_CHAT_MODEL  | Chat model name, ex. gpt://qwerty123/yandexgpt-lite/latest |
| OPENAI_BASE_URL    | Base url ex. `https://api.openai.com/v1`, `https://rest-assistant.api.cloud.yandex.net/v1` |