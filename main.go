package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
)

var s *discordgo.Session

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	var err error
	s, err = discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Errore creazione sessione: %v", err)
	}

	// Definizione dei comandi Slash
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "setup-ticket",
			Description: "Configura il sistema di ticket",
		},
		{
			Name:        "chiama-fdo",
			Description: "Invia una richiesta alle Forze dell'Ordine",
		},
	}

	// Gestore delle interazioni (quando l'utente usa / o clicca bottoni)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			switch i.ApplicationCommandData().Name {
			case "setup-ticket":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "📩 **Sistema Ticket Attivo**\nClicca il bottone qui sotto per aprire una segnalazione.",
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{
								Components: []discordgo.MessageComponent{
									discordgo.Button{
										Label:    "Apri Ticket",
										Style:    discordgo.PrimaryButton,
										CustomID: "open_ticket",
									},
								},
							},
						},
					},
				})
			case "chiama-fdo":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "🚨 **ALLERTA FDO**\nUna nuova richiesta di intervento è stata inviata!",
					},
				})
			}
		}

		// Gestione del bottone Ticket
		if i.Type == discordgo.InteractionMessageComponent {
			if i.MessageComponentData().CustomID == "open_ticket" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "✅ Hai aperto un ticket! Un membro dello staff ti aiuterà a breve.",
						Flags:   discordgo.MessageFlagsEphemeral, // Lo vede solo l'utente
					},
				})
			}
		}
	})

	err = s.Open()
	if err != nil {
		log.Fatalf("Errore apertura connessione: %v", err)
	}

	// Registra i comandi su Discord
	for _, v := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Impossibile creare il comando '%v': %v", v.Name, err)
		}
	}

	fmt.Println("Bot online con comandi Slash!")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	s.Close()
}
