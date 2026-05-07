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
	// Server per Render
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Bot Online") })
		port := os.Getenv("PORT")
		if port == "" { port = "8080" }
		http.ListenAndServe(":"+port, nil)
	}()

	token := os.Getenv("DISCORD_TOKEN")
	s, err := discordgo.New("Bot " + token)
	if err != nil { log.Fatalf("Errore: %v", err) }

	// --- MODIFICA QUESTI NOMI CON I RUOLI REALI DEL TUO SERVER ---
	gradiFDO := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Allievo", Value: "Allievo"},
		{Name: "Agente", Value: "Agente"},
		{Name: "Agente Scelto", Value: "Agente Scelto"},
		{Name: "Assistente", Value: "Assistente"},
		{Name: "Vice Sovrintendente", Value: "Vice Sovrintendente"},
		{Name: "Sovrintendente", Value: "Sovrintendente"},
		{Name: "Vice Ispettore", Value: "Vice Ispettore"},
		{Name: "Ispettore", Value: "Ispettore"},
		{Name: "Commissario", Value: "Commissario"},
		{Name: "Vice Questore", Value: "Vice Questore"},
		{Name: "Questore", Value: "Questore"},
		{Name: "Comandante", Value: "Comandante"},
	}

	commands := []*discordgo.ApplicationCommand{
		{Name: "setup-ticket", Description: "Configura il pannello ticket"},
		{Name: "chiama-fdo", Description: "Invia notifica alla Categoria FDO"},
		{
			Name: "promozione",
			Description: "Annuncia una promozione",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "Utente da promuovere", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "grado", Description: "Seleziona il nuovo grado", Required: true, Choices: gradiFDO},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivazione", Required: true},
			},
		},
		{
			Name: "retrocessione",
			Description: "Annuncia una retrocessione",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "Utente da retrocedere", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "grado", Description: "Seleziona il grado assegnato", Required: true, Choices: gradiFDO},
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
				res := fmt.Sprintf("🎖️ **PROMOZIONE UFFICIALE**\n\n**Soggetto:** %s\n**Nuovo Grado:** %s\n**Motivazione:** %s", u.Mention(), data.Options[1].StringValue(), data.Options[2].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: res}})
			
			case "retrocessione":
				u := data.Options[0].UserValue(s)
				res := fmt.Sprintf("📉 **RETROCESSIONE DI GRADO**\n\n**Soggetto:** %s\n**Nuovo Grado:** %s\n**Motivazione:** %s", u.Mention(), data.Options[1].StringValue(), data.Options[2].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: res}})

			case "avvertimento":
				u := data.Options[0].UserValue(s)
				res := fmt.Sprintf("⚠️ **AVVERTIMENTO UFFICIALE**\n\n**Soggetto:** %s\n**Motivazione:** %s", u.Mention(), data.Options[1].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: res}})

			case "chiama-fdo":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: "🚨 **CHIAMATA FDO**\nNotifica inviata alla <@&1492918778885963836>!"}})

			case "setup-ticket":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4,
					Data: &discordgo.InteractionResponseData{
						Content: "🎫 **PANNELLO SUPPORTO**\nSeleziona una categoria per aprire un ticket.",
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID: "select_ticket",
									Placeholder: "Scegli categoria...",
									Options: []discordgo.SelectMenuOption{
										{Label: "Generale", Value: "Generale", Emoji: discordgo.ComponentEmoji{Name: "💡"}},
										{Label: "Piani Alti", Value: "Piani Alti", Emoji: discordgo.ComponentEmoji{Name: "👑"}},
										{Label: "Segnala Agente", Value: "Segnala Agente", Emoji: discordgo.ComponentEmoji{Name: "⚠️"}},
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
	s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
