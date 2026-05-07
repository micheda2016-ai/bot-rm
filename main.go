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
		{Name: "setup-ticket", Description: "Configura il pannello ticket con menu a tendina"},
		{Name: "chiama-fdo", Description: "Invia una notifica alla Categoria FDO"},
		{
			Name: "promozione",
			Description: "Comunica una promozione ufficiale",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "L'utente da promuovere", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "nuovo-ruolo", Description: "Il grado ottenuto", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo della promozione", Required: true},
			},
		},
		{
			Name: "retrocessione",
			Description: "Comunica una retrocessione",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "L'utente da retrocedere", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "vecchio-ruolo", Description: "Il grado perso", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo del provvedimento", Required: true},
			},
		},
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
										{Label: "Generale", Value: "Generale", Emoji: discordgo.ComponentEmoji{Name: "💡"}, Description: "Richieste generiche o info"},
										{Label: "Richiesta Piani Alti", Value: "Richiesta Piani Alti", Emoji: discordgo.ComponentEmoji{Name: "👑"}, Description: "Parlare con l'Alta Amministrazione"},
										{Label: "Segnala Agente", Value: "Segnala Agente", Emoji: discordgo.ComponentEmoji{Name: "⚠️"}, Description: "Segnalazioni su membri FDO"},
									},
								},
							},
						},
					},
				},
			})
		}

		// 2. GESTIONE SELEZIONE MENU (CREAZIONE CANALE E NOTIFICA)
		if i.Type == discordgo.InteractionMessageComponent && i.MessageComponentData().CustomID == "select_ticket" {
			category := i.MessageComponentData().Values[0]
