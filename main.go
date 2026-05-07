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
	// Protezione per Render
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Bot Online!") })
		port := os.Getenv("PORT")
		if port == "" { port = "8080" }
		http.ListenAndServe(":"+port, nil)
	}()

	token := os.Getenv("DISCORD_TOKEN")
	s, err := discordgo.New("Bot " + token)
	if err != nil { log.Fatalf("Errore: %v", err) }

	// Comandi Slash
	commands := []*discordgo.ApplicationCommand{
		{Name: "setup-ticket", Description: "Configura il pannello ticket"},
		{Name: "chiama-fdo", Description: "Invia una notifica alla Categoria FDO"},
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// 1. GESTIONE COMANDO SETUP
		if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "setup-ticket" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "🎫 **PANNELLO SUPPORTO UTENTI**\nSeleziona il motivo della tua richiesta dal menu qui sotto per aprire un ticket.",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID:    "select_ticket",
									Placeholder: "Scegli una categoria...",
									Options: []discordgo.SelectMenuOption{
										{Label: "Generale", Value: "generale", Emoji: discordgo.ComponentEmoji{Name: "💡"}, Description: "Richieste generiche"},
										{Label: "Richiesta Piani Alti", Value: "piani_alti", Emoji: discordgo.ComponentEmoji{Name: "👑"}, Description: "Alta Amministrazione"},
										{Label: "Segnala Agente", Value: "segnala_agente", Emoji: discordgo.ComponentEmoji{Name: "⚠️"}, Description: "Segnalazioni FDO"},
									},
								},
							},
						},
					},
				},
			})
		}

		// 2. GESTIONE MENU A TENDINA (CREAZIONE CANALE)
		if i.Type == discordgo.InteractionMessageComponent && i.MessageComponentData().CustomID == "select_ticket" {
			category := i.MessageComponentData().Values[0]
			guildID := i.GuildID
			userID := i.Member.User.ID
			userName := i.Member.User.Username

			// Creazione canale privato
			channel, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
				Name: fmt.Sprintf("ticket-%s-%s", category, userName),
				Type: discordgo.ChannelTypeGuildText,
				PermissionOverwrites: []*discordgo.PermissionOverwrite{
					{ID: guildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel},
					{ID: userID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages},
					{ID: "1501973077263912980", Type: discordgo.PermissionOverwriteTypeRole, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages},
					{ID: "1501973204925808661", Type: discordgo.PermissionOverwriteTypeRole, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages},
				},
			})

			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: "❌ Errore permessi bot!", Flags: discordgo.MessageFlagsEphemeral},
				})
				return
			}

			// Messaggio nel nuovo canale
			pingStaff := "<@&1501973077263912980> <@&1501973204925808661>"
			s.ChannelMessageSend(channel.ID, fmt.Sprintf("👋 %s benvenuto!\nHai aperto un ticket: **%s**\n%s Riceverai assistenza a breve.", i.Member.User.Mention(), category, pingStaff))

			// Risposta all'utente
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "✅ Ticket creato: <#" + channel.ID + ">", Flags: discordgo.MessageFlagsEphemeral},
			})
		}

		// 3. COMANDO FDO
		if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "chiama-fdo" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "🚨 **CHIAMATA FDO**\nNotifica inviata alla <@&1492918778885963836>!"},
			})
		}
	})

	s.Open()
	for _, v := range commands { s.ApplicationCommandCreate(s.State.User.ID, "", v) }
	fmt.Println("Bot pronto!")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
