package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	dg, _ := discordgo.New("Bot " + token)

	dg.AddHandler(messageCreate)
	dg.AddHandler(onInteraction)

	dg.Open()
	defer dg.Close()

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Bot Online") })
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()

	fmt.Println("Bot attivo!")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID { return }
	args := strings.Split(m.Content, " ")

	switch args[0] {
	case "!setup-ticket":
		msg := &discordgo.MessageSend{
			Embed: &discordgo.MessageEmbed{
				Title: "🎫 SISTEMA TICKET",
				Description: "Clicca il bottone per aprire un ticket!",
				Color: 0x3498db,
			},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{Label: "Apri Supporto", Style: discordgo.PrimaryButton, CustomID: "t_gen"},
					},
				},
			},
		}
		s.ChannelMessageSendComplex(m.ChannelID, msg)
	case "!chiama-fdo":
		s.ChannelMessageSend(m.ChannelID, "🚨 **FDO RICHIESTE NELLA MAPPA!**")
	}
}

func onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent { return }
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "Creazione ticket...", Flags: 64},
	})
	s.GuildChannelCreate(i.GuildID, "ticket-"+i.Member.User.Username, discordgo.ChannelTypeGuildText)
}
