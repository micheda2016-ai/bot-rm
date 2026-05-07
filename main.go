package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
)

func main() {
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Bot Online!") })
		port := os.Getenv("PORT")
		if port == "" { port = "8080" }
		http.ListenAndServe(":"+port, nil)
	}()

	token := os.Getenv("DISCORD_TOKEN")
	s, err := discordgo.New("Bot " + token)
	if err != nil { log.Fatalf("Errore: %v", err) }

	// Lista dei Gradi per il menu a scelta
	gradiFDO := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Agente", Value: "Agente"},
		{Name: "Agente Scelto", Value: "Agente Scelto"},
		{Name: "Assistente", Value: "Assistente"},
		{Name: "Sovrintendente", Value: "Sovrintendente"},
		{Name: "Ispettore", Value: "Ispettore"},
		{Name: "Commissario", Value: "Commissario"},
		{Name: "Comandante", Value: "Comandante"},
	}

	commands := []*discordgo.ApplicationCommand{
		{Name: "setup-ticket", Description: "Configura il pannello ticket"},
		{Name: "chiama-fdo", Description: "Invia una notifica alla Categoria FDO"},
		{
			Name: "promozione",
			Description: "Comunica una promozione",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "Utente da promuovere", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "grado", Description: "Nuovo grado", Required: true, Choices: gradiFDO},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivazione", Required: true},
			},
		},
		{
			Name: "retrocessione",
			Description: "Comunica una retrocessione",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "Utente da retrocedere", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "grado", Description: "Nuovo grado", Required: true, Choices: gradiFDO},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivazione", Required: true},
			},
		},
		{
			Name: "avvertimento",
			Description: "Invia un avvertimento",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "Utente da avvertire", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivazione", Required: true},
			},
		},
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			data := i.ApplicationCommandData()
			switch data.Name {
			case "promozione":
				u := data.Options[0].UserValue(s)
				msg := fmt.Sprintf("🎖️ **PROMOZIONE**\n\n**Soggetto:** %s\n**Nuovo Grado:** %s\n**Motivo:** %s", u.Mention(), data.Options[1].StringValue(), data.Options[2].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: msg}})
			
			case "retrocessione":
				u := data.Options[0].UserValue(s)
				msg := fmt.Sprintf("📉 **RETROCESSIONE**\n\n**Soggetto:** %s\n**Nuovo Grado:** %s\n**Motivo:** %s", u.Mention(), data.Options[1].StringValue(), data.Options[2].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: msg}})

			case "avvertimento":
				u := data.Options[0].UserValue(s)
				msg := fmt.Sprintf("⚠️ **AVVERTIMENTO**\n\n**Soggetto:** %s\n**Motivo:** %s", u.Mention(), data.Options[1].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: msg}})

			case "chiama-fdo":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: "🚨 **CHIAMATA FDO**\nNotifica inviata alla <@&1492918778885963836>!"}})

			case "setup-ticket":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4,
					Data: &discordgo.InteractionResponseData{
						Content: "🎫 **PANNELLO SUPPORTO**\nUsa il menu sotto per aprire un ticket.",
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID: "select_ticket",
									Placeholder: "Scegli categoria...",
									Options: []discordgo.SelectMenuOption{
										{Label: "Generale", Value: "generale", Emoji: discordgo.ComponentEmoji{Name: "💡"}},
										{Label: "Piani Alti", Value: "piani_alti", Emoji: discordgo.ComponentEmoji{Name: "👑"}},
										{Label: "Segnala Agente", Value: "segnala_agente", Emoji: discordgo.ComponentEmoji{Name: "⚠️"}},
									},
								},
							}},
						},
					},
				})
			}
		}

		if i.Type == discordgo.InteractionMessageComponent && i.MessageComponentData().CustomID == "select_ticket" {
			cat := i.MessageComponentData().Values[0]
			ch, _ := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
				Name: "ticket-" + cat + "-" + i.Member.User.Username,
				PermissionOverwrites: []*discordgo.PermissionOverwrite{
					{ID: i.GuildID, Type: 0, Deny: 1024},
					{ID: i.Member.User.ID, Type: 1, Allow: 3072},
					{ID: "1501973077263912980", Type: 0, Allow: 3072},
					{ID: "1501973204925808661", Type: 0, Allow: 3072},
				},
			})
			s.ChannelMessageSend(ch.ID, "👋 Ticket: "+cat+"\nUtente: "+i.Member.User.Mention()+"\n<@&1501973077263912980> <@&1501973204925808661>")
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: "✅ Aperto: <#"+ch.ID+">", Flags: 64}})
		}
	})

	s.Open()
	s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands) // Forza l'aggiornamento di tutti i comandi
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
