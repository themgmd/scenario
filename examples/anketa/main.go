package main

import (
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/themgmd/scenario"
)

type startHandler struct {
	scenario *scenario.Scenario
}

func (h *startHandler) Handle(c telebot.Context) error {
	sceneCtx := scenario.NewContext(h.scenario, c)
	return sceneCtx.Enter("user_register_scene")
}

func (h *startHandler) RegisterScene() {
	wizard := scenario.NewWizard("user_register_scene",
		func(c *scenario.Context) (bool, error) {
			m := c.Message()

			if strings.HasPrefix(m.Text, "/") {
				return false, c.Reply("Введите ваше имя")
			}

			if m != nil && strings.TrimSpace(m.Text) != "" {
				c.Session.Data["name"] = strings.TrimSpace(m.Text)
				return true, c.Reply("введите ваше ДР")
			}
			return false, c.Reply("Введите ваше имя")
		},
		func(c *scenario.Context) (bool, error) {
			m := c.Message()

			if m != nil && strings.TrimSpace(m.Text) != "" {
				c.Session.Data["bd"] = strings.TrimSpace(m.Text)
				return true, c.Reply("Спасибо!")
			}
			return false, c.Reply("Введите ваше ДР")
		},
	)

	h.scenario.Use(wizard)
}

func main() {
	settings := telebot.Settings{
		Token: "XXX",
		Poller: &telebot.LongPoller{
			Timeout: time.Duration(10) * time.Second,
		},
	}

	bot, err := telebot.NewBot(settings)
	if err != nil {
		log.Fatal(err)
		return
	}

	scn := scenario.New(bot)
	bot.Use(scn.Middleware)

	handler := startHandler{scenario: scn}
	handler.RegisterScene()
	bot.Handle("/start", handler.Handle)

	// Fallback handlers to ensure stage middleware receives messages
	bot.Handle(telebot.OnText, func(c telebot.Context) error { return nil })
	bot.Handle(telebot.OnContact, func(c telebot.Context) error { return nil })

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("Telegram bot started")
		bot.Start()
	}()

	wg.Wait()
}
