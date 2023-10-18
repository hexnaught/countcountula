package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/disgoorg/log"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
)

const (
	TOKEN = ""
)

type State struct {
	GuildList map[string]Guild
}

type Guild struct {
	ID             string
	ActiveChannels map[string]struct{}
	Count          int
}

func main() {
	log.SetLevel(log.LevelDebug)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	state := &State{
		GuildList: make(map[string]Guild),
	}

	client, err := disgo.New(TOKEN,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuildMessages,
				gateway.IntentMessageContent,
			),
		),
		bot.WithEventListenerFunc(state.onMessageCreate),
	)
	if err != nil {
		log.Fatal("error while building disgo: ", err)
	}

	defer client.Close(context.TODO())

	if err = client.OpenGateway(context.TODO()); err != nil {
		log.Fatal("errors while connecting to gateway: ", err)
	}

	log.Info("Bot Running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
}

func (s *State) onMessageCreate(event *events.MessageCreate) {
	if event.Message.Author.Bot {
		return
	}

	countGiven, err := strconv.Atoi(event.Message.Content)
	if err != nil {
		return
	}
	currentGuildInfo, ok := s.GuildList[event.GuildID.String()]
	if !ok {
		currentGuildInfo = Guild{
			ID:             event.GuildID.String(),
			ActiveChannels: map[string]struct{}{event.ChannelID.String(): {}},
			Count:          0,
		}
	}

	if countGiven != currentGuildInfo.Count+1 {
		_, _ = event.Client().Rest().CreateMessage(event.ChannelID, discord.NewMessageCreateBuilder().SetContent("OOP! Count resetting, you're a dumbo!").Build())
		currentGuildInfo.Count = 0
	} else {
		currentGuildInfo.Count += 1
	}

	s.GuildList[event.GuildID.String()] = currentGuildInfo
	log.Infof("State Update: %+v", s)
}