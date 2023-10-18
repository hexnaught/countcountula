package main

import (
	"context"
	"fmt"
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
	GuildList map[string]*Guild
}

type Guild struct {
	ID             string
	ActiveChannels map[string]*Count
}

type Count struct {
	ChannelID        string
	Count            int
	HighestCount     int
	PreviousSenderID string
}

func main() {
	log.SetLevel(log.LevelDebug)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	state := &State{
		GuildList: make(map[string]*Guild),
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

	if event.Message.Content == "!cc help" {
		event.Client().Rest().CreateMessage(event.ChannelID, discord.NewMessageCreateBuilder().SetContent("Just count! Use `!cc enable` and `!cc disable` to enable/disable the bot for specific channels.").Build())
		return
	}

	currentGuildInfo, ok := s.GuildList[event.GuildID.String()]
	if !ok {
		currentGuildInfo = &Guild{
			ID: event.GuildID.String(),
			ActiveChannels: map[string]*Count{event.ChannelID.String(): {
				ChannelID:        event.ChannelID.String(),
				Count:            0,
				HighestCount:     0,
				PreviousSenderID: "",
			}},
		}
	}
	currentChannelInfo, ok := currentGuildInfo.ActiveChannels[event.ChannelID.String()]
	if !ok {
		currentGuildInfo.ActiveChannels[event.ChannelID.String()] = &Count{
			ChannelID:        event.ChannelID.String(),
			Count:            0,
			HighestCount:     0,
			PreviousSenderID: "",
		}
	}
	if event.Message.Content == "!cc disable" {
		event.Client().Rest().CreateMessage(event.ChannelID, discord.NewMessageCreateBuilder().SetContent("Counting disabled for the channel.").Build())
		currentGuildInfo.ActiveChannels[currentChannelInfo.ChannelID].Count = -1
		s.GuildList[event.GuildID.String()] = currentGuildInfo
		return
	}
	if event.Message.Content == "!cc enable" {
		event.Client().Rest().CreateMessage(event.ChannelID, discord.NewMessageCreateBuilder().SetContent("Counting enabled for the channel, starting at 0.").Build())
		currentGuildInfo.ActiveChannels[currentChannelInfo.ChannelID].Count = 0
		s.GuildList[event.GuildID.String()] = currentGuildInfo
		return
	}
	if currentChannelInfo.Count == -1 {
		return
	}

	countGiven, err := strconv.Atoi(event.Message.Content)
	if err != nil {
		return
	}

	if countGiven != currentChannelInfo.Count+1 || event.Message.Author.ID.String() == currentChannelInfo.PreviousSenderID {
		str := "OOP! Count resetting, you're a dumbo!"

		if countGiven != currentChannelInfo.Count+1 {
			str += " You can't count!"
		}
		if event.Message.Author.ID.String() == currentChannelInfo.PreviousSenderID {
			str += " You sent the last count!"
		}

		if currentChannelInfo.HighestCount < currentChannelInfo.Count {
			str += fmt.Sprintf(" You got a new highest count of %v! Previously it was %v.", currentChannelInfo.Count, currentChannelInfo.HighestCount)
			currentChannelInfo.HighestCount = currentChannelInfo.Count
		} else {
			str += fmt.Sprintf(" Highest count reached is still %v.", currentChannelInfo.HighestCount)
		}

		event.Client().Rest().AddReaction(event.ChannelID, event.MessageID, "🚫")
		_, _ = event.Client().Rest().CreateMessage(event.ChannelID, discord.NewMessageCreateBuilder().SetContent(str).Build())
		currentChannelInfo.Count = 0
		currentChannelInfo.PreviousSenderID = ""
	} else {
		event.Client().Rest().AddReaction(event.ChannelID, event.MessageID, "✅")
		currentChannelInfo.Count += 1
		currentChannelInfo.PreviousSenderID = event.Message.Author.ID.String()
	}

	s.GuildList[event.GuildID.String()] = currentGuildInfo
	log.Debugf("State Update: %+v", s)
	log.Debugf("State Update: %+v", s.GuildList)
	for k, g := range s.GuildList {
		log.Debugf("State Update: %v, %+v", k, g)
		for k2, c := range g.ActiveChannels {
			log.Debugf("State Update: %v, %+v", k2, c)
		}
	}
}
