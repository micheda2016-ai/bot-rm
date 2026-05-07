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
	// Questo pezzo serve per far credere a Render che il bot sia un sito web
	// Così non lo spegne dopo 5 minuti (errore timeout porta)
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Bot Online!")
		})
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Printf("Server finto attivo sulla porta %s", port)
		http.ListenAndServe(":"+port, nil)
	}()

	// Recupera il Token dalle variabili di Render
	token := os.Getenv("DISCORD_TOKEN")
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Errore creazione sessione: %v", err)
	}

	// Definizione dei comandi Slash
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "setup-ticket",
			Description: "Configura il sistema di ticket con bottone",
		},
		{
			Name:        "chiama-fdo",
			Description: "Invia un'allerta per le Forze dell'Ordine",
		},
	}

	// Gestore dei comandi e dei bottoni
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// Gestione dei comandi "/"
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
						Content: "🚨 **ALLERTA FORZE DELL'ORDINE**\nUna nuova richiesta di intervento è stata registrata!",
					},
				})
			}
		}

		// Gestione del click sul bottone "Apri Ticket"
		if i.Type == discordgo.InteractionMessageComponent {
			if i.MessageComponentData().CustomID == "open_ticket" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "✅ Ticket creato con successo! Verrai contattato dallo staff.",
						Flags:   discordgo.MessageFlagsEphemeral, // Lo vede solo chi preme
					},
				})
			}
		}
	})

	// Apertura connessione
	err = s.Open()
	if err != nil {
		log.Fatalf("Errore apertura: %v", err)
	}

	// Registrazione dei comandi slash su Discord
	for _, v := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			log.Printf("Impossibile creare il comando '%v': %v", v.Name, err)
		}
	}

	fmt.Println("Bot online con comandi Slash e protezione Timeout!")
	
	// Mantieni il bot acceso
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	s.Close()
}
